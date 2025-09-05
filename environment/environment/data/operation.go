package data

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
)

const (
	ID = "id"

	STATION_ID      = "station_id"
	MONITOR_ID      = "monitor_id"
	MONITOR_CODE_ID = "monitor_code_id"
	DATA_TIME       = "data_time"
	ORIGIN_DATA     = "origin_data"

	RTD = "rtd"
	AVG = "avg"
	MIN = "min"
	MAX = "max"
	COU = "cou"

	FLAG     = "flag"
	FLAG_BIT = "flag_bit"
	REVIEWED = "reviewed"
)

var e_invalid_data_type = errors.New("数据类型不正确")
var e_invalid_data_interface = errors.New("数据接口未实现")
var e_invalid_data_value = errors.New("数据与类型不吻合")
var E_data_exists = errors.New("数据已存在")

var updateColumn = []string{FLAG, FLAG_BIT, ORIGIN_DATA}
var IntervalColumn = []string{AVG, MIN, MAX, COU}

func SelectColumn(siteID string) []string {
	result := make([]string, 0)
	m, err := GetModule(siteID)
	if err != nil {
		log.Println("error get module: ", err)
		return result
	}

	return []string{ID, STATION_ID, m.MonitorField, DATA_TIME, FLAG, FLAG_BIT}
}

func uniqueWrapping(siteID string, d IData) (columns []string, values []interface{}) {
	m, err := GetModule(siteID)
	if err != nil {
		log.Println("error get module: ", err)
		return
	}

	columns = append(columns, m.MonitorField, STATION_ID, DATA_TIME)
	switch m.MonitorField {
	case MONITOR_ID:
		values = append(values, d.GetMonitorID())
	case MONITOR_CODE_ID:
		values = append(values, d.GetMonitorCodeID())
	}

	values = append(values, d.GetStationID(), time.Time(d.GetDataTime()))
	return
}

func insertWrapping(siteID string, d IData) (columns []string, valueColumns []string, values []interface{}) {

	m, err := GetModule(siteID)
	if err != nil {
		log.Println("error get module: ", err)
		return
	}

	columns = append(columns, m.MonitorField, STATION_ID, DATA_TIME, FLAG, FLAG_BIT, ORIGIN_DATA)
	switch m.MonitorField {
	case MONITOR_ID:
		values = append(values, d.GetMonitorID())
	case MONITOR_CODE_ID:
		values = append(values, d.GetMonitorCodeID())
	}
	values = append(values, d.GetStationID(), time.Time(d.GetDataTime()), d.GetFlag(), d.GetFlagBit())
	var originData []byte

	d.RLockOriginData()
	defer d.RUnlockOriginData()

	if d.GetOriginData() != nil && len(d.GetOriginData()) > 0 {
		originData, _ = json.Marshal(d.GetOriginData())
	}
	values = append(values, string(originData))

	if dd, ok := d.(IRealTime); ok {
		columns = append(columns, RTD)
		valueColumns = append(valueColumns, RTD)
		values = append(values, dd.GetRtd())
	} else if dd, ok := d.(IInterval); ok {
		columns = append(columns, IntervalColumn...)
		valueColumns = append(valueColumns, IntervalColumn...)
		values = append(values, dd.GetAvg(), dd.GetMin(), dd.GetMax(), dd.GetCou())
	}

	return
}

