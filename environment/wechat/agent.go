package wechat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/relation"
	"obsessiontech/wechat/util"
)

var e_not_exist = errors.New("微信授权不存在")
var e_not_authorized = errors.New("微信授权未成功")
var e_expired = errors.New("微信授权已过期,请重新授权")

const AGENT_INIT = "INIT"
const AGENT_AUTHORIZED = "AUTHORIZED"
const AGENT_CANCELED = "CANCELED"
const AGENT_EXPIRED = "EXPIRED"

type WechatAgent struct {
	AppID                  string      `json:"ID"`
	Type                   string      `json:"type"`
	Status                 string      `json:"status"`
	AuthorizerAccessToken  string      `json:"-"`
	AuthorizerRefreshToken string      `json:"-"`
	ExpireTime             time.Time   `json:"-"`
	AppInfo                interface{} `json:"appInfo"`
}

func (a *WechatAgent) scan(rows *sql.Rows) error {
	var appInfo string
	if err := rows.Scan(&a.AppID, &a.Type, &a.Status, &a.AuthorizerAccessToken, &a.AuthorizerRefreshToken, &a.ExpireTime, &appInfo); err != nil {
		return err
	}

	return json.Unmarshal([]byte(appInfo), &a.AppInfo)
}

func GetAgentAccessToken(appID string) (string, error) {

	agent, err := GetAgent(appID)
	if err != nil {
		return "", err
	}

	if agent.Status == AGENT_CANCELED || agent.Status == AGENT_INIT {
		return "", e_not_authorized
	}

	if agent.Status == AGENT_EXPIRED {
		return "", e_expired
	}

	if agent.ExpireTime.Before(time.Now()) {
		if err := agent.RefreshAuthorization(); err != nil {
			return "", err
		}
	}

	return agent.AuthorizerAccessToken, nil
}

func Grant(appID, redirectURL string) (pcLink string, wechatLink string, err error) {
	return util.CreateAuthorizationLink(redirectURL, appID)
}

