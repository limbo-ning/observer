package role

import (
	"database/sql"
	"encoding/json"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_ROLE = "role"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"

	ACTION_GRANT_ALL       = "grant_all"
	ACTION_GRANT_AUTHORITY = "grant_authority"
	ACTION_GRANT_ROLE      = "grant_role"

	ACTION_ADMIN_VIEW_USERROLE = "admin_view_userrole"
)

type RoleModule struct {
	Series []*RoleSeries `json:"series,omitempty"`
}

type RoleSeries struct {
	Series         string `json:"series"`
	Name           string `json:"name"`
	IsUnique       bool   `json:"isUnique"`
	AuthTemplateID int    `json:"templateID,omitempty"`
}

func (sm *RoleModule) GetRoleSeries(series string) *RoleSeries {
	for _, s := range sm.Series {
		if s.Series == series {
			return s
		}
	}

	return nil
}

func GetModule(siteID string, flags ...bool) (*RoleModule, error) {
	var m *RoleModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_ROLE, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal role module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal role module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *RoleModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_ROLE, true)
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
