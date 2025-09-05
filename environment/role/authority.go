package role

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/site"
)

type RoleAuthority struct {
	ID       int    `json:"ID,omitempty"`
	RoleID   int    `json:"roleID,omitempty"`
	ModuleID string `json:"moduleID"`
	Action   string `json:"action"`
	RoleType string `json:"roleType"`
}

const roleAuthColumns = "roleAuth.id,roleAuth.role_id,roleAuth.module_id,roleAuth.action,roleAuth.role_type"

func roleAuthorityTableName(siteID string) string {
	return siteID + "_roleauthority"
}

func (a *RoleAuthority) scan(rows *sql.Rows, appendix ...interface{}) error {

	dest := make([]interface{}, 0)
	dest = append(dest, &a.ID, &a.RoleID, &a.ModuleID, &a.Action, &a.RoleType)
	dest = append(dest, appendix...)

	if err := rows.Scan(dest...); err != nil {
		return err
	}
	return nil
}

func (a *RoleAuthority) add(siteID string) error {

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(role_id,module_id,action,role_type)
		VALUES
			(?,?,?,?)
	`, roleAuthorityTableName(siteID)), a.RoleID, a.ModuleID, a.Action, a.RoleType); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		a.ID = int(id)
	}

	return nil
}

func (a *RoleAuthority) delete(siteID string) error {
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			role_id=? AND module_id=? AND action=? AND role_type=?
	`, roleAuthorityTableName(siteID)), a.RoleID, a.ModuleID, a.Action, a.RoleType); err != nil {
		return err
	}
	return nil
}

func GetAuthorityActions(siteID, moduleID, session, clientAgent string, uid int, action ...string) (authority.ActionAuthSet, error) {

	result := make(authority.ActionAuthSet, 0)

	if len(action) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	joinSiteModule, joinWhere, joinValues := site.JoinSiteModuleAuth(siteID, "roleAuth", "module_id")
	whereStmts = append(whereStmts, joinWhere...)
	values = append(values, joinValues...)

	whereStmts = append(whereStmts, "userRole.uid = ?")
	values = append(values, uid)

	whereStmts = append(whereStmts, "(userRole.expires = 0 OR userRole.expires > UNIX_TIMESTAMP())")

	whereStmts = append(whereStmts, "roleAuth.module_id = ?")
	values = append(values, moduleID)

	if len(action) == 1 {
		whereStmts = append(whereStmts, "roleAuth.action = ?")
		values = append(values, action[0])
	} else {
		placeholder := make([]string, 0)
		for _, a := range action {
			placeholder = append(placeholder, "?")
			values = append(values, a)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("roleAuth.action IN (%s)", strings.Join(placeholder, ",")))
	}

	roleTable, err := roleTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			roleAuth.action, roleAuth.role_type, roleAuth.role_id, role.series
		FROM
			%s roleAuth
		JOIN
			%s userRole
		ON
			roleAuth.role_id = userRole.role_id
		JOIN
			%s role
		ON 
			roleAuth.role_id = role.id
		%s
		WHERE
			%s
	`, roleAuthorityTableName(siteID), userRoleTableName(siteID), roleTable, joinSiteModule, strings.Join(whereStmts, " AND "))

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get auth action: ", SQL, values, err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var a authority.ActionAuth
		if err := rows.Scan(&a.Action, &a.RoleType, &a.RoleID, &a.RoleSeries); err != nil {
			return nil, err
		}
		a.UID = uid
		a.Session = session
		a.ClientAgent = clientAgent

		result = append(result, &a)
	}

	return result, nil
}

func getRoleAuthoritiesWithTxn(siteID string, txn *sql.Tx, moduleID []string, roleID ...int) (map[int][]*RoleAuthority, error) {
	result := make(map[int][]*RoleAuthority)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	joinAuth, joinAuthWhere, joinAuthValues := site.JoinSiteModuleAuth(siteID, "roleAuth", "module_id")

	whereStmts = append(whereStmts, joinAuthWhere...)
	values = append(values, joinAuthValues...)

	if len(moduleID) > 0 {
		if len(moduleID) == 1 {
			whereStmts = append(whereStmts, "roleAuth.module_id = ?")
			values = append(values, moduleID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range moduleID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("roleAuth.module_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(roleID) > 0 {
		if len(roleID) == 1 {
			whereStmts = append(whereStmts, "roleAuth.role_id = ?")
			values = append(values, roleID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range roleID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("roleAuth.role_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s roleAuth
		%s
	`, roleAuthColumns, roleAuthorityTableName(siteID), joinAuth)

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	var rows *sql.Rows
	var err error
	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		rows, err = txn.Query(SQL, values...)
	}
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var a RoleAuthority

		if err := a.scan(rows); err != nil {
			return nil, err
		}

		if _, exists := result[a.RoleID]; !exists {
			result[a.RoleID] = make([]*RoleAuthority, 0)
		}

		result[a.RoleID] = append(result[a.RoleID], &a)
	}

	return result, nil
}

