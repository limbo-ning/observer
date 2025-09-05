package module

import (
	"database/sql"
	"encoding/json"
	"log"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_CATEGORY = "category"

	ACTION_VIEW      = "view"
	ACTION_VIEW_TYPE = "view_type"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"
)

type CategoryModule struct {
	CategoryTypes []*CategoryType `json:"categoryTypes"`
}

type CategoryType struct {
	Source      string `json:"source"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func GetModule(siteID string) (*CategoryModule, error) {
	var m *CategoryModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_CATEGORY, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal category module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal category module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *CategoryModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_CATEGORY, true)
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
