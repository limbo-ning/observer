package monitor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/category"
	"obsessiontech/environment/environment/entity"
)

const (
	MONITOR_DATA = 1 << iota
	MONITOR_SWITCH
)

type Monitor struct {
	ID        int                    `json:"ID"`
	Name      string                 `json:"name"`
	Type      int                    `json:"type"`
	Unit      string                 `json:"unit"`
	CouUnit   string                 `json:"couUnit"`
	Precision int                    `json:"precision"`
	Ext       map[string]interface{} `json:"ext"`
}

const monitorColumn = "monitor.id,monitor.name,monitor.type,monitor.unit,monitor.cou_unit,monitor.decimal_precision,monitor.ext"

func monitorTableName(siteID string) string {
	return siteID + "_monitor"
}

func (m *Monitor) scan(rows *sql.Rows, appendix ...interface{}) error {
	var ext string
	dest := make([]interface{}, 0)
	dest = append(dest, &m.ID, &m.Name, &m.Type, &m.Unit, &m.CouUnit, &m.Precision, &ext)
	dest = append(dest, appendix...)
	if err := rows.Scan(dest...); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(ext), &m.Ext); err != nil {
		return err
	}

	return nil
}

func (m *Monitor) Add(siteID string) error {
	if m.Ext == nil {
		m.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(&m.Ext)
	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name,type,unit,cou_unit,decimal_precision,ext)
		VALUES
			(?,?,?,?,?,?)
	`, monitorTableName(siteID)), m.Name, m.Type, m.Unit, m.CouUnit, m.Precision, string(ext)); err != nil {
		log.Println("error insert monitor: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor: ", err)
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *Monitor) Update(siteID string) error {
	if m.Ext == nil {
		m.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(&m.Ext)
	SQL := fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?, type=?, unit=?, cou_unit=?, decimal_precision=?, ext=? 
		WHERE
			id=?
	`, monitorTableName(siteID))

	if _, err := datasource.GetConn().Exec(SQL, m.Name, m.Type, m.Unit, m.CouUnit, m.Precision, string(ext), m.ID); err != nil {
		log.Println("error update monitor: ", SQL, err)
		return err
	}

	return nil
}
func (m *Monitor) Delete(siteID string) error {
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, monitorTableName(siteID)), m.ID); err != nil {
		log.Println("error delete monitor: ", err)
		return err
	}
	return nil
}

func GetMonitors(siteID string, cids []int, monitorTypeBits []int, monitorID ...int) ([]*Monitor, error) {

	result := make([]*Monitor, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s monitor
	`, monitorColumn, monitorTableName(siteID))

	if len(cids) > 0 {
		joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "monitor", "", cids...)
		if err != nil {
			return nil, err
		}
		if joinSQL == "" {
			return result, nil
		}

		SQL += joinSQL

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if len(monitorID) != 0 {
		if len(monitorID) == 1 {
			whereStmts = append(whereStmts, "monitor.id = ?")
			values = append(values, monitorID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("monitor.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(monitorTypeBits) > 0 {
		if len(monitorTypeBits) == 1 {
			whereStmts = append(whereStmts, "monitor.type & ? = ?")
			values = append(values, monitorTypeBits[0], monitorTypeBits[0])
		} else {
			subStmts := make([]string, 0)
			for _, t := range monitorTypeBits {
				subStmts = append(subStmts, "monitor.type & ? = ?")
				values = append(values, t, t)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("(%s)", strings.Join(subStmts, " OR ")))
		}
	}

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor: ", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var s Monitor
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}

func GetStationMonitors(siteID string, actionAuth authority.ActionAuthSet, stationID ...int) (map[int][]int, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(stationID) > 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "monitor_code.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("monitor_code.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			monitor.id, monitor_code.station_id, monitor_code.code
		FROM
			%s monitor
		JOIN
			%s monitor_code
		ON
			monitor.id = monitor_code.monitor_id
	`, monitorTableName(siteID), monitorCodeTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY monitor_code.code ASC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get station monitor: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]int)
	stationIDs := make([]int, 0)

	for rows.Next() {
		var monitorID, stationID int
		var code string

		if err := rows.Scan(&monitorID, &stationID, &code); err != nil {
			log.Println("error get station monitor: ", err)
			return nil, err
		}

		list, exists := result[stationID]
		if !exists {
			list = make([]int, 0)
			stationIDs = append(stationIDs, stationID)
		}

		exists = false
		for _, id := range list {
			if id == monitorID {
				exists = true
				continue
			}
		}

		if exists {
			continue
		}

		list = append(list, monitorID)
		result[stationID] = list
	}

	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, stationIDs, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return nil, err
	}

	for _, s := range stationIDs {
		if !filtered[s] {
			delete(result, s)
		}
	}

	return result, nil
}

func AddMonitorCategory(siteID string, monitorID, categoryID int) error {
	return category.AddCategoryMapping(siteID, "monitor", monitorID, categoryID)
}

func DeleteMonitorCategory(siteID string, monitorID, categoryID int) error {
	return category.DeleteCategoryMapping(siteID, "entity", monitorID, categoryID)
}

func CountStationMonitor(siteID string, actionAuth authority.ActionAuthSet, monitorTypeBits []int, stationID ...int) (map[int]int, error) {

	result := make(map[int]int)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(monitorTypeBits) > 0 {
		if len(monitorTypeBits) == 1 {
			whereStmts = append(whereStmts, "m.type & ? = ?")
			values = append(values, monitorTypeBits[0], monitorTypeBits[0])
		} else {
			subStmts := make([]string, 0)
			for _, t := range monitorTypeBits {
				subStmts = append(subStmts, "m.type & ? = ?")
				values = append(values, t, t)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("(%s)", strings.Join(subStmts, " OR ")))
		}
	}

	if len(stationID) > 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "mc.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("mc.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT 
			COUNT(DISTINCT mc.monitor_id), mc.station_id
		FROM
			%s mc
		JOIN
			%s m
		ON
			m.id = mc.monitor_id
		WHERE
			%s
		GROUP BY
			mc.station_id
	`, monitorCodeTableName(siteID), monitorTableName(siteID), strings.Join(whereStmts, " AND ")), values...)

	if err != nil {
		log.Println("error count station monitor: ", err)
		return nil, err
	}

	defer rows.Close()

	stationIDs := make([]int, 0)
	for rows.Next() {
		var count, stationID int

		if err := rows.Scan(&count, &stationID); err != nil {
			log.Println("error get station monitor: ", err)
			return nil, err
		}
		result[stationID] = count
		stationIDs = append(stationIDs, stationID)
	}

	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, stationIDs, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return nil, err
	}

	for _, s := range stationIDs {
		if !filtered[s] {
			delete(result, s)
		}
	}

	return result, nil
}
