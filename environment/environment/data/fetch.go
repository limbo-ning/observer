package data

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment/entity"
)

func wrapByStation(result map[int]interface{}, count int, stationID int, monitorID int, flag string, groupByMonitor, groupByFlag bool) {

	entry := result[stationID]

	if groupByMonitor {
		if entry == nil {
			entry = make(map[int]interface{})
		}
		wrapByMonitor(entry.(map[int]interface{}), count, monitorID, flag, groupByFlag)
	} else if groupByFlag {
		if entry == nil {
			entry = make(map[string]interface{})
		}
		wrapByFlag(entry.(map[string]interface{}), count, flag)
	} else {
		if entry == nil {
			entry = 0
		}
		entry = entry.(int) + count
	}

	result[stationID] = entry
}

func wrapByMonitor(result map[int]interface{}, count int, monitorID int, flag string, groupByFlag bool) {
	entry := result[monitorID]

	if groupByFlag {
		if entry == nil {
			entry = make(map[string]interface{})
		}
		wrapByFlag(entry.(map[string]interface{}), count, flag)
	} else {
		if entry == nil {
			entry = 0
		}
		entry = entry.(int) + count
	}

	result[monitorID] = entry
}

func wrapByFlag(result map[string]interface{}, count int, flag string) {
	entry := result[flag]
	if entry == nil {
		entry = 0
	}
	entry = entry.(int) + count

	result[flag] = entry
}