func Scan(siteID string, rows *sql.Rows, d IData, columns []string) (e error) {

	dest := make([]interface{}, 0)
	for _, c := range columns {
		switch c {
		case ID:
			var id int
			defer func() {
				d.SetID(id)
			}()
			dest = append(dest, &id)
		case STATION_ID:
			var id int
			defer func() {
				d.SetStationID(id)
			}()
			dest = append(dest, &id)
		case MONITOR_ID:
			var id int
			defer func() {
				d.SetMonitorID(id)
			}()
			dest = append(dest, &id)
		case MONITOR_CODE_ID:
			var id int
			defer func() {
				d.SetMonitorCodeID(id)
			}()
			dest = append(dest, &id)
		case DATA_TIME:
			var t util.Time
			defer func() {
				d.SetDataTime(t)
			}()
			dest = append(dest, &t)
		case FLAG:
			var f string
			defer func() {
				d.SetFlag(f)
			}()
			dest = append(dest, &f)
		case FLAG_BIT:
			var b int
			defer func() {
				d.SetFlagBit(b)
			}()
			dest = append(dest, &b)
		case RTD:
			if _, ok := d.(IRealTime); !ok {
				return e_invalid_data_interface
			}
			var v float64
			defer func() {
				d.(IRealTime).SetRtd(v)
			}()
			dest = append(dest, &v)
		case AVG:
			if _, ok := d.(IInterval); !ok {
				return e_invalid_data_interface
			}
			var v float64
			defer func() {
				d.(IInterval).SetAvg(v)
			}()
			dest = append(dest, &v)
		case MIN:
			if _, ok := d.(IInterval); !ok {
				return e_invalid_data_interface
			}
			var v float64
			defer func() {
				d.(IInterval).SetMin(v)
			}()
			dest = append(dest, &v)
		case MAX:
			if _, ok := d.(IInterval); !ok {
				return e_invalid_data_interface
			}
			var v float64
			defer func() {
				d.(IInterval).SetMax(v)
			}()
			dest = append(dest, &v)
		case COU:
			if _, ok := d.(IInterval); !ok {
				return e_invalid_data_interface
			}
			var v float64
			defer func() {
				d.(IInterval).SetCou(v)
			}()
			dest = append(dest, &v)
		case REVIEWED:
			if _, ok := d.(IReview); !ok {
				return e_invalid_data_interface
			}
			var i int
			defer func() {
				if i > 0 {
					d.(IReview).SetReviewed(true)
				} else {
					d.(IReview).SetReviewed(false)
				}
			}()
			dest = append(dest, &i)
		case ORIGIN_DATA:
			var originData string
			defer func() {
				if originData != "" {
					d.LockOriginData()
					defer d.UnLockOriginData()

					var origin map[string]interface{}
					if err := json.Unmarshal([]byte(originData), &origin); err != nil {
						e = err
					} else {
						d.SetOriginData(origin)
					}
				}
			}()
			dest = append(dest, &originData)
		}
	}

	return rows.Scan(dest...)
}

