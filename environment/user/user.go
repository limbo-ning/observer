package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/datasource"
	"obsessiontech/common/ipc"
	"obsessiontech/common/util"
	"obsessiontech/environment/role"
	"obsessiontech/environment/site"
)

var Config struct {
	IsUserRegisterGlobalLockHost     bool
	UserRegisterGLobalLockHostType   string
	UserRegisterGlobalLockHost       string
	UserRegisterGlobalLockTimeoutSec time.Duration
}

func init() {
	config.GetConfig("config.yaml", &Config)
	if Config.IsUserRegisterGlobalLockHost {
		host := new(ipc.GlobalLocker)

		if err := host.Host(Config.UserRegisterGLobalLockHostType, Config.UserRegisterGlobalLockHost); err != nil {
			log.Panic("fail to host user register global lock: ", err)
		}
	}
}

func RequestRegisterLock(siteID, requestID string, lockKeys []string, action func() error) error {

	if Config.UserRegisterGlobalLockHost == "" {
		return action()
	}

	host := Config.UserRegisterGlobalLockHost
	if Config.IsUserRegisterGlobalLockHost && strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}

	return ipc.RequestGlobalLock(Config.UserRegisterGLobalLockHostType, host, siteID, requestID, lockKeys, Config.UserRegisterGlobalLockTimeoutSec*time.Second, action)
}

const (
	USER_ACTIVE   = "ACTIVE"
	USER_INACTIVE = "INACTIVE"
	USER_DELETE   = "DELETE"
)

var E_user_exists = errors.New("用户已存在")
var E_user_not_exists = errors.New("用户不存在")

type UserBrief struct {
	UserID     int                 `json:"ID"`
	Username   string              `json:"username"`
	Mobile     string              `json:"mobile"`
	Email      string              `json:"email"`
	Profile    map[string][]string `json:"profile"`
	CreateTime util.Time           `json:"createTime"`
}

type UserInfo struct {
	UserBrief
	RealName     string            `json:"realName"`
	IDentityNo   string            `json:"identityNo"`
	Organization string            `json:"organization"`
	Department   string            `json:"department"`
	Stats        map[string]string `json:"stats"`
}

type User struct {
	UserInfo
	Status        string                 `json:"status"`
	IsPasswordSet bool                   `json:"isPasswordSet"`
	Password      string                 `json:"-"`
	WechatInfo    map[string]interface{} `json:"-"`
	WechatBinding map[string]bool        `json:"wechatBinding"`
	Ext           map[string]interface{} `json:"ext"`
	LastLoginTime util.Time              `json:"lastLoginTime"`
	UpdateTime    util.Time              `json:"updateTime"`
}

const userBriefColumns = "user.id, user.mobile, user.email, user.profile, user.username, user.create_time"
const userInfoColumns = "user.id, user.mobile, user.email, user.profile, user.username, user.real_name, user.identity_no, user.organization, user.department, user.stats, user.create_time"
const userColumns = "user.id, user.status, user.profile, user.username, user.password, user.mobile, user.email, user.wechat_info, user.real_name, user.identity_no, user.organization, user.department, user.stats, user.ext, user.last_login_time, user.create_time, user.update_time"

func (u *UserBrief) scan(rows *sql.Rows) error {
	var profile string
	if err := rows.Scan(&u.UserID, &u.Mobile, &u.Email, &profile, &u.Username, &u.CreateTime); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(profile), &u.Profile); err != nil {
		return err
	}

	u.Mobile = util.Mask(u.Mobile, "*")

	return nil
}

func (u *UserInfo) scan(rows *sql.Rows) error {
	var profile, stats string
	if err := rows.Scan(&u.UserID, &u.Mobile, &u.Email, &profile, &u.Username, &u.RealName, &u.IDentityNo, &u.Organization, &u.Department, &stats, &u.CreateTime); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(profile), &u.Profile); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(stats), &u.Stats); err != nil {
		return err
	}

	return nil
}

func (u *User) scan(rows *sql.Rows) error {
	var profile, wechatinfoStr, statsStr, extStr string
	if err := rows.Scan(&u.UserID, &u.Status, &profile, &u.Username, &u.Password, &u.Mobile, &u.Email, &wechatinfoStr, &u.RealName, &u.IDentityNo, &u.Organization, &u.Department, &statsStr, &extStr, &u.LastLoginTime, &u.CreateTime, &u.UpdateTime); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(profile), &u.Profile); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(wechatinfoStr), &u.WechatInfo); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(statsStr), &u.Stats); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(extStr), &u.Ext); err != nil {
		return err
	}

	if u.Password != "" {
		u.IsPasswordSet = true
	} else {
		u.IsPasswordSet = false
	}

	u.WechatBinding = make(map[string]bool)
	for appID := range u.WechatInfo {
		u.WechatBinding[appID] = true
	}

	return nil
}

