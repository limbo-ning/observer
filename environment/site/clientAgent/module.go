package clientAgent

import (
	"database/sql"
	"encoding/json"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/common/excel"
	"obsessiontech/environment/site"
)

const (
	MODULE_CLIENTAGENT = "clientagent"

	ACTION_VIEW = "view"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"
)

type ClientAgentModule struct {
	SettingTypeKeys map[string][]string `json:"settingTypeKeys"`
	ExcelUploaders  []*excel.Uploader   `json:"excelUploaders"`
}

func GetModule(siteID string) (*ClientAgentModule, error) {
	var m *ClientAgentModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_CLIENTAGENT, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal clientagent module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal clientagent module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *ClientAgentModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_CLIENTAGENT, true)
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