func Add(siteID string, d IData) error {

	columns, _, values := insertWrapping(siteID, d)

	placeholder := make([]string, len(columns))
	for i := range columns {
		placeholder[i] = "?"
	}

	table := TableName(siteID, d.GetDataType())

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(%s)
		VALUES
			(%s)
	`, table, strings.Join(columns, ","), strings.Join(placeholder, ",")), values...); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return E_data_exists
		}
		log.Println("error insert data: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert data: ", err)
		return err
	} else {
		d.SetID(int(id))
	}

	return nil
}

func AddUpdate(siteID string, d IData) error {
	columns, valueColumns, values := insertWrapping(siteID, d)

	placeholder := make([]string, len(columns))
	for i := range columns {
		placeholder[i] = "?"
	}

	var updates = []string{"update_time = Now()", "origin_data = VALUES(origin_data)"}
	for _, v := range valueColumns {
		updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", v, v))
	}

	if _, ok := d.(IReview); ok {
		updates = append(updates, fmt.Sprintf("%s = ?", REVIEWED))
		values = append(values, 0)
	}

	table := TableName(siteID, d.GetDataType())

	SQL := fmt.Sprintf(`
		INSERT INTO %s
			(%s)
		VALUES
			(%s)
		ON DUPLICATE KEY UPDATE
			%s
	`, table, strings.Join(columns, ","), strings.Join(placeholder, ","), strings.Join(updates, ","))

	if ret, err := datasource.GetConn().Exec(SQL, values...); err != nil {
		log.Println("error insert data: ", SQL, values, err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert data: ", err)
		return err
	} else {
		d.SetID(int(id))
	}

	return nil
}

func Update(siteID string, d IData, field ...string) error {
	return update(siteID, nil, d, field...)
}

func UpdateWithTxn(siteID string, txn *sql.Tx, d IData, field ...string) error {
	if txn == nil {
		return errors.New("txn nil")
	}
	return update(siteID, txn, d, field...)
}

func update(siteID string, txn *sql.Tx, d IData, field ...string) error {

	updates := make([]string, 0)
	values := make([]interface{}, 0)

	var columns []string

	if len(field) > 0 {
		columns = make([]string, 0)
		for _, f := range field {
			columns = append(columns, f)

			switch f {
			case RTD:
				if rtd, ok := d.(IRealTime); ok {
					values = append(values, rtd.GetRtd())
				} else {
					return e_invalid_data_interface
				}
			case AVG:
				if interval, ok := d.(IInterval); ok {
					values = append(values, interval.GetAvg())
				} else {
					return e_invalid_data_interface
				}
			case MIN:
				if interval, ok := d.(IInterval); ok {
					values = append(values, interval.GetMin())
				} else {
					return e_invalid_data_interface
				}
			case MAX:
				if interval, ok := d.(IInterval); ok {
					values = append(values, interval.GetMax())
				} else {
					return e_invalid_data_interface
				}
			case COU:
				if interval, ok := d.(IInterval); ok {
					values = append(values, interval.GetCou())
				} else {
					return e_invalid_data_interface
				}
			case FLAG:
				values = append(values, d.GetFlag())
			case FLAG_BIT:
				values = append(values, d.GetFlagBit())
			case REVIEWED:
				if reviewed, ok := d.(IReview); ok {
					columns = append(columns, REVIEWED)
					if reviewed.GetReviewed() {
						values = append(values, 1)
					} else {
						values = append(values, 0)
					}
				} else {
					return e_invalid_data_interface
				}
			}
		}
		columns = append(columns, ORIGIN_DATA)

		var originData []byte
		d.RLockOriginData()
		defer d.RUnlockOriginData()
		if d.GetOriginData() != nil && len(d.GetOriginData()) > 0 {
			originData, _ = json.Marshal(d.GetOriginData())
		}
		values = append(values, string(originData))
	} else {

		columns = updateColumn
		values = append(values, d.GetFlag(), d.GetFlagBit())
		var originData []byte
		d.RLockOriginData()
		defer d.RUnlockOriginData()
		if d.GetOriginData() != nil && len(d.GetOriginData()) > 0 {
			originData, _ = json.Marshal(d.GetOriginData())
		}
		values = append(values, string(originData))

		if dd, ok := d.(IRealTime); ok {
			columns = append(columns, RTD)
			values = append(values, dd.GetRtd())
		} else if dd, ok := d.(IInterval); ok {
			columns = append(columns, IntervalColumn...)
			values = append(values, dd.GetAvg(), dd.GetMin(), dd.GetMax(), dd.GetCou())
		} else {
			return e_invalid_data_interface
		}

		if reviewed, ok := d.(IReview); ok {
			columns = append(columns, REVIEWED)
			if reviewed.GetReviewed() {
				values = append(values, 1)
			} else {
				values = append(values, 0)
			}
		}
	}

	for _, v := range columns {
		updates = append(updates, fmt.Sprintf("%s = ?", v))
	}

	table := TableName(siteID, d.GetDataType())

	whereColumns, whereValues := uniqueWrapping(siteID, d)
	whereStmts := make([]string, 0)

	for _, c := range whereColumns {
		whereStmts = append(whereStmts, c+"=?")
	}

	values = append(values, whereValues...)

	SQL := fmt.Sprintf(`
		UPDATE
			%s
		SET
			%s
		WHERE
			%s
	`, table, strings.Join(updates, ","), strings.Join(whereStmts, " AND "))

	var err error
	if txn != nil {
		_, err = txn.Exec(SQL, values...)
	} else {
		_, err = datasource.GetConn().Exec(SQL, values...)
	}

	if err != nil {
		log.Println("error update data: ", err)
		return err
	}

	return nil
}

func Delete(siteID string, d IData) error {

	table := TableName(siteID, d.GetDataType())

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
		 	id = ?
	`, table), d.GetID()); err != nil {
		log.Println("error delete data: ", err)
		return err
	}

	return nil
}

func RestoreValue(d IData, field ...string) bool {
	restored := false

	d.RLockOriginData()
	defer d.RUnlockOriginData()

	if len(field) == 0 {
		if rtd, ok := d.(IRealTime); ok {
			if origin, exists := d.GetOriginData()[RTD]; exists {
				rtd.SetRtd(origin.(float64))
				restored = true
			}
		} else if interval, ok := d.(IInterval); ok {
			if origin, exists := d.GetOriginData()[AVG]; exists {
				interval.SetAvg(origin.(float64))
				restored = true
			}
			if origin, exists := d.GetOriginData()[MIN]; exists {
				interval.SetMin(origin.(float64))
				restored = true
			}
			if origin, exists := d.GetOriginData()[MAX]; exists {
				interval.SetMax(origin.(float64))
				restored = true
			}
			if origin, exists := d.GetOriginData()[COU]; exists {
				interval.SetCou(origin.(float64))
				restored = true
			}
		}

		if origin, exists := d.GetOriginData()[FLAG]; exists {
			d.SetFlag(origin.(string))
			restored = true
		}
		if origin, exists := d.GetOriginData()[FLAG_BIT]; exists {
			d.SetFlagBit(origin.(int))
			restored = true
		}
	} else {
		for _, f := range field {
			switch f {
			case RTD:
				rtd, ok := d.(IRealTime)
				if !ok {
					continue
				}
				if origin, exists := d.GetOriginData()[RTD]; exists {
					rtd.SetRtd(origin.(float64))
					restored = true
				}
			case AVG:
				interval, ok := d.(IInterval)
				if !ok {
					continue
				}
				if origin, exists := d.GetOriginData()[AVG]; exists {
					interval.SetAvg(origin.(float64))
					restored = true
				}
			case MIN:
				interval, ok := d.(IInterval)
				if !ok {
					continue
				}
				if origin, exists := d.GetOriginData()[MIN]; exists {
					interval.SetMin(origin.(float64))
					restored = true
				}
			case MAX:
				interval, ok := d.(IInterval)
				if !ok {
					continue
				}
				if origin, exists := d.GetOriginData()[MAX]; exists {
					interval.SetMax(origin.(float64))
					restored = true
				}
			case COU:
				interval, ok := d.(IInterval)
				if !ok {
					continue
				}
				if origin, exists := d.GetOriginData()[COU]; exists {
					interval.SetCou(origin.(float64))
					restored = true
				}
			case FLAG:
				if origin, exists := d.GetOriginData()[FLAG]; exists {
					d.SetFlag(origin.(string))
					restored = true
				}
			case FLAG_BIT:
				if origin, exists := d.GetOriginData()[FLAG_BIT]; exists {
					d.SetFlagBit(origin.(int))
					restored = true
				}
			}
		}
	}

	return restored
}

