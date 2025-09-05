package user

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
	"obsessiontech/environment/site/initialization"
)

const (
	MODULE_USER = "user"

	ACTION_EDIT = "edit"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"

	ACTION_ADMIN_VIEW_MODULE = "admin_view_module"
	ACTION_ADMIN_EDIT_MODULE = "admin_edit_module"
)

func init() {
	initialization.Register(MODULE_USER, []string{"user"})
}

type UserModule struct {
	PostRegisterMission []int         `json:"postRegister,omitempty"`
	PostLoginMission    []int         `json:"postLogin,omitempty"`
	LoginExpireMin      time.Duration `json:"loginExpireMin,omitempty"`
	Referers            []string      `json:"referers,omitempty"`
}

func GetUserModule(siteID string) (*UserModule, error) {
	var userModule *UserModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_USER, true)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal user module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &userModule); err != nil {
		log.Println("error unmarshal user module: ", err)
		return nil, err
	}

	return userModule, nil
}

func GetUserModuleWithTxn(siteID string, txn *sql.Tx, forUpdate bool) (*UserModule, error) {
	sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_USER, forUpdate)
	if err != nil {
		return nil, err
	}

	var result UserModule

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal user module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &result); err != nil {
		log.Println("error unmarshal user module: ", err)
		return nil, err
	}

	return &result, nil
}

func (m *UserModule) Save(siteID string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_USER, true)
		if err != nil {
			panic(err)
		}

		paramByte, _ := json.Marshal(&m)
		json.Unmarshal(paramByte, &sm.Param)

		if err := sm.Save(siteID, txn); err != nil {
			panic(err)
		}
	})
}
