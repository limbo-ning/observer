package engine

import (
	"database/sql"
	"encoding/json"
	"errors"

	"obsessiontech/common/random"
	"obsessiontech/environment/user"
)

const ONETIME_AUTH_MODULE = "user_auth_onetime"
const ONETIME_AUTH = "onetime"

var e_token_fail = errors.New("凭证已失效")

func init() {
	register(ONETIME_AUTH, ONETIME_AUTH_MODULE, func() IAuth {
		return &OneTimeAuth{}
	})
}

type OneTimeAuth struct {
}

type OneTimeAuthAuthParam struct {
	UsernameColumn string `json:"usernameColumn"`
	Username       string `json:"username"`
	Token          string `json:"token"`
}

func (a *OneTimeAuth) Validate() error {
	return nil
}

func (a *OneTimeAuth) CheckExists(siteID, requestID string, txn *sql.Tx, toCheck *user.User) error {
	return nil
}

func (a *OneTimeAuth) Tip() map[string]any {
	return make(map[string]any)
}

func (p *OneTimeAuth) Register(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	return nil, nil, errors.New("不支持的注册方式")
}

func (p *OneTimeAuth) Login(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	var param OneTimeAuthAuthParam

	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}
	if param.Token == "" || param.Username == "" {
		return nil, nil, e_token_fail
	}

	switch param.UsernameColumn {
	case "username":
	case "mobile":
	default:
		param.UsernameColumn = "username"
	}

	existUser, err := user.GetUserWithTxn(siteID, txn, param.UsernameColumn, param.Username, true)
	if err != nil {
		if err != user.E_user_not_exists {
			return nil, nil, err
		}
		return nil, nil, e_token_fail
	}

	if token, exists := existUser.Ext[ONETIME_AUTH]; exists {
		if token == param.Token {
			delete(existUser.Ext, ONETIME_AUTH)
			existUser.Update(siteID, txn)
			return existUser, nil, nil
		}
	}

	return nil, nil, e_token_fail
}

func (p *OneTimeAuth) Bind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	if existUser.Ext == nil {
		existUser.Ext = make(map[string]interface{})
	}

	existUser.Ext[ONETIME_AUTH] = random.GenerateNonce(12)
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["token"] = existUser.Ext[ONETIME_AUTH]

	return result, nil
}

func (p *OneTimeAuth) UnBind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {

	delete(existUser.Ext, ONETIME_AUTH)
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	return nil, nil
}
