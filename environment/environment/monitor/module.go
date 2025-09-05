package monitor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_MONITOR = "environment_monitor"
)

type MonitorModule struct {
	Flags                      []*Flag            `json:"flags"`
	EffectiveIntervalThreshold map[string]float64 `json:"effectiveIntervalThreshold"`
}

func GetModule(siteID string, flags ...bool) (*MonitorModule, error) {
	var m *MonitorModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_MONITOR, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal environment monitor module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal environment monitor module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *MonitorModule) Save(siteID string) error {

	flags := make(map[string]*Flag)
	flagBits := make(map[int]bool)

	checkBit := func(flagBit int, f *Flag, canDuplicate bool) error {
		if CheckFlag(flagBit, f.Bits) {
			if !canDuplicate {
				if flagBits[flagBit] {
					return errors.New("重复的标记效力: " + f.Flag)
				}
			}
			flagBits[flagBit] = true
		}

		return nil
	}

	for _, f := range m.Flags {
		if flags[f.Flag] != nil {
			return errors.New("重复的标记: " + f.Flag)
		}

		if err := checkBit(FLAG_NORMAL, f, false); err != nil {
			return err
		}
		if err := checkBit(FLAG_OVERPROOF, f, true); err != nil {
			return err
		}
		flags[f.Flag] = f
	}

	for _, fb := range []int{FLAG_NORMAL, FLAG_OVERPROOF} {
		if !flagBits[fb] {
			return errors.New("必需标记不齐全")
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_MONITOR, true)
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
