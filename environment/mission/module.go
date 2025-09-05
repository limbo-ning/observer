package mission

import (
	"database/sql"
	"encoding/json"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_MISSION = "mission"

	ACTION_VIEW     = "view"
	ACTION_EDIT     = "edit"
	ACTION_COMPLETE = "complete"
	ACTION_REVIEW   = "review"

	ACTION_ADMIN_VIEW     = "admin_view"
	ACTION_ADMIN_EDIT     = "admin_edit"
	ACTION_ADMIN_COMPLETE = "admin_complete"
	ACTION_ADMIN_REVIEW   = "admin_review"
)

var AdminActions = map[string]string{
	ACTION_VIEW:     ACTION_ADMIN_VIEW,
	ACTION_EDIT:     ACTION_ADMIN_EDIT,
	ACTION_COMPLETE: ACTION_ADMIN_COMPLETE,
	ACTION_REVIEW:   ACTION_ADMIN_REVIEW,
}

type MissionModule struct {
	Types []*MissionType `json:"types"`
}

type MissionType struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Template *Mission `json:"template,omitempty"`
}

func GetModule(siteID string, flags ...bool) (*MissionModule, error) {
	var m *MissionModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_MISSION, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal mission module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal mission module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *MissionModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_MISSION, true)
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
