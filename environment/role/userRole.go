package role

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

var e_user_role_series_exists = errors.New("用户已有系列角色")

type UserRole struct {
	UID     int `json:"UID"`
	RoleID  int `json:"roleID"`
	Expires int `json:"expires"`
}

const userRoleColumns = "userRole.uid,userRole.role_id,userRole.expires"

func userRoleTableName(siteID string) string {
	return siteID + "_userrole"
}

func (u *UserRole) scan(rows *sql.Rows) error {
	return rows.Scan(&u.UID, &u.RoleID, &u.Expires)
}

func (u *UserRole) add(siteID string, txn *sql.Tx) error {
	if _, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(uid,role_id,expires)
		VALUES
			(?,?,?)
		ON DUPLICATE KEY UPDATE expires=VALUES(expires)
	`, userRoleTableName(siteID)), u.UID, u.RoleID, u.Expires); err != nil {
		log.Println("error add user role: ", err)
		return err
	}
	return nil
}

func (u *UserRole) delete(siteID string, txn *sql.Tx) error {
	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM 
			%s
		WHERE
			uid=? AND role_id=?
	`, userRoleTableName(siteID)), u.UID, u.RoleID); err != nil {
		log.Println("error delete user role: ", err)
		return err
	}
	return nil
}

func checkGrantUserRolePrivilege(roleID int, actionAuth authority.ActionAuthSet) error {
	for _, a := range actionAuth {
		if a.UID <= 0 {
			continue
		}
		action := strings.TrimPrefix(a.Action, MODULE_ROLE+"#")

		switch action {
		case ACTION_GRANT_ALL:
		case ACTION_GRANT_ROLE:
			grantedRoleID, _ := strconv.Atoi(a.RoleType)
			if grantedRoleID != -1 && roleID != grantedRoleID {
				continue
			}
		default:
			continue
		}

		return nil
	}

	return errors.New("无权限")
}

func GetUserRoleExpiresWithTxn(siteID string, txn *sql.Tx, uid int, roleID ...int) (map[int]*UserRole, error) {

	result := make(map[int]*UserRole)

	if uid <= 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]any, 0)

	whereStmts = append(whereStmts, "userRole.uid = ?")
	values = append(values, uid)

	if len(roleID) > 0 {
		if len(roleID) == 1 {
			whereStmts = append(whereStmts, "userRole.role_id = ?")
			values = append(values, roleID[0])
		} else {
			placeholder := make([]string, len(roleID))
			for i, id := range roleID {
				placeholder[i] = "?"
				values = append(values, id)
			}

			whereStmts = append(whereStmts, fmt.Sprintf("userRole.role_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s userRole
		WHERE
			%s
		FOR UPDATE
	`, userRoleColumns, userRoleTableName(siteID), strings.Join(whereStmts, " AND "))

	rows, err := txn.Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var ur UserRole
		if err := ur.scan(rows); err != nil {
			return nil, err
		}

		result[ur.RoleID] = &ur
	}

	return result, nil
}

func GrantUserRole(siteID string, uid, roleID, expires int, actionAuth authority.ActionAuthSet) error {

	if err := checkGrantUserRolePrivilege(roleID, actionAuth); err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if _, err := bindUserRole(siteID, txn, uid, roleID, expires); err != nil {
			panic(err)
		}
	})
}

func WithdrawUserRole(siteID string, uid, roleID int, actionAuth authority.ActionAuthSet) error {

	if err := checkGrantUserRolePrivilege(roleID, actionAuth); err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := unbindUserRole(siteID, txn, uid, roleID); err != nil {
			panic(err)
		}
	})
}

func bindUserRole(siteID string, txn *sql.Tx, uid, roleID, expires int) (*UserRole, error) {

	if uid < 0 || roleID == 0 {
		return nil, errors.New("缺少参数")
	}

	origins, err := checkUserExistSeriesRole(siteID, txn, uid, roleID)
	if err != nil {
		log.Println("error check user exist series role: ", err)
		return nil, err
	}

	var origin *UserRole
	for _, ur := range origins {
		origin = ur
		if err := unbindUserRole(siteID, txn, ur.UID, ur.RoleID); err != nil {
			return nil, err
		}
	}

	ur := new(UserRole)
	ur.UID = uid
	ur.RoleID = roleID
	ur.Expires = expires
	if err := ur.add(siteID, txn); err != nil {
		return nil, err
	}

	return origin, nil
}

func unbindUserRole(siteID string, txn *sql.Tx, uid, roleID int) error {

	ur := new(UserRole)
	ur.UID = uid
	ur.RoleID = roleID
	if err := ur.delete(siteID, txn); err != nil {
		return err
	}

	return nil
}

func checkUserExistSeriesRole(siteID string, txn *sql.Tx, uid int, roleID int) ([]*UserRole, error) {
	role, err := GetRoleWithTxn(siteID, txn, roleID, false)
	if err != nil {
		return nil, err
	}

	if role.Series == "" {
		return nil, nil
	}

	sm, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	rs := sm.GetRoleSeries(role.Series)
	if rs == nil || !rs.IsUnique {
		return nil, nil
	}

	roleTable, err := roleTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s userRole
		JOIN
			%s role
		ON
			role.id = userRole.role_id AND role.series = ?
		WHERE
			userRole.uid = ?
	`, userRoleColumns, userRoleTableName(siteID), roleTable)

	rows, err := txn.Query(SQL, role.Series, uid)
	if err != nil {
		log.Println("error get user exists role series: ", SQL, role.Series, uid, err)
		return nil, err
	}
	defer rows.Close()

	result := make([]*UserRole, 0)
	for rows.Next() {
		var ur UserRole
		if err := ur.scan(rows); err != nil {
			log.Println("error scan user exists role series: ", err)
			continue
		}

		if ur.RoleID != roleID {
			result = append(result, &ur)
		}
	}

	return result, nil
}

func JoinRole(siteID, joinField, joinOn string, roleType string, roleID ...int) (string, []string, []any) {
	var join string
	joinWhere := make([]string, 0)
	joinValues := make([]interface{}, 0)

	joinTable := "userRole"

	if roleType != "" {
		joinTable += "_" + roleType
	}

	for _, rid := range roleID {
		joinTable += fmt.Sprintf("_%d", rid)
	}

	joinWhere = append(joinWhere, fmt.Sprintf("(%s.expires = 0 or %s.expires > UNIX_TIMESTAMP())", joinTable, joinTable))

	if len(roleID) == 1 {
		joinWhere = append(joinWhere, fmt.Sprintf("%s.role_id = ?", joinTable))
		joinValues = append(joinValues, roleID[0])
	} else if len(roleID) > 1 {
		placeholder := make([]string, 0)
		for _, id := range roleID {
			placeholder = append(placeholder, "?")
			joinValues = append(joinValues, id)
		}
		joinWhere = append(joinWhere, fmt.Sprintf("%s.role_id IN (%s)", joinTable, strings.Join(placeholder, ",")))
	}

	join = fmt.Sprintf(`
		JOIN
			%s %s
		ON
			%s.%s = %s
	`, userRoleTableName(siteID), joinTable, joinTable, joinField, joinOn)

	return join, joinWhere, joinValues
}

func ExpireUserRole(siteID string, txn *sql.Tx, uid int, roleID ...int) error {

	userRoles, err := GetUserRoleExpiresWithTxn(siteID, txn, uid, roleID...)
	if err != nil {
		return err
	}

	expires := int(time.Now().Unix())

	for _, ur := range userRoles {
		ur.Expires = expires
		if err := ur.add(siteID, txn); err != nil {
			return err
		}
	}

	return nil
}