func CheckUserRoleAuthority(siteID string, moduleID, action, roleType string, uid ...int) (map[int]bool, error) {
	result := make(map[int]bool)

	if len(uid) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	joinAuth, joinAuthWhere, joinAuthValues := site.JoinSiteModuleAuth(siteID, "roleAuth", "module_id")
	whereStmts = append(whereStmts, joinAuthWhere...)
	values = append(values, joinAuthValues...)

	joinRole, joinRoleWhere, joinRoleValues := JoinRole(siteID, "role_id", "roleAuth.role_id", "")
	whereStmts = append(whereStmts, joinRoleWhere...)
	values = append(values, joinRoleValues...)

	if len(uid) == 1 {
		whereStmts = append(whereStmts, "userRole.uid = ?")
		values = append(values, uid[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range uid {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("userRole.uid IN (%s)", strings.Join(placeholder, ",")))
	}

	whereStmts = append(whereStmts, "roleAuth.module_id = ?", "roleAuth.action = ?")
	values = append(values, moduleID, action)

	if roleType != "" {
		whereStmts = append(whereStmts, "roleAuth.role_type = ?")
		values = append(values, roleType)
	}

	SQL := fmt.Sprintf(`
		SELECT
			userRole.uid
		FROM
			%s roleAuth
		%s
		%s
	`, roleAuthorityTableName(siteID), joinAuth, joinRole)

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	log.Println("check user role authority: ", SQL, values)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}

		result[uid] = true
	}

	return result, nil
}

func GetRoleAuthority(siteID string, moduleID []string, roleID ...int) (map[int][]*RoleAuthority, error) {
	return getRoleAuthoritiesWithTxn(siteID, nil, moduleID, roleID...)
}

