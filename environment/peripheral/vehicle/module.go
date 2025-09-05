package vehicle

import (
	"database/sql"
	"encoding/json"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const MODULE_VEHICLE = "peripheral_vehicle"

const (
	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"

	ACTION_VIEW = "view"
	ACTION_EDIT = "edit"
)

var AdminActions = map[string]string{
	ACTION_VIEW: ACTION_ADMIN_VIEW,
	ACTION_EDIT: ACTION_ADMIN_EDIT,
}

type VehicleModule struct {
	Types []*VehicleType `json:"types"`
}

type VehicleType struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	ExtInfos []string `json:"extInfos"`
}

func GetModule(siteID string) (*VehicleModule, error) {
	var m *VehicleModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_VEHICLE, false)
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

func (m *VehicleModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_VEHICLE, true)
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
