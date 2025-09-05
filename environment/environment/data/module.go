package data

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_DATA = "environment_data"
)

type DataModule struct {
	MonitorField            string        `json:"monitorField"`
	RotationBatchSize       int           `json:"rotationBatchSize"`
	Rotations               []*Rotation   `json:"rotations"`
	ArchiveActiveTimeoutMin time.Duration `json:"archiveActiveTimeoutMin"`
}

func GetModule(siteID string) (*DataModule, error) {
	var m *DataModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_DATA, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal data module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal data module: ", err)
		return nil, err
	}

	if m.RotationBatchSize <= 0 {
		m.RotationBatchSize = 10000
	}

	switch m.MonitorField {
	case MONITOR_ID:
	case MONITOR_CODE_ID:
	default:
		m.MonitorField = MONITOR_ID
	}

	return m, nil
}

func (m *DataModule) Save(siteID string) error {

	checked := make(map[string]byte)

	if m.MonitorField != "" {
		switch m.MonitorField {
		case MONITOR_ID:
		case MONITOR_CODE_ID:
		default:
			return errors.New("不支持的数据因子字段")
		}
	}

	for _, r := range m.Rotations {

		switch r.DataType {
		case REAL_TIME:
		case MINUTELY:
		case HOURLY:
		case DAILY:
		default:
			return e_invalid_data_type
		}

		if _, exists := checked[r.DataType]; exists {
			return errors.New("重复的数据类型")
		}

		switch r.Active {
		case rotation_year:
		case rotation_month:
		case rotation_quarter:
		case rotation_week:
		default:
			return errors.New("周期不正确")
		}

		switch r.Archive {
		case rotation_year:
		case rotation_month:
		case rotation_quarter:
		case rotation_week:
		default:
			return errors.New("周期不正确")
		}

		checked[r.DataType] = 1
	}

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_DATA, true)
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