func GetUserRoleAuthority(siteID string, uid ...int) (map[int]map[int][]*RoleAuthority, map[int]map[int]int, error) {

	roleAuths := make(map[int]map[int][]*RoleAuthority)
	expiring := make(map[int]map[int]int)

	if len(uid) == 0 {
		return roleAuths, expiring, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	joinAuth, joinAuthWhere, joinAuthValues := site.JoinSiteModuleAuth(siteID, "roleAuth", "module_id")
	whereStmts = append(whereStmts, joinAuthWhere...)
	values = append(values, joinAuthValues...)

	joinRole, joinRoleWhere, joinRoleValues := JoinRole(siteID, "role_id", "roleAuth.role_id", "")
	whereStmts = append(whereStmts, joinRoleWhere...)
	values = append(values, joinRoleValues...)

	if len(uid) > 0 {
		if len(uid) == 1 {
			whereStmts = append(whereStmts, "userRole.uid = ?")
			values = append(values, uid[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range uid {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("userRole.uid IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s, userRole.uid, userRole.expires
		FROM
			%s roleAuth
		%s
		%s
	`, roleAuthColumns, roleAuthorityTableName(siteID), joinAuth, joinRole)

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	log.Println("get user role: ", SQL, values)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	tmp := make(map[int]map[int]map[int]*RoleAuthority)

	for rows.Next() {
		var UID, expires int
		var a RoleAuthority
		if err := a.scan(rows, &UID, &expires); err != nil {
			return nil, nil, err
		}

		roleMap, exists := tmp[UID]
		if !exists {
			roleMap = make(map[int]map[int]*RoleAuthority)
			tmp[UID] = roleMap
		}

		authMap, exists := roleMap[a.RoleID]
		if !exists {
			authMap = make(map[int]*RoleAuthority)
			roleMap[a.RoleID] = authMap
		}

		if exist, exists := authMap[a.ID]; !exists || (exist.RoleType == "" && a.RoleType != "") {
			authMap[a.ID] = &a
		}

		if expires > 0 {
			if _, exists := expiring[UID]; !exists {
				expiring[UID] = make(map[int]int)
			}
			expiring[UID][a.RoleID] = expires
		}
	}

	for UID, roleMap := range tmp {
		resultRoleMap := make(map[int][]*RoleAuthority)

		for roleID, mapping := range roleMap {
			list := make([]*RoleAuthority, 0)
			for _, m := range mapping {
				list = append(list, m)
			}
			resultRoleMap[roleID] = list
		}

		roleAuths[UID] = resultRoleMap
	}

	return roleAuths, expiring, nil
}

func checkGrantPrivilege(siteID string, roleID int, moduleID, action string, actionAuth authority.ActionAuthSet) error {

	for _, a := range actionAuth {
		if a.UID <= 0 {
			continue
		}
		action := strings.TrimPrefix(a.Action, MODULE_ROLE+"#")

		switch action {
		case ACTION_GRANT_ALL:
		case ACTION_GRANT_AUTHORITY:
			grantedRoleID, _ := strconv.Atoi(a.RoleType)
			if grantedRoleID != -1 && roleID != grantedRoleID {
				continue
			}

			grantedActionAuth, err := GetAuthorityActions(siteID, moduleID, "", "", a.UID, action)
			if err != nil {
				return err
			}

			if len(grantedActionAuth) == 0 {
				log.Println("info grant privilege fail as module action to grant is not part of user auth: ", a.UID, moduleID, action)
				continue
			}

		default:
			continue
		}

		return nil
	}

	return errors.New("无权限")
}

func GetGrantableAuthorities(siteID string, uid int, actionAuth authority.ActionAuthSet) ([]*RoleAuthority, error) {

	result := make([]*RoleAuthority, 0)

	getAll := false
	getOwn := false

	for _, a := range actionAuth {
		if a.UID <= 0 {
			continue
		}
		action := strings.TrimPrefix(a.Action, MODULE_ROLE+"#")
		switch action {
		case ACTION_GRANT_ALL:
			getAll = true
		case ACTION_GRANT_AUTHORITY:
			getOwn = true
		}
	}

	if getAll {

		log.Println("grant all")

		whereStmts := make([]string, 0)
		values := make([]interface{}, 0)

		joinAuth, joinWhere, joinValues := site.JoinSiteModuleAuth(siteID, "module", "id")

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)

		SQL := fmt.Sprintf(`
			SELECT
				%s
			FROM
				%s module
			%s
		`, site.ModuleColumns, site.ModuleTableName, joinAuth)

		if len(whereStmts) > 0 {
			SQL += "WHERE " + strings.Join(whereStmts, " AND ")
		}

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var m site.Module

			if err := m.Scan(rows); err != nil {
				return nil, err
			}

			for _, a := range m.Action {
				auth := new(RoleAuthority)
				auth.ModuleID = m.ModuleID
				auth.Action = a.Action

				result = append(result, auth)
			}
		}
	} else if getOwn {

		log.Println("grant own")

		roleAuthoritys, _, err := GetUserRoleAuthority(siteID, uid)
		if err != nil {
			return nil, err
		}

		for _, authList := range roleAuthoritys[uid] {
			result = append(result, authList...)
		}
	}

	return result, nil
}

func GrantRoleModule(siteID string, roleID int, moduleID, action, roleType string, actionAuth authority.ActionAuthSet) error {

	if err := checkGrantPrivilege(siteID, roleID, moduleID, action, actionAuth); err != nil {
		return err
	}

	auth := new(RoleAuthority)
	auth.ModuleID = moduleID
	auth.RoleID = roleID
	auth.Action = action
	auth.RoleType = roleType

	return auth.add(siteID)
}

func WithdrawRoleModule(siteID string, roleID int, moduleID, action, roleType string, actionAuth authority.ActionAuthSet) error {

	if err := checkGrantPrivilege(siteID, roleID, moduleID, action, actionAuth); err != nil {
		return err
	}

	auth := new(RoleAuthority)
	auth.ModuleID = moduleID
	auth.RoleID = roleID
	auth.Action = action
	auth.RoleType = roleType

	return auth.delete(siteID)
}

type RoleAuthorityTemplate struct {
	ID   int    `json:"ID"`
	Name string `json:"name"`
}

func roleAuthorityTemplateTableName(siteID string) string {
	return siteID + "_roleauthoritytemplate"
}

const roleAuthorityTemplateColumn = "roleAuthorityTemplate.id,roleAuthorityTemplate.name"

func (m *RoleAuthorityTemplate) scan(rows *sql.Rows) error {
	if err := rows.Scan(&m.ID, &m.Name); err != nil {
		return err
	}
	return nil
}

func (m *RoleAuthorityTemplate) Add(siteID string) error {

	if m.Name == "" {
		return errors.New("请命名模版")
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name)
		VALUES
			(?)
	`, roleAuthorityTemplateTableName(siteID)), m.Name); err != nil {
		log.Println("error insert monitor code template: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor code template: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *RoleAuthorityTemplate) Update(siteID string) error {
	if m.Name == "" {
		return errors.New("请命名模版")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?
		WHERE
			id=?
	`, roleAuthorityTemplateTableName(siteID)), m.Name, m.ID); err != nil {
		log.Println("error update monitor code template: ", err)
		return err
	}

	return nil
}
func (m *RoleAuthorityTemplate) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, roleAuthorityTemplateTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor code template: ", err)
		return err
	}

	return nil
}

func GetRoleAuthorityTemplates(siteID string) ([]*RoleAuthorityTemplate, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s roleAuthorityTemplate
	`, roleAuthorityTemplateColumn, roleAuthorityTemplateTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor code template: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*RoleAuthorityTemplate, 0)

	for rows.Next() {
		var s RoleAuthorityTemplate
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor code template: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}