func userTable(siteID string) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_USER, true)
	if err != nil {
		return "", err
	}
	return moduleSite + "_user", nil
}

func GetUserInfo(siteID string, uid ...int) ([]*UserInfo, error) {
	table, err := userTable(siteID)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	result := make([]*UserInfo, 0)

	if len(uid) == 0 {
		return result, nil
	} else {
		if len(uid) == 1 {
			whereStmts = append(whereStmts, "user.id = ?")
			values = append(values, uid[0])
		} else {
			placeHolder := make([]string, 0)
			for _, id := range uid {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("user.id IN (%s)", strings.Join(placeHolder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s user
		WHERE
			%s
	`, userInfoColumns, table, strings.Join(whereStmts, " AND "))

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var u UserInfo
		if u.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &u)
	}

	return result, nil
}

func GetUserBrief(siteID string, uid ...int) ([]*UserBrief, error) {
	table, err := userTable(siteID)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	result := make([]*UserBrief, 0)

	if len(uid) == 0 {
		return result, nil
	} else {
		if len(uid) == 1 {
			whereStmts = append(whereStmts, "user.id = ?")
			values = append(values, uid[0])
		} else {
			placeHolder := make([]string, 0)
			for _, id := range uid {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("user.id IN (%s)", strings.Join(placeHolder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s user
		WHERE
			%s
	`, userBriefColumns, table, strings.Join(whereStmts, " AND "))

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var u UserBrief
		if u.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &u)
	}

	return result, nil
}

func GetUsers(siteID, match, q, status string, pageNo, pageSize int, orderBy string, UID ...int) ([]*User, int, error) {
	return getUsers(siteID, match, q, status, pageNo, pageSize, orderBy, "", nil, nil, UID...)
}

func GetRoleUsers(siteID, roleType, match, q, status string, pageNo, pageSize int, orderBy string, roleIDs ...int) ([]*User, int, error) {
	joinSQL, joinWhere, joinValues := role.JoinRole(siteID, "uid", "user.id", roleType, roleIDs...)

	return getUsers(siteID, match, q, status, pageNo, pageSize, orderBy, joinSQL, joinWhere, joinValues)
}

func getUsers(siteID, match, q, status string, pageNo, pageSize int, orderBy, join string, joinWhere []string, joinValues []interface{}, userIDs ...int) ([]*User, int, error) {

	table, err := userTable(siteID)
	if err != nil {
		return nil, -1, err
	}

	switch status {
	case USER_ACTIVE:
	case USER_INACTIVE:
	default:
		status = USER_ACTIVE
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if joinWhere != nil {
		whereStmts = append(whereStmts, joinWhere...)
	}
	if joinValues != nil {
		values = append(values, joinValues...)
	}

	if len(userIDs) > 0 {
		placeHolder := make([]string, 0)
		for _, id := range userIDs {
			placeHolder = append(placeHolder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("user.id IN (%s)", strings.Join(placeHolder, ",")))
		pageNo = 1
		pageSize = -1
	}

	if match != "" {
		whereStmts = append(whereStmts, "(user.username = ? OR user.mobile = ? OR user.email = ? OR user.real_name = ?)")
		values = append(values, match, match, match, match)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(user.username LIKE ? OR user.mobile LIKE ? OR user.email LIKE ? OR user.identity_no LIKE ? OR user.real_name LIKE ? OR user.department LIKE ? OR user.organization LIKE ?)")
		values = append(values, qq, qq, qq, qq, qq, qq, qq)
	}

	if status != "" {
		whereStmts = append(whereStmts, "user.status = ?")
		values = append(values, status)
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s user
		%s
	`, userColumns, table, join)
	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s user
		%s
	`, table, join)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	var total int
	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count user: ", countSQL, values, err)
		return nil, -1, err
	}

	if orderBy == "" {
		orderBy = "user.id DESC"
	}

	SQL += "\n ORDER BY " + orderBy

	if pageSize != -1 {
		if pageNo <= 0 {
			pageNo = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		SQL += " LIMIT ?, ?"
		values = append(values, (pageNo-1)*pageSize, pageSize)
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get user: ", SQL, values, err)
		return nil, -1, err
	}

	defer rows.Close()

	result := make([]*User, 0)

	for rows.Next() {
		var r User
		if err := r.scan(rows); err != nil {
			return nil, -1, err
		}
		result = append(result, &r)
	}

	return result, total, nil
}

func GetUser(siteID, uniqueColumn string, uniqueValue interface{}) (*User, error) {
	result, err := GetUserWithTxn(siteID, nil, uniqueColumn, uniqueValue, false)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetUserWithTxn(siteID string, txn *sql.Tx, uniqueColumn string, uniqueValue interface{}, forUpdate bool) (*User, error) {
	var rows *sql.Rows
	var err error

	table, err := userTable(siteID)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s user
		WHERE
			%s = ? AND user.status != ?
	`, userColumns, table, uniqueColumn)

	if forUpdate {
		sql += "\nFOR UPDATE"
	}

	log.Println("get unique user: ", sql, uniqueValue)

	if txn == nil {
		rows, err = datasource.GetConn().Query(sql, uniqueValue, USER_DELETE)
	} else {
		rows, err = txn.Query(sql, uniqueValue, USER_DELETE)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		log.Println("unique user get")
		var u User
		if err := u.scan(rows); err != nil {
			return nil, err
		}

		return &u, nil
	}

	log.Println("unique user not exists")

	return nil, E_user_not_exists
}

func (user *User) Add(siteID string, txn *sql.Tx) error {

	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	switch user.Status {
	case USER_ACTIVE:
	case USER_INACTIVE:
	default:
		user.Status = USER_ACTIVE
	}

	if user.Profile == nil {
		user.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(user.Profile)
	if user.WechatInfo == nil {
		user.WechatInfo = make(map[string]interface{})
	}
	wechatInfo, _ := json.Marshal(user.WechatInfo)
	if user.Stats == nil {
		user.Stats = make(map[string]string)
	}
	stats, _ := json.Marshal(user.Stats)
	if user.Ext == nil {
		user.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(user.Ext)

	SQL := fmt.Sprintf(`
		INSERT INTO %s
			(status, profile, username, password, mobile, email, wechat_info, real_name, identity_no, organization, department, stats, ext)
		VALUES
			(?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, table)

	values := make([]interface{}, 0)
	values = append(values, user.Status, string(profile), user.Username, user.Password, user.Mobile, user.Email, string(wechatInfo), user.RealName, user.IDentityNo, user.Organization, user.Department, string(stats), string(ext))

	var ret sql.Result

	if txn != nil {
		ret, err = txn.Exec(SQL, values...)
	} else {
		ret, err = datasource.GetConn().Exec(SQL, values...)
	}

	if err != nil {
		log.Println("error insert user: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert user: ", err)
		return err
	} else {
		user.UserID = int(id)
	}

	return nil
}

func (user *User) Activate(siteID string) error {
	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			status = ?
		WHERE
			id = ?
	`, table), USER_ACTIVE, user.UserID); err != nil {
		log.Println("error activate user: ", err)
		return err
	}

	return nil
}

func (user *User) Deactivate(siteID string) error {
	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			status = ?
		WHERE
			id = ?
	`, table), USER_INACTIVE, user.UserID); err != nil {
		log.Println("error deactive user: ", err)
		return err
	}

	return nil
}

func (user *User) Update(siteID string, txn *sql.Tx) error {

	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	switch user.Status {
	case USER_ACTIVE:
	case USER_INACTIVE:
	case USER_DELETE:
	default:
		user.Status = USER_ACTIVE
	}

	if user.Profile == nil {
		user.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(user.Profile)
	if user.WechatInfo == nil {
		user.WechatInfo = make(map[string]interface{})
	}
	wechatInfo, _ := json.Marshal(user.WechatInfo)
	if user.Stats == nil {
		user.Stats = make(map[string]string)
	}
	stats, _ := json.Marshal(user.Stats)
	if user.Ext == nil {
		user.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(user.Ext)

	SQL := fmt.Sprintf(`
		UPDATE
			%s
		SET
			status = ?, profile = ?, username = ?, password = ?, mobile = ?, email = ?, wechat_info = ?, real_name = ?, identity_no = ?, organization = ?, department = ?, stats = ?, ext = ?
		WHERE
			id = ?
	`, table)

	values := make([]interface{}, 0)
	values = append(values, user.Status, string(profile), user.Username, user.Password, user.Mobile, user.Email, string(wechatInfo), user.RealName, user.IDentityNo, user.Organization, user.Department, string(stats), string(ext), user.UserID)

	var ret sql.Result

	if txn != nil {
		ret, err = txn.Exec(SQL, values...)
	} else {
		ret, err = datasource.GetConn().Exec(SQL, values...)
	}

	if err != nil {
		log.Println("error update user: ", err)
		return err
	}

	if _, err := ret.RowsAffected(); err != nil {
		log.Println("error get roles inffected: ", err)
		return err
	}

	return nil
}

func (user *User) UpdateLoginTime(siteID string) error {

	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			last_login_time = Now()
		WHERE
			id = ?
	`, table), user.UserID); err != nil {
		log.Println("error update user logined time: ", err)
		return err
	}

	return nil
}

func (user *User) Delete(siteID string) error {

	table, err := userTable(siteID)
	if err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
			UPDATE
				%s
			SET
				status = ?
			WHERE
				id = ?
		`, table), USER_DELETE, user.UserID); err != nil {
			log.Println("error delete user: ", err)
			panic(err)
		}

		if err := role.ExpireUserRole(siteID, txn, user.UserID); err != nil {
			panic(err)
		}
	})
}
