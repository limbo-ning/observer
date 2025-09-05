package monitor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/entity"
)

const (
	CODE_DEFAULT = "DEFAULT"
)

var e_need_code = errors.New("需要因子代码")
var e_need_monitor_id = errors.New("需要监控物")

// var e_need_station_id = errors.New("需要排放点")
var e_duplicate = errors.New("因子代码重复")

type MonitorCodeKey struct {
	ID        int    `json:"ID"`
	Code      string `json:"code"`
	MonitorID int    `json:"monitorID"`
	StationID int    `json:"stationID"`
}

type MonitorCode struct {
	MonitorCodeKey
	Processors dataprocess.DataProcessors `json:"processors"`
	Ext        map[string]interface{}     `json:"ext"`
}

func monitorCodeTableName(siteID string) string {
	return siteID + "_monitorcode"
}

const monitorCodeKeyColumn = "monitorCode.id,monitorCode.code,monitorCode.monitor_id,monitorCode.station_id"

const monitorCodeColumn = monitorCodeKeyColumn + ",monitorCode.processors,monitorCode.ext"

func (m *MonitorCodeKey) scan(rows *sql.Rows) error {
	if err := rows.Scan(&m.ID, &m.Code, &m.MonitorID, &m.StationID); err != nil {
		return err
	}
	return nil
}

func (m *MonitorCode) scan(rows *sql.Rows) error {
	var processor, ext string
	if err := rows.Scan(&m.ID, &m.Code, &m.MonitorID, &m.StationID, &processor, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(processor), &m.Processors); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(ext), &m.Ext); err != nil {
		return err
	}
	return nil
}

func (m *MonitorCodeKey) GetStationID() int { return m.StationID }

func (m *MonitorCode) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if m.Code == "" {
		return e_need_code
	}
	if m.MonitorID <= 0 {
		return e_need_monitor_id
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{m.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[m.StationID] {
		return errors.New("无权限")
	}

	processors, _ := json.Marshal(m.Processors)

	if m.Ext == nil {
		m.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(m.Ext)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(code,monitor_id,station_id,processors,ext)
		VALUES
			(?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			monitor_id=VALUES(monitor_id), processors=VALUES(processors), ext=VALUES(ext)
	`, monitorCodeTableName(siteID)), m.Code, m.MonitorID, m.StationID, string(processors), string(ext)); err != nil {
		log.Println("error insert monitor code: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor code: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *MonitorCode) Update(siteID string, actionAuth authority.ActionAuthSet) error {
	if m.Code == "" {
		return e_need_code
	}
	if m.MonitorID <= 0 {
		return e_need_monitor_id
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{m.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[m.StationID] {
		return errors.New("无权限")
	}

	processors, _ := json.Marshal(m.Processors)

	if m.Ext == nil {
		m.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(m.Ext)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			code=?,monitor_id=?,station_id=?,processors=?,ext=?
		WHERE
			id=?
	`, monitorCodeTableName(siteID)), m.Code, m.MonitorID, m.StationID, string(processors), string(ext), m.ID); err != nil {
		log.Println("error update monitor code: ", err)
		return err
	}

	return nil
}
func (m *MonitorCode) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{m.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[m.StationID] {
		return errors.New("无权限")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, monitorCodeTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor code: ", err)
		return err
	}

	return nil
}

func GetMonitorCodes(siteID string, monitorID int, q string, stationID ...int) ([]*MonitorCode, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if monitorID > 0 {
		whereStmts = append(whereStmts, "monitorCode.monitor_id = ?")
		values = append(values, monitorID)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "monitorCode.code LIKE ?")
		values = append(values, qq)
	}

	if len(stationID) != 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "monitorCode.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("monitorCode.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s monitorCode
	`, monitorCodeColumn, monitorCodeTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY monitorCode.code ASC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor code: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*MonitorCode, 0)

	for rows.Next() {
		var s MonitorCode
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor code: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}

type MonitorCodeTemplate struct {
	ID   int    `json:"ID"`
	Name string `json:"name"`
}

func monitorCodeTemplateTableName(siteID string) string {
	return siteID + "_monitorcodetemplate"
}

const monitorCodeTemplateColumn = "monitorCodeTemplate.id,monitorCodeTemplate.name"

func (m *MonitorCodeTemplate) scan(rows *sql.Rows) error {
	if err := rows.Scan(&m.ID, &m.Name); err != nil {
		return err
	}
	return nil
}

func (m *MonitorCodeTemplate) Add(siteID string) error {

	if m.Name == "" {
		return errors.New("请命名模版")
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name)
		VALUES
			(?)
	`, monitorCodeTemplateTableName(siteID)), m.Name); err != nil {
		log.Println("error insert monitor code template: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor code template: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *MonitorCodeTemplate) Update(siteID string) error {
	if m.Name == "" {
		return errors.New("请命名模版")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?
		WHERE
			id=?
	`, monitorCodeTemplateTableName(siteID)), m.Name, m.ID); err != nil {
		log.Println("error update monitor code template: ", err)
		return err
	}

	return nil
}
func (m *MonitorCodeTemplate) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, monitorCodeTemplateTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor code template: ", err)
		return err
	}

	return nil
}

func GetMonitorCodeTemplates(siteID string) ([]*MonitorCodeTemplate, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s monitorCodeTemplate
	`, monitorCodeTemplateColumn, monitorCodeTemplateTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor code template: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*MonitorCodeTemplate, 0)

	for rows.Next() {
		var s MonitorCodeTemplate
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor code template: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}
