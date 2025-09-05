package monitor

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
)

const (
	FLAG_EFFECTIVE = 1 << iota

	FLAG_TRANSMISSION

	FLAG_NORMAL

	FLAG_OVERPROOF
	FLAG_DATA_INVARIANCE
	FLAG_DATA_FLOW_ZERO

	FLAG_TOP_LIMIT
	FLAG_LOW_LIMIT

	FLAG_MANUAL

	FLAG_PRIMARY_POLLUTANT

	FLAG_PUSH
)

func init() {
	dataprocess.Register("flag", func() dataprocess.IDataProcessor { return new(flagProcessor) })
}

type Flag struct {
	Flag        string `json:"flag"`
	Name        string `json:"name"`
	Bits        int    `json:"bits"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

func CheckFlag(flagBit int, bits int) bool {
	return bits&flagBit == flagBit
}

func GetFlagByBit(siteID string, flagBit int) (*Flag, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	for _, f := range m.Flags {
		if CheckFlag(flagBit, f.Bits) {
			return f, nil
		}
	}

	return nil, nil
}

func GetFlag(siteID, flag string) (*Flag, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	for _, f := range m.Flags {
		if f.Flag == flag {
			return f, nil
		}
	}

	return nil, nil
}

func ChangeFlag(siteID string, entry data.IData, toFlag string, UID int) error {

	if entry.GetFlag() == toFlag {
		return nil
	}

	prev, err := GetFlag(siteID, entry.GetFlag())
	if err != nil {
		return err
	}

	if prev != nil {
		if err := data.ModifyValue(entry, data.FLAG_BIT, data.ClearFlagBit(entry.GetFlagBit(), prev.Bits), UID); err != nil {
			return err
		}
	}

	if toFlag != "" {
		to, err := GetFlag(siteID, toFlag)
		if err != nil {
			return err
		}

		if to == nil {
			return fmt.Errorf("标记未定义:%s", toFlag)
		}

		if err := data.ModifyValue(entry, data.FLAG_BIT, data.SetFlagBit(entry.GetFlagBit(), to.Bits), UID); err != nil {
			return err
		}
	}

	if err := data.ModifyValue(entry, data.FLAG, toFlag, -1); err != nil {
		return err
	}

	return nil
}

type flagProcessor struct {
	dataprocess.BaseDataProcessor
}

func (p *flagProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) (bool, error) {

	log.Println("flag processor: ", entry.GetStationID(), entry.GetMonitorID(), entry.GetDataTime())

	m, err := GetModule(siteID)
	if err != nil {
		return false, err
	}

	normal, err := GetFlagByBit(siteID, FLAG_NORMAL)
	if err != nil {
		return false, err
	}

	var value float64

	if absctractData, ok := entry.(data.IInterval); ok {
		value = absctractData.GetAvg()
	} else if absctractData, ok := entry.(data.IRealTime); ok {
		value = absctractData.GetRtd()
	} else {
		log.Printf("error data with unknown interface: %t", entry)
		return false, errors.New("未知数据接口类型")
	}

	if CheckFlag(FLAG_MANUAL, entry.GetFlagBit()) {
		log.Println("manual bit set")
		return false, nil
	}

	for _, f := range m.Flags {
		flagLimit := GetFlagLimit(siteID, entry.GetStationID(), entry.GetMonitorID(), f.Flag)
		if flagLimit != nil {
			if CheckFlag(FLAG_DATA_INVARIANCE, f.Bits) {
				if len(flagLimit.regionSegments) == 1 && len(flagLimit.regionSegments[0]) == 1 {
					invarianceHour := flagLimit.regionSegments[0][0].Value
					if isDi, err := isDataInvariance(siteID, txn, entry, invarianceHour); err != nil {
						log.Println("error check data invariance: ", err)
						return false, err
					} else if isDi {
						if err := ChangeFlag(siteID, entry, f.Flag, -1); err != nil {
							return false, err
						}

						return false, nil
					}
				}
			} else if flagLimit.IsInRegion(value) {
				if err := ChangeFlag(siteID, entry, f.Flag, -1); err != nil {
					return false, err
				}

				return false, nil
			}
		}
	}

	if normal != nil {
		if err := ChangeFlag(siteID, entry, normal.Flag, -1); err != nil {
			return false, err
		}
	}

	return false, nil
}

func isDataInvariance(siteID string, txn *sql.Tx, entry data.IData, invarianceHour float64) (bool, error) {

	if invarianceHour <= 0 {
		return false, nil
	}

	var fields string

	//仅对小时和日数据做数据不变检查 太消耗资源
	switch entry.GetDataType() {
	case data.REAL_TIME:
		return false, nil
	case data.MINUTELY:
		return false, nil
	default:
		fields = strings.Join([]string{data.AVG, data.MIN, data.MAX}, ",")
	}

	SQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s
		WHERE
			%s = ? AND %s = ? AND %s > ? AND %s <= ?
		GROUP BY
			%s
	`, data.TableName(siteID, entry.GetDataType()), data.STATION_ID, data.MONITOR_ID, data.DATA_TIME, data.DATA_TIME, fields)

	rows, err := txn.Query(SQL, entry.GetStationID(), entry.GetMonitorID(), time.Time(entry.GetDataTime()).Add(time.Minute*time.Duration(invarianceHour*60)*-1), time.Time(entry.GetDataTime()))
	if err != nil {
		log.Println("error get data invariance count: ", SQL, err)
		return false, err
	}
	defer rows.Close()

	count := 0
	var entryCount int
	for rows.Next() {
		count++
		if count > 1 {
			return false, nil
		}
		if err := rows.Scan(&entryCount); err != nil {
			log.Println("error get data invariance entry count: ", SQL, err)
			return false, err
		}
	}

	return entryCount > 1, nil
}
