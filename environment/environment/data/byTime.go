package data

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
)

type TimeData struct {
	DataTime util.Time       `json:"dataTime"`
	Data     map[int][]IData `json:"data"`
}

var e_need_datatime = errors.New("需要时间")
var e_real_time_range_restricted = errors.New("实时数据的查询时间跨度最多48小时")
var e_minutely_range_restricted = errors.New("分钟数据的查询时间跨度最多48小时")
var e_hourly_range_restricted = errors.New("小时数据的查询时间跨度最多31天")
var e_daily_range_restricted = errors.New("日数据查询跨度最多365天")

func GetDataByTime(siteID string, dataType string, stationID []int, criterias Criterias, beginTime, endTime time.Time, withOriginData, withReviewed bool, order string, pageNo, pageSize int, monitorCodeID []int, monitorID []int) ([]*TimeData, int, error) {

	log.Println("get data by time: ", siteID, dataType, stationID, criterias, beginTime, endTime, pageNo, pageSize)

	result := make([]*TimeData, 0)

	if len(stationID) == 0 {
		return result, 0, nil
	}

	if beginTime.IsZero() || endTime.IsZero() {
		return nil, 0, e_need_datatime
	}

	if beginTime.Equal(endTime) || beginTime.After(endTime) {
		return result, 0, nil
	}

	m, err := GetModule(siteID)
	if err != nil {
		return nil, 0, err
	}

	switch strings.ToUpper(order) {
	case "ASC":
	case "DESC":
	default:
		order = "ASC"
	}

	var columns []string
	var instance func() IData

	switch dataType {
	case REAL_TIME:
		columns = append(SelectColumn(siteID), RTD)
		instance = func() IData { return new(RealTimeData) }
		if endTime.Sub(beginTime).Hours() > 48 {
			return nil, 0, e_real_time_range_restricted
		}
	case MINUTELY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(MinutelyData) }
		if endTime.Sub(beginTime).Hours() > 48 {
			return nil, 0, e_minutely_range_restricted
		}
	case HOURLY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(HourlyData) }
		if endTime.Sub(beginTime).Hours() > 24*32 {
			return nil, 0, e_hourly_range_restricted
		}
	case DAILY:
		columns = append(SelectColumn(siteID), IntervalColumn...)
		instance = func() IData { return new(DailyData) }
		if endTime.Sub(beginTime).Hours() > 24*365 {
			return nil, 0, e_daily_range_restricted
		}
	default:
		return nil, 0, e_invalid_data_type
	}

	effectiveBegin, effectveEnd, total, err := getDataTimes(siteID, dataType, stationID, criterias, beginTime, endTime, order, pageNo, pageSize, monitorCodeID, monitorID)
	if err != nil {
		return nil, total, err
	}

	if effectiveBegin == nil {
		return result, total, nil
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
	values = append(values, effectiveBegin, effectveEnd)

	if withOriginData {
		columns = append(columns, ORIGIN_DATA)
	}

	if withReviewed {
		if _, ok := instance().(IReview); ok {
			columns = append(columns, REVIEWED)
		}
	}

	fields := make([]string, len(columns))
	for i, c := range columns {
		fields[i] = fmt.Sprintf("data.%s", c)
	}

	tables := FetchTableNames(siteID, dataType, *effectiveBegin, *effectveEnd)

	log.Println("tables: ", siteID, len(tables))

	filtered := make([]IData, 0)
	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				%s
			FROM
				%s data
			WHERE
				%s
			ORDER BY data.%s %s
		`, strings.Join(fields, ","), table, strings.Join(whereStmts, " AND "), DATA_TIME, order)

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error get data by time: ", err)
			return nil, total, err
		}
		defer rows.Close()

		for rows.Next() {
			d := instance()
			if Scan(siteID, rows, d, columns); err != nil {
				log.Println("error get dat data by time: ", err)
				return nil, total, err
			}
			filtered = append(filtered, d)
		}
	}

	log.Println("to filter: ", siteID, len(filtered))

	dataTimeMapping := make(map[string]*TimeData)

	for _, d := range filtered {
		dataTimeStr := util.FormatDateTime(time.Time(d.GetDataTime()))

		if _, exists := dataTimeMapping[dataTimeStr]; !exists {
			timeData := new(TimeData)
			timeData.DataTime = d.GetDataTime()
			timeData.Data = make(map[int][]IData)
			dataTimeMapping[dataTimeStr] = timeData
		}
	}

	if len(criterias) > 0 {
		filtered = criterias.FilterData(filtered, true)
	}

	for _, d := range filtered {
		dataTimeStr := util.FormatDateTime(time.Time(d.GetDataTime()))
		timeData := dataTimeMapping[dataTimeStr]
		if _, exists := timeData.Data[d.GetStationID()]; !exists {
			timeData.Data[d.GetStationID()] = make([]IData, 0)
		}
		timeData.Data[d.GetStationID()] = append(timeData.Data[d.GetStationID()], d)
	}

	for _, d := range dataTimeMapping {
		result = append(result, d)
	}

	if pageSize == -1 {
		total = len(result)
	}

	log.Println("return : ", siteID, len(result), total)

	return result, total, nil
}

func getDataTimes(siteID, dataType string, stationID []int, criterias Criterias, beginTime, endTime time.Time, order string, pageNo, pageSize int, monitorCodeID []int, monitorID []int) (*time.Time, *time.Time, int, error) {

	total := 0

	if pageSize == -1 {
		return &beginTime, &endTime, total, nil
	}

	m, err := GetModule(siteID)
	if err != nil {
		return nil, nil, 0, err
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

	tables := FetchTableNames(siteID, dataType, beginTime, endTime)

	timeSlots := make([]*time.Time, 0)

	for _, table := range tables {
		SQL := fmt.Sprintf(`
			SELECT
				DISTINCT %s
			FROM
				%s data
			WHERE
				%s
			ORDER BY data.%s %s
		`, DATA_TIME, table, strings.Join(whereStmts, " AND "), DATA_TIME, order)

		rows, err := datasource.GetConn().Query(SQL, values...)
		if err != nil {
			log.Println("error get data time slots by time: ", err)
			return nil, nil, total, err
		}
		defer rows.Close()

		for rows.Next() {
			var dataTime time.Time

			if err := rows.Scan(&dataTime); err != nil {
				return nil, nil, total, err
			}

			timeSlots = append(timeSlots, &dataTime)
		}
	}

	sort.Slice(timeSlots, func(i, j int) bool {
		if order == "ASC" {
			return timeSlots[i].Before(*timeSlots[j])
		} else {
			return timeSlots[j].Before(*timeSlots[i])
		}
	})

	total = len(timeSlots)

	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var begin, end *time.Time

	if (pageNo-1)*pageSize < len(timeSlots) {
		begin = timeSlots[(pageNo-1)*pageSize]
	} else {
		return nil, nil, total, nil
	}

	if pageNo*pageSize-1 < len(timeSlots) {
		end = timeSlots[pageNo*pageSize-1]
	} else {
		end = timeSlots[len(timeSlots)-1]
	}

	log.Println("time slots: ", siteID, begin, end, order)

	if order == "ASC" {
		return begin, end, total, nil
	}
	return end, begin, total, nil
}
