package stats

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
)

func CountSlots(dataType string, beginDateTime, endDateTime *time.Time) int {
	switch dataType {
	case data.REAL_TIME:
		return int(math.Round(endDateTime.Sub(*beginDateTime).Minutes()))
	case data.MINUTELY:
		return int(math.Round(endDateTime.Sub(*beginDateTime).Minutes()) / 10)
	case data.HOURLY:
		return int(math.Round(endDateTime.Sub(*beginDateTime).Hours()))
	case data.DAILY:
		return int(math.Round(endDateTime.Sub(*beginDateTime).Hours()) / 24)
	default:
		return 0
	}
}

func countHourlySlot(beginDateTime, endDateTime *time.Time) int {
	return CountSlots(data.HOURLY, beginDateTime, endDateTime)
}

func getMonitorStatistics(siteID string, beginDateTime, endDateTime *time.Time, stationID int, monitorID ...int) (map[int]map[string]int, error) {
	result := make(map[int]map[string]int)

	if len(monitorID) == 0 {
		return result, nil
	}

	tables := data.FetchTableNames(siteID, data.HOURLY, *beginDateTime, *endDateTime)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	whereStmts = append(whereStmts, "station_id = ?", "data_time >= ?", "data_time < ?")
	values = append(values, stationID, beginDateTime, endDateTime)

	types := []int{monitor.MONITOR_DATA}
	monitors, err := monitor.GetMonitors(siteID, nil, types, monitorID...)
	if err != nil {
		return nil, err
	}

	if len(monitors) == 0 {
		return result, nil
	}

	if len(monitors) == 1 {
		whereStmts = append(whereStmts, "monitor_id =? ")
		values = append(values, monitors[0].ID)
	} else {
		placeholder := make([]string, 0)
		for _, m := range monitors {
			placeholder = append(placeholder, "?")
			values = append(values, m.ID)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("monitor_id IN (%s)", strings.Join(placeholder, ",")))
	}

	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				monitor_id, flag, COUNT(1)
			FROM
				%s
			WHERE
				%s
			GROUP BY
				monitor_id, flag
		`, table, strings.Join(whereStmts, " AND "))

		log.Println("SQL: ", SQL, values)

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error count station monitor data flag: ", SQL, values, err)
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var f string
			var m, c int

			rows.Scan(&m, &f, &c)
			if _, exists := result[m]; !exists {
				result[m] = make(map[string]int)
			}
			result[m][f] = c

			// flag, err := monitor.GetFlag(siteID, f)
			// if err != nil {
			// 	return nil, err
			// }

			// if _, exists := result[m]; !exists {
			// 	result[m] = &[2]int{}
			// }

			// if flag != nil && monitor.CheckFlag(monitor.FLAG_EFFECTIVE, flag.Bits) {
			// 	result[m][0] += c
			// }

			// result[m][1] += c
		}
	}

	return result, nil
}

func getStationStatistics(siteID string, beginDateTime, endDateTime *time.Time, stationID ...int) (map[int]map[string]int, error) {

	result := make(map[int]map[string]int)

	if len(stationID) == 0 {
		return result, nil
	}

	whereStmt := make([]string, 0)
	values := make([]interface{}, 0)

	whereStmt = append(whereStmt, "data_time >= ?", "data_time < ?")
	values = append(values, beginDateTime, endDateTime)

	stationMonitors, err := monitor.GetStationMonitors(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, stationID...)
	if err != nil {
		return nil, err
	}

	if len(stationID) > 0 {
		stmts := make([]string, 0)

		for _, sid := range stationID {
			mids := stationMonitors[sid]

			if len(mids) == 0 {
				continue
			}

			values = append(values, sid)

			placeholder := make([]string, 0)
			for _, mid := range mids {
				placeholder = append(placeholder, "?")
				values = append(values, mid)
			}

			stmts = append(stmts, fmt.Sprintf("(station_id = ? AND monitor_id IN (%s))", strings.Join(placeholder, ",")))
		}

		whereStmt = append(whereStmt, fmt.Sprintf("(%s)", strings.Join(stmts, " OR ")))
	}
	types := []int{monitor.MONITOR_DATA}

	monitors, err := monitor.GetMonitors(siteID, nil, types)
	if err != nil {
		return nil, err
	}

	if len(monitors) == 0 {
		return result, nil
	}

	placeholder := make([]string, 0)
	for _, m := range monitors {
		placeholder = append(placeholder, "?")
		values = append(values, m.ID)
	}
	whereStmt = append(whereStmt, fmt.Sprintf("monitor_id IN (%s)", strings.Join(placeholder, ",")))

	tables := data.FetchTableNames(siteID, data.HOURLY, *beginDateTime, *endDateTime)

	for _, table := range tables {

		SQL := `
			SELECT
				station_id, flag, COUNT(1)
			FROM
				` + table + `
			WHERE
				` + strings.Join(whereStmt, " AND ") + `
			GROUP BY
				station_id, flag
		`

		log.Println("SQL: ", SQL, values)

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error count station data flag: ", SQL, err)
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var f string
			var s, c int

			rows.Scan(&s, &f, &c)
			if _, exists := result[s]; !exists {
				result[s] = make(map[string]int)
			}
			result[s][f] = c

			// flag, err := monitor.GetFlag(siteID, f)
			// if err != nil {
			// 	return nil, err
			// }

			// if _, exists := result[s]; !exists {
			// 	result[s] = &[2]int{}
			// }

			// if flag != nil && monitor.CheckFlag(monitor.FLAG_EFFECTIVE, flag.Bits) {
			// 	result[s][0] += c
			// }

			// result[s][1] += c

		}
	}

	return result, nil
}
