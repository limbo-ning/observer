package environment

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/environment/protocol"
	"obsessiontech/environment/site"
)

const (
	MODULE_ENVIRONMENT = "environment"

	ACTION_VIEW = "view"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"
)

type EnvironmentModule struct {
	Protocols                 []*Protocol   `json:"protocols"`
	StationStatusCacheTimeMin time.Duration `json:"stationStatusCacheTimeMin,omitempty"`

	Extra map[string]interface{} `json:"extra"`
}

type Protocol struct {
	Name                string                 `json:"name"`
	Protocol            string                 `json:"protocol"`
	OfflineDelayMin     time.Duration          `json:"offlineDelayMin"`
	OfflineCountdownMin time.Duration          `json:"offlineCountdownMin"`
	Extra               map[string]interface{} `json:"extra,omitempty"`
}

func GetModule(siteID string, flags ...bool) (*EnvironmentModule, error) {
	var m *EnvironmentModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_ENVIRONMENT, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal environment module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal environment module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *EnvironmentModule) Save(siteID string) error {

	for _, p := range m.Protocols {
		if protocol.GetProtocol(p.Protocol) == nil {
			return fmt.Errorf("不支持的协议：【%s】", p.Protocol)
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_ENVIRONMENT, true)
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
