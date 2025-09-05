package stats

// import (
// 	"fmt"
// 	"log"
// 	"strings"
// 	"time"

// 	"obsessiontech/common/datasource"
// 	"obsessiontech/environment/environment/data"
// 	"obsessiontech/environment/environment/flag"
// )

// func countHourlySlot(beginDateTime, endDateTime *time.Time) int {
// 	return int(endDateTime.Sub(*beginDateTime).Hours())
// }

// func getMonitorStatistics(siteID string, beginDateTime, endDateTime *time.Time, stationID int, monitorID ...int) (map[int]*[3]int, error) {

// 	result := make(map[int]*[3]int)

// 	if len(monitorID) == 0 {
// 		return result, nil
// 	}

// 	flagList, err := flag.GetFlags(siteID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	flags := make(map[string]*flag.Flag)
// 	for _, f := range flagList {
// 		flags[f.Flag] = f
// 	}

// 	tables := data.FetchTableNames(siteID, data.HOURLY, *beginDateTime, *endDateTime)

// 	whereStmts := make([]string, 0)
// 	values := make([]interface{}, 0)

// 	whereStmts = append(whereStmts, "station_id = ?", "data_time >= ?", "data_time <= ?")
// 	values = append(values, stationID, beginDateTime, endDateTime)

// 	if len(monitorID) == 1 {
// 		whereStmts = append(whereStmts, "monitor_id =? ")
// 		values = append(values, monitorID[0])
// 	} else {
// 		placeholder := make([]string, 0)
// 		for _, id := range monitorID {
// 			placeholder = append(placeholder, "?")
// 			values = append(values, id)
// 		}
// 		whereStmts = append(whereStmts, fmt.Sprintf("monitor_id IN (%s)", strings.Join(placeholder, ",")))
// 	}

// 	for _, table := range tables {
// 		SQL := fmt.Sprintf(`
// 			SELECT
// 				monitor_id, flag, COUNT(1)
// 			FROM
// 				%s
// 			WHERE
// 				%s
// 			GROUP BY
// 				monitor_id, flag
// 		`, table, strings.Join(whereStmts, " AND "))

// 		rows, err := datasource.GetConn().Query(SQL, values...)
// 		if err != nil {
// 			log.Println("error count station monitor data flag: ", SQL, values, err)
// 			return nil, err
// 		}

// 		defer rows.Close()

// 		for rows.Next() {
// 			var f string
// 			var m, c int

// 			rows.Scan(&m, &f, &c)

// 			if _, exists := result[m]; !exists {
// 				result[m] = &[3]int{}
// 			}

// 			flag, exists := flags[f]
// 			if exists {
// 				if flag.IsEffective {
// 					result[m][0] += c
// 				} else if flag.IsOutOfService {
// 					result[m][1] += c
// 				}
// 			}

// 			result[m][2] += c
// 		}
// 	}

// 	return result, nil
// }

// func getStationStatistics(siteID string, beginDateTime, endDateTime *time.Time, stationID ...int) (map[int]*[3]int, error) {

// 	flagList, err := flag.GetFlags(siteID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	flags := make(map[string]*flag.Flag)
// 	for _, f := range flagList {
// 		flags[f.Flag] = f
// 	}

// 	result := make(map[int]*[3]int)

// 	whereStmt := make([]string, 0)
// 	values := make([]interface{}, 0)

// 	whereStmt = append(whereStmt, "data_time >= ?", "data_time < ?")
// 	values = append(values, beginDateTime, endDateTime)

// 	if len(stationID) > 0 {
// 		stationPlaceHolder := strings.Repeat("?,", len(stationID))
// 		whereStmt = append(whereStmt, "station_id IN ("+stationPlaceHolder[:len(stationPlaceHolder)-1]+")")

// 		for _, stationID := range stationID {
// 			values = append(values, stationID)
// 		}
// 	}

// 	tables := data.FetchTableNames(siteID, data.HOURLY, *beginDateTime, *endDateTime)

// 	for _, table := range tables {

// 		rows, err := datasource.GetConn().Query(`
// 			SELECT
// 				station_id, flag, COUNT(1)
// 			FROM
// 				`+table+`
// 			WHERE
// 				`+strings.Join(whereStmt, " AND ")+`
// 			GROUP BY
// 				station_id, flag
// 		`, values...)
// 		if err != nil {
// 			log.Println("error count station data flag: ", err)
// 			return nil, err
// 		}

// 		defer rows.Close()

// 		for rows.Next() {
// 			var f string
// 			var s, c int

// 			rows.Scan(&s, &f, &c)

// 			if _, exists := result[s]; !exists {
// 				result[s] = &[3]int{}
// 			}

// 			flag, exists := flags[f]
// 			if exists {
// 				if flag.IsEffective {
// 					result[s][0] += c
// 				} else if flag.IsOutOfService {
// 					result[s][1] += c
// 				}
// 			}

// 			result[s][2] += c
// 		}
// 	}

// 	return result, nil
// }