func BindSiteAgent(siteID, appID string) (*WechatAgent, error) {
	a, err := GetAgent(appID)
	if err != nil {
		if err == e_not_exist {
			a = &WechatAgent{
				AppID: appID,
			}
			if err := a.Add(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if exists, err := relation.ExistRelations("c", "site", "wechat_agent", "", []string{siteID}, []string{appID}); err != nil {
		return nil, err
	} else if len(exists) == 0 {
		r := relation.Relation[string, string]{
			A:   "site",
			AID: &siteID,
			B:   "wechat_agent",
			BID: &appID,
		}

		if err := datasource.Txn(func(txn *sql.Tx) {
			if err := r.Add("c", txn); err != nil {
				panic(err)
			}
		}); err != nil {
			return nil, err
		}
	}

	return a, nil
}

func UnbindSiteAgent(siteID, appID string) error {
	_, err := GetAgent(appID)
	if err != nil {
		return err
	}

	r := relation.Relation[string, string]{
		A:   "site",
		AID: &siteID,
		B:   "wechat_agent",
		BID: &appID,
	}

	if err := datasource.Txn(func(txn *sql.Tx) {
		if err := r.Delete("c", txn); err != nil {
			panic(err)
		}
	}); err != nil {
		return err
	}

	return nil
}

func Authorize(siteID, authorizationCode string) error {
	ret, err := util.GetAuthorizerAccessCode(authorizationCode)
	if err != nil {
		return err
	}

	log.Printf("[%s] wechat authorizing get authorizer access token: %+v", siteID, *ret)

	a, err := GetAgent(ret.AuthorizationInfo.AuthorizationAppID)
	if err != nil {
		if err == e_not_exist {
			a = &WechatAgent{
				AppID: ret.AuthorizationInfo.AuthorizationAppID,
			}
			a.AuthorizerAccessToken = ret.AuthorizationInfo.AuthorizationAccessToken
			a.AuthorizerRefreshToken = ret.AuthorizationInfo.AuthorizerRrefreshToken
			a.ExpireTime = time.Now().Add(time.Second * ret.AuthorizationInfo.ExpiresIn)

			a.Status = AGENT_AUTHORIZED
			if err := a.Add(); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		a.AuthorizerAccessToken = ret.AuthorizationInfo.AuthorizationAccessToken
		a.AuthorizerRefreshToken = ret.AuthorizationInfo.AuthorizerRrefreshToken
		a.ExpireTime = time.Now().Add(time.Second * ret.AuthorizationInfo.ExpiresIn)

		a.Status = AGENT_AUTHORIZED

		if err := a.Update(); err != nil {
			return err
		}
	}

	if siteID != "" {
		if err := a.bind(siteID); err != nil {
			return err
		}
	}

	go func() {
		a.RefreshInfo()
	}()

	return nil
}

func GetAgentList(siteID, appID, appType, status, q string, pageNo, pageSize int, orderBy string) ([]*WechatAgent, int, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	var join string
	var err error
	if siteID != "" {
		joinSQL, _, joinWhere, joinValues, err := relation.JoinSQL("c", "site", "wechat_agent", "", "wechat_agent", siteID)
		if err != nil {
			return nil, 0, err
		}
		join += "\n" + joinSQL
		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if appID != "" {
		whereStmts = append(whereStmts, "wechat_agent.id = ?")
		values = append(values, appID)
	}

	if appType != "" {
		whereStmts = append(whereStmts, "wechat_agent.type = ?")
		values = append(values, appType)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(wechat_agent.id LIKE ? OR wechat_agent.app_info LIKE ?)")
		values = append(values, qq, qq)
	}

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			c_wechat_agent as wechat_agent
		%s
	`, join)

	SQL := fmt.Sprintf(`
		SELECT
			wechat_agent.id, wechat_agent.type, wechat_agent.status, wechat_agent.access_token, wechat_agent.refresh_token, wechat_agent.expire_time, wechat_agent.app_info
		FROM
			c_wechat_agent as wechat_agent
		%s
	`, join)

	if len(whereStmts) > 0 {
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	result := make([]*WechatAgent, 0)
	total := 0

	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count agent: ", countSQL, err)
		return nil, 0, err
	}

	if orderBy == "" {
		orderBy = "wechat_agent.id desc"
	}

	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	SQL += "\nORDER BY " + orderBy + " LIMIT ?, ?"
	values = append(values, pageSize*(pageNo-1), pageSize)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get agent: ", SQL, err)
		return nil, 0, err
	}

	defer rows.Close()

	for rows.Next() {
		var a WechatAgent
		if err := a.scan(rows); err != nil {
			return nil, 0, err
		}
		result = append(result, &a)
	}

	return result, total, nil

}

func GetSiteAgents(siteID string) ([]*WechatAgent, error) {

	join, _, joinWhere, joinValues, err := relation.JoinSQL("c", "site", "wechat_agent", "", "wechat_agent", siteID)
	if err != nil {
		return nil, err
	}

	var where string
	if len(joinWhere) > 0 {
		where = "WHERE " + strings.Join(joinWhere, " AND ")
	}

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			wechat_agent.id, wechat_agent.type, wechat_agent.status, wechat_agent.access_token, wechat_agent.refresh_token, wechat_agent.expire_time, wechat_agent.app_info
		FROM
			c_wechat_agent as wechat_agent
		%s
		%s
	`, join, where), joinValues...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]*WechatAgent, 0)
	for rows.Next() {
		var a WechatAgent
		if err := a.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &a)
	}

	return result, nil
}

func GetAgent(appID string) (*WechatAgent, error) {
	rows, err := datasource.GetConn().Query(`
		SELECT
			id, type, status, access_token, refresh_token, expire_time, app_info
		FROM
			c_wechat_agent
		WHERE
			id = ?
	`, appID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var a WechatAgent
		a.scan(rows)

		return &a, nil
	}

	return nil, e_not_exist
}

func (a *WechatAgent) bind(siteID string) error {
	if exists, err := relation.ExistRelations("c", "site", "wechat_agent", "", []string{siteID}, []string{a.AppID}); err != nil {
		log.Println("error get binding: ", err)
		return err
	} else if len(exists) == 0 {
		r := relation.Relation[string, string]{
			A:   "site",
			AID: &siteID,
			B:   "wechat_agent",
			BID: &a.AppID,
		}

		if err := datasource.Txn(func(txn *sql.Tx) {
			if err := r.Add("c", txn); err != nil {
				panic(err)
			}
		}); err != nil {
			return err
		}
	} else {
		log.Println("binding exists", exists)
	}
	return nil
}

func (a *WechatAgent) RefreshInfo() error {
	if a.Type == util.WECHAT_APP_OPEN {
		info, err := util.GetAuthorizerOpenInfo(a.AppID)
		if err != nil {
			log.Println("error get authorizer open info: ", err)
			return err
		}
		a.AppInfo = info

	} else if a.Type == util.WECHAT_APP_MINIAPP {
		info, err := util.GetAuthorizerMiniAppInfo(a.AppID)
		if err != nil {
			log.Println("error get authorizer mini app info: ", err)
			return err
		}
		a.AppInfo = info
	} else {
		log.Println("unknown wechat app type. appID:", a.AppID)
		info, err := util.GetAuthorizerMiniAppInfo(a.AppID)
		if err != nil {
			log.Println("error get authorizer mini app info: ", err)
			return err
		}
		if info.AuthorizorInfo.MiniProgramInfo == nil || len(info.AuthorizorInfo.MiniProgramInfo) == 0 {
			a.Type = util.WECHAT_APP_OPEN
			info, err := util.GetAuthorizerOpenInfo(a.AppID)
			if err != nil {
				log.Println("error get authorizer open info: ", err)
				return err
			}
			a.AppInfo = info
		} else {
			a.Type = util.WECHAT_APP_MINIAPP
			a.AppInfo = info
		}
		log.Printf("wechat app [%s] type is %s", a.AppID, a.Type)
	}
	return a.Update()
}

func (a *WechatAgent) RefreshAuthorization() error {
	if a.AuthorizerRefreshToken == "" {
		return e_not_authorized
	}

	if a.ExpireTime.After(time.Now()) {
		log.Println("refresh authorization: no need to refresh. not expired. ", a.AppID, a.ExpireTime.Sub(time.Now()).Seconds())
		return nil
	}

	ret, err := util.RefreshAuthorizerAccessCode(a.AppID, a.AuthorizerRefreshToken)
	if err != nil {
		return err
	}

	if ret.AuthorizationAccessToken != "" {
		a.AuthorizerAccessToken = ret.AuthorizationAccessToken
	} else {
		log.Println("error refresh authorization, no access token: ", a.AppID, ret)
	}
	if ret.AuthorizerRrefreshToken != "" {
		a.AuthorizerRefreshToken = ret.AuthorizerRrefreshToken
		a.ExpireTime = time.Now().Add(time.Second * time.Duration(ret.ExpiresIn))
	} else {
		log.Println("error refresh authorization, no refresh token: ", a.AppID, ret)
	}

	return a.Update()
}

func (a *WechatAgent) Add() error {
	if a.AppID == "" {
		return errors.New("需要微信公众账号ID[AppID]")
	}
	if a.Status == "" {
		a.Status = AGENT_INIT
	}

	if a.ExpireTime.IsZero() {
		a.ExpireTime = time.Now()
	}

	if a.AppInfo == nil {
		a.AppInfo = make(map[string]interface{})
	}

	appInfo, err := json.Marshal(a.AppInfo)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(`
		INSERT INTO	c_wechat_agent
			(id, type, status, access_token, refresh_token, expire_time, app_info)
		VALUES
			(?,?,?,?,?,?,?)
	`, a.AppID, a.Type, a.Status, a.AuthorizerAccessToken, a.AuthorizerRefreshToken, a.ExpireTime, string(appInfo)); err != nil {
		return err
	}

	return nil
}

func (a *WechatAgent) Update() error {

	if a.AppInfo == nil {
		a.AppInfo = make(map[string]interface{})
	}

	appInfo, err := json.Marshal(a.AppInfo)
	if err != nil {
		return err
	}

	log.Println("wechat agent update status: ", a.AppID, a.Status)

	if _, err := datasource.GetConn().Exec(`
		UPDATE
			c_wechat_agent
		SET 
			type=?, status=?, access_token=?, refresh_token=?, expire_time=?, app_info=?
		WHERE
			id = ?
	`, a.Type, a.Status, a.AuthorizerAccessToken, a.AuthorizerRefreshToken, a.ExpireTime, string(appInfo), a.AppID); err != nil {
		return err
	}

	return nil
}

func (a *WechatAgent) Delete() error {

	if _, err := datasource.GetConn().Exec(`
		DELETE FROM
			c_wechat_agent
		WHERE
			id = ?
	`, a.AppID); err != nil {
		return err
	}

	return nil
}