func CountData(siteID, dataType string, stationID, monitorID, monitorCodeID []int, beginTime, endTime time.Time, flag []string, criterias Criterias, groupByTime, groupByStation, groupByMonitor, groupByFlag bool) (result interface{}, err error) {

	defer func() {
		if result == nil {
			if groupByStation || groupByMonitor || groupByFlag {
				result = make(map[int]interface{})
			} else {
				result = 0
			}
		}
	}()

	if len(stationID) == 0 {
		return result, nil
	}

	if beginTime.IsZero() || endTime.IsZero() {
		return result, e_need_datatime
	}

	if beginTime.Equal(endTime) || beginTime.After(endTime) {
		return result, nil
	}

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(stationID) == 1 {
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", STATION_ID))
		values = append(values, stationID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range stationID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", STATION_ID, strings.Join(placeholder, ",")))
	}

	if m.MonitorField == MONITOR_CODE_ID && len(monitorCodeID) > 0 {
		if len(monitorCodeID) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", MONITOR_CODE_ID))
			values = append(values, monitorCodeID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorCodeID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", MONITOR_CODE_ID, strings.Join(placeholder, ",")))
		}
	}

	if m.MonitorField == MONITOR_ID && len(monitorID) > 0 {
		if len(monitorID) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", MONITOR_ID))
			values = append(values, monitorID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", MONITOR_ID, strings.Join(placeholder, ",")))
		}
	}

	whereStmts = append(whereStmts, fmt.Sprintf("data.%s >= ? AND data.%s <= ?", DATA_TIME, DATA_TIME))
	values = append(values, beginTime, endTime)

	if len(flag) > 0 {
		if len(flag) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", FLAG))
			values = append(values, flag[0])
		} else {
			placeholder := make([]string, 0)
			for _, f := range flag {
				placeholder = append(placeholder, "?")
				values = append(values, f)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", FLAG, strings.Join(placeholder, ",")))
		}
	}

	if len(criterias) > 0 {
		subWhere, subValues := criterias.ParseSQL(dataType, "data")
		whereStmts = append(whereStmts, subWhere)
		values = append(values, subValues...)
	}

	tables := FetchTableNames(siteID, dataType, beginTime, endTime)

	var field string
	var groupBy string

	if groupByTime {
		field = fmt.Sprintf("COUNT(DISTINCT data.%s)", DATA_TIME)
	} else {
		field = "COUNT(1)"
	}

	if groupByStation {
		field += fmt.Sprintf(", data.%s", STATION_ID)
		groupBy += fmt.Sprintf("GROUP BY data.%s", STATION_ID)
	}

	if groupByMonitor {
		field += fmt.Sprintf(", data.%s", m.MonitorField)
		if groupBy == "" {
			groupBy += "GROUP BY "
		} else {
			groupBy += ","
		}
		groupBy += fmt.Sprintf("data.%s", m.MonitorField)
	}

	if groupByFlag {
		field += fmt.Sprintf(", data.%s", FLAG)
		if groupBy == "" {
			groupBy += "GROUP BY "
		} else {
			groupBy += ","
		}
		groupBy += fmt.Sprintf("data.%s", FLAG)
	}

	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				%s
			FROM
				%s data
			WHERE
				%s
			%s
		`, field, table, strings.Join(whereStmts, " AND "), groupBy)

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error count data: ", err)
			return 0, err
		}
		defer rows.Close()

		for rows.Next() {
			var count, stationID, mid int
			var flag string

			dest := make([]interface{}, 0)
			dest = append(dest, &count)

			if groupByStation {
				dest = append(dest, &stationID)
			}
			if groupByMonitor {
				dest = append(dest, &mid)
			}
			if groupByFlag {
				dest = append(dest, &flag)
			}
			if err := rows.Scan(dest...); err != nil {
				return result, err
			}

			if groupByStation {
				if result == nil {
					result = make(map[int]interface{})
				}
				wrapByStation(result.(map[int]interface{}), count, stationID, mid, flag, groupByMonitor, groupByFlag)
			} else if groupByMonitor {
				if result == nil {
					result = make(map[int]interface{})
				}
				wrapByMonitor(result.(map[int]interface{}), count, mid, flag, groupByFlag)
			} else if groupByFlag {
				if result == nil {
					result = make(map[string]interface{})
				}
				wrapByFlag(result.(map[string]interface{}), count, flag)
			} else {
				if result == nil {
					result = 0
				}
				result = result.(int) + count
			}
		}
	}

	return result, nil
}

func GetData(siteID, dataType string, stationID, monitorID, monitorCodeID []int, criterias Criterias, beginTime, endTime time.Time, flag []string, extraColumn ...string) ([]IData, error) {
	result := make([]IData, 0)

	if len(stationID) == 0 {
		return result, nil
	}

	if beginTime.IsZero() || endTime.IsZero() {
		return nil, e_need_datatime
	}

	if beginTime.After(endTime) {
		return result, nil
	}

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(stationID) == 1 {
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", STATION_ID))
		values = append(values, stationID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range stationID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", STATION_ID, strings.Join(placeholder, ",")))
	}

	if m.MonitorField == MONITOR_CODE_ID && len(monitorCodeID) > 0 {
		if len(monitorCodeID) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", MONITOR_CODE_ID))
			values = append(values, monitorCodeID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorCodeID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", MONITOR_CODE_ID, strings.Join(placeholder, ",")))
		}
	}

	if m.MonitorField == MONITOR_ID && len(monitorID) > 0 {
		if len(monitorID) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", MONITOR_ID))
			values = append(values, monitorID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", MONITOR_ID, strings.Join(placeholder, ",")))
		}
	}

	if beginTime.Equal(endTime) {
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", DATA_TIME))
		values = append(values, beginTime)
	} else {
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s >= ? AND data.%s <= ?", DATA_TIME, DATA_TIME))
		values = append(values, beginTime, endTime)
	}

	if len(flag) > 0 {
		if len(flag) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", FLAG))
			values = append(values, flag[0])
		} else {
			placeholder := make([]string, 0)
			for _, f := range flag {
				placeholder = append(placeholder, "?")
				values = append(values, f)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", FLAG, strings.Join(placeholder, ",")))
		}
	}

	if len(criterias) > 0 {
		subSQL, subValues := criterias.ParseSQL(dataType, "data")
		if subSQL != "" {
			whereStmts = append(whereStmts, subSQL)
			values = append(values, subValues...)
		}
	}

	var columns []string
	var instance func() IData

	switch dataType {
	case REAL_TIME:
		columns = append(SelectColumn(siteID), RTD)
		instance = func() IData { return new(RealTimeData) }
	case MINUTELY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(MinutelyData) }
	case HOURLY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(HourlyData) }
	case DAILY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(DailyData) }
	default:
		return nil, e_invalid_data_type
	}

	columns = append(columns, extraColumn...)

	fields := make([]string, len(columns))
	for i, c := range columns {
		fields[i] = fmt.Sprintf("data.%s", c)
	}

	tables := FetchTableNames(siteID, dataType, beginTime, endTime)

	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				%s
			FROM
				%s data
			WHERE
				%s
		`, strings.Join(fields, ","), table, strings.Join(whereStmts, " AND "))

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error get data: ", err)
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			d := instance()
			if Scan(siteID, rows, d, columns); err != nil {
				log.Println("error get data: ", err)
				return nil, err
			}

			result = append(result, d)
		}
	}

	return result, nil
}

func getStep(dataType string) time.Duration {
	var step time.Duration

	switch dataType {
	case REAL_TIME:
		step = time.Minute
	case MINUTELY:
		step = time.Minute * 10
	case HOURLY:
		step = time.Hour
	case DAILY:
		step = time.Hour * 24
	}
	return step
}

func getDataTimeEntries(dataType string, beginTime, endTime time.Time) []time.Time {
	result := make([]time.Time, 0)

	step := getStep(dataType)
	stepper := util.TruncateLocal(beginTime, step)
	endTime = util.TruncateLocal(endTime, step)

	for {
		if !stepper.Before(endTime) {
			break
		}
		if !beginTime.After(stepper) {
			result = append(result, time.Time(stepper))
		}
		stepper = stepper.Add(step)
	}

	return result
}

