package vehicle

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"strings"
	"time"
)

const (
	_ = iota
	VEHICLE_IDLE
	VEHICLE_INUSE
	VEHICLE_MAINTENANCE
	VEHICLE_INACTIVE
)

type Vehicle struct {
	ID         int                 `json:"ID"`
	Serial     string              `json:"serial"`
	Type       string              `json:"type"`
	Name       string              `json:"name"`
	Status     int                 `json:"status"`
	Profile    map[string][]string `json:"profile"`
	Ext        map[string]any      `json:"ext"`
	CreateTime util.Time           `json:"createTime"`
	UpdateTime util.Time           `json:"updateTime"`
}

const vehicleColumns = "vehicle.id, vehicle.serial, vehicle.type, vehicle.name, vehicle.status, vehicle.profile, vehicle.ext, vehicle.create_time, vehicle.update_time"

func vehicleTable(siteID string) string {
	return siteID + "_vehicle"
}

func (v *Vehicle) scan(rows *sql.Rows) error {

	var profile, ext string

	if err := rows.Scan(&v.ID, &v.Serial, &v.Type, &v.Name, &v.Status, &profile, &ext, &v.CreateTime, &v.UpdateTime); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(profile), &v.Profile); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(ext), &v.Ext); err != nil {
		return err
	}

	return nil
}

func GetVehicleList(siteID string, vehicleType string, status int, search string) ([]*Vehicle, error) {

	result := make([]*Vehicle, 0)

	whereStmts := make([]string, 0)
	values := make([]any, 0)

	if vehicleType != "" {
		whereStmts = append(whereStmts, "vehicle.type = ?")
		values = append(values, vehicleType)
	}

	if status > 0 {
		whereStmts = append(whereStmts, "vehicle.status = ?")
		values = append(values, status)
	}

	if search != "" {
		whereStmts = append(whereStmts, "(vehicle.name LIKE ? OR vehicle.serial LIKE ?)")
		values = append(values, "%"+search+"%", "%"+search+"%")
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s vehicle
	`, vehicleColumns, vehicleTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY vehicle.update_time DESC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get vehicle: ", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var v Vehicle
		if err := v.scan(rows); err != nil {
			log.Println("error scan vehicle: ", err)
			return nil, err
		}
		result = append(result, &v)
	}

	return result, nil

}

func (v *Vehicle) Validate(siteID string) error {
	if v.Serial == "" {
		return errors.New("need serial")
	}
	if v.Name == "" {
		return errors.New("need name")
	}

	switch v.Status {
	case VEHICLE_IDLE:
	case VEHICLE_INUSE:
	case VEHICLE_MAINTENANCE:
	case VEHICLE_INACTIVE:
	default:
		v.Status = VEHICLE_IDLE
	}

	sm, err := GetModule(siteID)
	if err != nil {
		return err
	}

	checked := false
	for _, vt := range sm.Types {
		if vt.Type == v.Type {
			checked = true
			break
		}
	}

	if !checked {
		return errors.New("error invalid vehicle type")
	}

	if v.Profile == nil {
		v.Profile = make(map[string][]string, 0)
	}
	if v.Ext == nil {
		v.Ext = make(map[string]any)
	}
	return nil
}

func (v *Vehicle) Add(siteID string) error {
	if err := v.Validate(siteID); err != nil {
		return err
	}

	profile, _ := json.Marshal(v.Profile)
	ext, _ := json.Marshal(v.Ext)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(serial,type,name,status,profile,ext)
		VALUES
			(?,?,?,?,?,?)
	`, vehicleTable(siteID)), v.Serial, v.Type, v.Name, v.Status, string(profile), string(ext)); err != nil {
		log.Println("error add vehicle: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add vehicle: ", err)
		return err
	} else {
		v.ID = int(id)
		v.CreateTime = util.Time(time.Now())
		v.UpdateTime = util.Time(time.Now())
	}

	return nil
}

func (v *Vehicle) Update(siteID string) error {
	if err := v.Validate(siteID); err != nil {
		return err
	}

	profile, _ := json.Marshal(v.Profile)
	ext, _ := json.Marshal(v.Ext)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			serial=?,type=?,name=?,status=?,profile=?,ext=?
		WHERE
			id=?
	`, vehicleTable(siteID)), v.Serial, v.Type, v.Name, v.Status, string(profile), string(ext), v.ID); err != nil {
		log.Println("error update vehicle: ", err)
		return err
	}

	return nil
}

func (v *Vehicle) Delete(siteID string) error {
	if err := v.Validate(siteID); err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, vehicleTable(siteID)), v.ID); err != nil {
		log.Println("error delete vehicle: ", err)
		return err
	}

	return nil
}