func ModifyValue(d IData, field string, value interface{}, uid int) error {

	var origin interface{}

	switch field {
	case RTD:
		rtd, ok := d.(IRealTime)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(float64)
		if !ok {
			return e_invalid_data_value
		}
		if v == rtd.GetRtd() {
			return nil
		}
		origin = rtd.GetRtd()
		rtd.SetRtd(v)
	case AVG:
		interval, ok := d.(IInterval)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(float64)
		if !ok {
			return e_invalid_data_value
		}
		if v == interval.GetAvg() {
			return nil
		}
		origin = interval.GetAvg()
		interval.SetAvg(v)
	case MIN:
		interval, ok := d.(IInterval)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(float64)
		if !ok {
			return e_invalid_data_value
		}
		if v == interval.GetMin() {
			return nil
		}
		origin = interval.GetMin()
		interval.SetMin(v)
	case MAX:
		interval, ok := d.(IInterval)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(float64)
		if !ok {
			return e_invalid_data_value
		}
		if v == interval.GetMax() {
			return nil
		}
		origin = interval.GetMax()
		interval.SetMax(v)
	case COU:
		interval, ok := d.(IInterval)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(float64)
		if !ok {
			return e_invalid_data_value
		}
		if v == interval.GetCou() {
			return nil
		}
		origin = interval.GetCou()
		interval.SetCou(v)
	case FLAG:
		v, ok := value.(string)
		if !ok {
			return e_invalid_data_value
		}
		if v == d.GetFlag() || v == "" {
			return nil
		}
		if d.GetFlag() != "" {
			origin = d.GetFlag()
		}
		d.SetFlag(v)
	case FLAG_BIT:
		v, ok := value.(int)
		if !ok {
			return e_invalid_data_value
		}
		if v == d.GetFlagBit() || v < 0 {
			return nil
		}
		if d.GetFlagBit() > 0 {
			origin = d.GetFlagBit()
		}
		d.SetFlagBit(v)
		log.Println("modify flag bit: ", v, origin, d.GetFlagBit())
	case REVIEWED:
		review, ok := d.(IReview)
		if !ok {
			return e_invalid_data_interface
		}
		v, ok := value.(bool)
		if !ok {
			return e_invalid_data_value
		}
		review.SetReviewed(v)
	}

	if origin != nil {

		d.LockOriginData()
		defer d.UnLockOriginData()

		originData := d.GetOriginData()
		if originData == nil {
			originData = make(map[string]interface{})
		}

		if _, exists := originData[field]; !exists {
			originData[field] = origin
		} else {
			if value == originData[field] {
				delete(originData, field)
				delete(originData, field+"_at")
				delete(originData, field+"_by")

				d.SetOriginData(originData)
				return nil
			}
		}

		if uid > 0 {
			originData[field+"_at"] = util.FormatDateTime(time.Now())
			originData[field+"_by"] = uid
		}
		d.SetOriginData(originData)
	}

	return nil
}

func SetFlagBit(dataBit, flagBit int) int {
	return dataBit | flagBit
}

func ClearFlagBit(dataBit, flagBit int) int {
	if dataBit&flagBit == flagBit {
		return dataBit ^ flagBit
	}
	return dataBit
}