func GetDataVacancy(siteID, dataType string, stationID []int, beginTime, endTime time.Time) (map[int][][2]time.Time, error) {

	result := make(map[int][][2]time.Time)

	tables := FetchTableNames(siteID, dataType, beginTime, endTime)

	log.Println("get data time vacancy: ", len(tables))

	if len(stationID) == 0 {
		return result, nil
	}

	if beginTime.IsZero() || endTime.IsZero() {
		return nil, e_need_datatime
	}

	if beginTime.Equal(endTime) || beginTime.After(endTime) {
		return result, nil
	}

	dataTimeEntries := getDataTimeEntries(dataType, beginTime, endTime)
	if len(dataTimeEntries) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(stationID) == 1 {
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", STATION_ID))
		values = append(values, stationID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range stationID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", STATION_ID, strings.Join(placeholder, ",")))
	}

	whereStmts = append(whereStmts, fmt.Sprintf("data.%s >= ? AND data.%s <= ?", DATA_TIME, DATA_TIME))
	values = append(values, beginTime, endTime)

	timeSlots := make(map[int]map[string]int)
	for _, stationID := range stationID {
		timeSlots[stationID] = make(map[string]int)
		for i, t := range dataTimeEntries {
			timeSlots[stationID][util.FormatDateTime(t)] = i
		}
	}

	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				DISTINCT data.%s, data.%s
			FROM
				%s data
			WHERE
				%s
		`, DATA_TIME, STATION_ID, table, strings.Join(whereStmts, " AND "))

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error get data: ", err)
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var stationID int
			var dataTime time.Time
			if err := rows.Scan(&dataTime, &stationID); err != nil {
				return nil, err
			}

			ts := util.FormatDateTime(dataTime)
			tsMap := timeSlots[stationID]
			delete(tsMap, ts)
		}
	}

	needTraceBackStationIDs := make([]int, 0)

	for stationID, tsMap := range timeSlots {
		if len(tsMap) > 0 {
			result[stationID] = make([][2]time.Time, 0)

			keys := make([]string, 0)
			for k := range tsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			currentIndex := -1
			var slot *[2]time.Time

			for _, k := range keys {
				if slot == nil {
					currentIndex = tsMap[k]
					slot = new([2]time.Time)
					slot[0] = dataTimeEntries[currentIndex]
				} else {
					if tsMap[k] == currentIndex+1 {
						currentIndex = tsMap[k]
					} else {
						slot[1] = dataTimeEntries[currentIndex]
						result[stationID] = append(result[stationID], *slot)
						slot = new([2]time.Time)
						currentIndex = tsMap[k]
						slot[0] = dataTimeEntries[currentIndex]
					}
				}
			}

			if tsMap[keys[0]] == 0 {
				needTraceBackStationIDs = append(needTraceBackStationIDs, stationID)
			}

			if slot != nil {
				slot[1] = dataTimeEntries[currentIndex]
				result[stationID] = append(result[stationID], *slot)
			}
		}
	}

	step := getStep(dataType)

	log.Println("get data time entries")

	if len(needTraceBackStationIDs) > 0 {

		traceTimes, err := getMaxDataTime(siteID, dataType, needTraceBackStationIDs, beginTime)
		if err != nil {
			return nil, err
		}

		cantTraceStations := make([]int, 0)

		for _, stationID := range needTraceBackStationIDs {
			traceTime, exists := traceTimes[stationID]
			slots := result[stationID]
			if len(slots) > 0 {
				if exists {
					if traceTime.Before(beginTime) {
						slots[0][0] = traceTime.Add(step)
					}
				} else {
					cantTraceStations = append(cantTraceStations, stationID)
				}
			}
		}

		if len(cantTraceStations) > 0 {
			stationList, err := entity.GetStation(siteID, cantTraceStations...)
			if err != nil {
				return nil, err
			}

			for _, station := range stationList {
				slots := result[station.ID]
				if len(slots) > 0 {
					slots[0][0] = time.Time(station.OnlineTime)
				}
			}
		}
	}

	log.Println("get data time trace back")

	return result, nil
}

func getMaxDataTime(siteID, dataType string, stationID []int, beforeTime time.Time) (map[int]time.Time, error) {

	result := make(map[int]time.Time)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(stationID) > 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s = ?", STATION_ID))
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("data.%s IN (%s)", STATION_ID, strings.Join(placeholder, ",")))
		}
	} else {
		return result, nil
	}

	whereStmts = append(whereStmts, fmt.Sprintf("data.%s < ?", DATA_TIME))
	values = append(values, beforeTime)

	SQL := fmt.Sprintf(`
		SELECT
			MAX(data.%s), data.%s
		FROM
			%s data
		WHERE
			%s
		GROUP BY
			data.%s
	`, DATA_TIME, STATION_ID, TableName(siteID, dataType), strings.Join(whereStmts, " AND "), STATION_ID)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get max data time: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stationID int
		var dataTime time.Time
		if err := rows.Scan(&dataTime, &stationID); err != nil {
			return nil, err
		}

		result[stationID] = dataTime
	}

	return result, nil
}
