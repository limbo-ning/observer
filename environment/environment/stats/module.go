package stats

import (
	"database/sql"
	"encoding/json"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_STATS = "environment_stats"
)

type StatsModule struct {
	TransEffectRateVersion int `json:"transEffectRateVersion"`
}

func GetModule(siteID string, flags ...bool) (*StatsModule, error) {
	var m *StatsModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_STATS, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal environment stats module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal environment stats module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *StatsModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_STATS, true)
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
