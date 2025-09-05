package surveillance

import (
	"database/sql"
	"encoding/json"
	"log"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_SURVEILLANCE = "peripheral_surveillance"

	ACTION_ADMIN_EDIT = "admin_edit"
	ACTION_ADMIN_VIEW = "admin_view"

	ACTION_VIEW = "view"
)

type SurveillanceModule struct {
	Ezviz *EzvizConfig `json:"ezviz"`
}

func GetModule(siteID string) (*SurveillanceModule, error) {
	var m *SurveillanceModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_SURVEILLANCE, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal surveillance module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal surveillance module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *SurveillanceModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_SURVEILLANCE, true)
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
