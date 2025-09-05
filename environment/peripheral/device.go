package peripheral

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

var e_device_not_exist = errors.New("设备不存在")

type Device struct {
	ID     int                    `json:"ID"`
	Serial string                 `json:"serial"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Ext    map[string]interface{} `json:"ext"`
}

const deviceColumns = "device.id,device.serial,device.name,device.type,device.ext"

func deviceTableName(siteID string) string {
	return siteID + "_device"
}

func (d *Device) scan(rows *sql.Rows) error {
	var ext string
	if err := rows.Scan(&d.ID, &d.Serial, &d.Name, &d.Type, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(ext), &d.Ext); err != nil {
		return err
	}
	return nil
}

func (d *Device) Add(siteID string) error {
	if d.Serial == "" {
		return errors.New("需要设备serial")
	}
	if d.Type == "" {
		return errors.New("需要设备type")
	}
	if d.Ext == nil {
		d.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(d.Ext)
	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name, serial, type, ext)
		VALUES
			(?,?,?,?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name), ext = VALUES(ext)
	`, deviceTableName(siteID)), d.Name, d.Serial, d.Type, string(ext)); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		d.ID = int(id)
	}
	return nil
}

func (d *Device) Update(siteID string) error {

	if d.Serial == "" {
		return errors.New("需要设备serial")
	}
	if d.Type == "" {
		return errors.New("需要设备type")
	}

	if d.Ext == nil {
		d.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(d.Ext)
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name = ?, serial = ?, type=?, ext = ?
		WHERE
			id = ?
	`, deviceTableName(siteID)), d.Name, d.Serial, d.Type, string(ext), d.ID); err != nil {
		return err
	}

	return nil
}

func (d *Device) Delete(siteID string) error {
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, deviceTableName(siteID)), d.ID); err != nil {
		return err
	}

	return nil
}

func getDevice(siteID string, txn *sql.Tx, forUpdate bool, deviceID ...int) ([]*Device, error) {

	result := make([]*Device, 0)

	if len(deviceID) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(deviceID) == 1 {
		whereStmts = append(whereStmts, "id = ?")
		values = append(values, deviceID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range deviceID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("id IN (%s)", strings.Join(placeholder, ",")))
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s device
		WHERE
			%s
	`, deviceColumns, deviceTableName(siteID), strings.Join(whereStmts, " AND "))

	var rows *sql.Rows
	var err error

	if txn != nil {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	} else {
		rows, err = datasource.GetConn().Query(SQL, values...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d Device
		if err := d.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &d)
	}
	return result, nil
}

func GetDevices(siteID string, actionAuth authority.ActionAuthSet, serial, deviceType, q string, pageNo, pageSize int, authType, empowerType string, empowerID ...string) ([]*Device, int, error) {

	whereStmts := make([]string, 0)
	values := make([]any, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s device
	`, deviceColumns, deviceTableName(siteID))

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT device.id)
		FROM
			%s device
	`, deviceTableName(siteID))

	if authType == "" {
		authType = ACTION_VIEW
	}
	authSQL, authWhere, authValues, err := authority.JoinEmpower(siteID, "device", actionAuth, AdminActions, authType, "device", "id", empowerType, empowerID...)
	if err != nil {
		return nil, 0, err
	}
	countSQL += authSQL
	SQL += authSQL
	if authWhere != nil {
		whereStmts = append(whereStmts, authWhere...)
	}
	if authValues != nil {
		values = append(values, authValues...)
	}

	if serial != "" {
		whereStmts = append(whereStmts, "device.serial = ?")
		values = append(values, serial)
	}

	if deviceType != "" {
		whereStmts = append(whereStmts, "device.type=?")
		values = append(values, deviceType)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "device.name LIKE ?")
		values = append(values, qq)
	}

	if len(whereStmts) > 0 {
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	total := 0

	if pageSize != -1 {
		if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
			log.Println("error count peripheral device: ", countSQL, values, err)
			return nil, 0, err
		}
	}

	SQL += "\nGROUP BY device.id"
	SQL += "\nORDER BY device.id DESC"
	if pageSize != -1 {
		if pageNo <= 0 {
			pageNo = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		SQL += "\nLIMIT ?,?"
		values = append(values, (pageNo-1)*pageSize, pageSize)
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get peripheral device: ", SQL, values, err)
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]*Device, 0)

	for rows.Next() {
		var d Device
		if err := d.scan(rows); err != nil {
			return nil, 0, err
		}
		result = append(result, &d)
	}

	if pageSize == -1 {
		total = len(result)
	}

	return result, total, nil
}
