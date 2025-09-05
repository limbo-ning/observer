package monitor

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/entity"
	"strconv"
	"strings"
)

const NOTATION_INFINITY = "∞"

const NOTATION_IGNORE = math.MinInt16

type MonitorLimit struct {
	ID                int    `json:"ID"`
	MonitorID         int    `json:"monitorID"`
	StationID         int    `json:"stationID"`
	Overproof         string `json:"overproof"`
	overproofSegments []*OverproofSegment
	TopEffective      float64 `json:"topEffective"`
	LowerDetection    float64 `json:"lowerDetection"`
	InvarianceHour    int     `json:"invarianceHour"`
}

type OverproofSegment struct {
	L float64
	U float64
}

const monitorLimitColumns = "monitorLimit.id, monitorLimit.monitor_id, monitorLimit.station_id, monitorLimit.overproof, monitorLimit.top_effective, monitorLimit.lower_detection, monitorLimit.invariance_hour"

func monitorLimitTableName(siteID string) string {
	return siteID + "_monitorlimit"
}

func (m *MonitorLimit) scan(rows *sql.Rows) error {

	if err := rows.Scan(&m.ID, &m.MonitorID, &m.StationID, &m.Overproof, &m.TopEffective, &m.LowerDetection, &m.InvarianceHour); err != nil {
		return err
	}

	if err := m.parseOverproofSegment(); err != nil {
		return err
	}
	return nil
}

func (m *MonitorLimit) parseOverproofSegment() error {
	m.overproofSegments = make([]*OverproofSegment, 0)
	if m.Overproof == "" {
		return nil
	}

	segments := strings.Split(m.Overproof, ";")
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		parts := strings.Split(segment, ",")
		if len(parts) != 2 {
			return fmt.Errorf("错误的超标区间:%s", segment)
		}

		result := &OverproofSegment{}

		if parts[0] == "-"+NOTATION_INFINITY {
			result.L = -math.MaxFloat64
		} else {
			v, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return fmt.Errorf("错误的下限取值:%s", parts[0])
			}
			result.L = v
		}

		if parts[1] == "+"+NOTATION_INFINITY {
			result.U = math.MaxFloat64
		} else {
			v, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return fmt.Errorf("错误的上限取值:%s", parts[1])
			}
			result.U = v
		}

		m.overproofSegments = append(m.overproofSegments, result)
	}

	return nil
}

func (m *MonitorLimit) IsOverproof(value float64) bool {
	for _, s := range m.overproofSegments {
		if s.L < value && s.U > value {
			return true
		}
	}

	return false
}

func (m *MonitorLimit) GetStationID() int { return m.StationID }

func (m *MonitorLimit) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if m.MonitorID <= 0 {
		return e_need_monitor_id
	}

	if err := m.parseOverproofSegment(); err != nil {
		return err
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{m.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[m.StationID] {
		return errors.New("无权限")
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(monitor_id,station_id,overproof,top_effective,lower_detection,invariance_hour)
		VALUES
			(?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
		 	overproof=VALUES(overproof),top_effective=VALUES(top_effective),lower_detection=VALUES(lower_detection),invariance_hour=VALUES(invariance_hour)
	`, monitorLimitTableName(siteID)), m.MonitorID, m.StationID, m.Overproof, m.TopEffective, m.LowerDetection, m.InvarianceHour); err != nil {
		log.Println("error insert monitor limit: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor limit: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *MonitorLimit) Update(siteID string, actionAuth authority.ActionAuthSet) error {
	if m.MonitorID <= 0 {
		return e_need_monitor_id
	}
	if err := m.parseOverproofSegment(); err != nil {
		return err
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{m.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[m.StationID] {
		return errors.New("无权限")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			monitor_id=?,station_id=?,overproof=?,top_effective=?,lower_detection=?,invariance_hour=?
		WHERE
			id=?
	`, monitorLimitTableName(siteID)), m.MonitorID, m.StationID, m.Overproof, m.TopEffective, m.LowerDetection, m.InvarianceHour, m.ID); err != nil {
		log.Println("error update monitor limit: ", err)
		return err
	}

	return nil
}
func (m *MonitorLimit) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

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
	`, monitorLimitTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor limit: ", err)
		return err
	}

	return nil
}

func GetMonitorLimits(siteID string, monitorIDs []int, stationID ...int) ([]*MonitorLimit, error) {
	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(monitorIDs) > 0 {
		if len(monitorIDs) == 1 {
			whereStmts = append(whereStmts, "monitorLimit.monitor_id = ?")
			values = append(values, monitorIDs[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorIDs {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("monitorLimit.monitor_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(stationID) != 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "monitorLimit.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("monitorLimit.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s monitorLimit
	`, monitorLimitColumns, monitorLimitTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor limit: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*MonitorLimit, 0)

	for rows.Next() {
		var s MonitorLimit
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor limit: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil

}

type MonitorLimitTemplate struct {
	ID   int    `json:"ID"`
	Name string `json:"name"`
}

func monitorLimitTemplateTableName(siteID string) string {
	return siteID + "_monitorlimittemplate"
}

const monitorLimitTemplateColumn = "monitorLimitTemplate.id,monitorLimitTemplate.name"

func (m *MonitorLimitTemplate) scan(rows *sql.Rows) error {
	if err := rows.Scan(&m.ID, &m.Name); err != nil {
		return err
	}
	return nil
}

func (m *MonitorLimitTemplate) Add(siteID string) error {

	if m.Name == "" {
		return errors.New("请命名模版")
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name)
		VALUES
			(?)
	`, monitorLimitTemplateTableName(siteID)), m.Name); err != nil {
		log.Println("error insert monitor limit template: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor limit template: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *MonitorLimitTemplate) Update(siteID string) error {
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
	`, monitorLimitTemplateTableName(siteID)), m.Name, m.ID); err != nil {
		log.Println("error update monitor limit template: ", err)
		return err
	}

	return nil
}
func (m *MonitorLimitTemplate) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, monitorLimitTemplateTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor limit template: ", err)
		return err
	}

	return nil
}

func GetMonitorLimitTemplates(siteID string) ([]*MonitorLimitTemplate, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s monitorLimitTemplate
	`, monitorLimitTemplateColumn, monitorLimitTemplateTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor limit template: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*MonitorLimitTemplate, 0)

	for rows.Next() {
		var s MonitorLimitTemplate
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor limit template: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}
