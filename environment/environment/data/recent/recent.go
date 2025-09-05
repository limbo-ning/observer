package recent

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
)

var recentCache map[string]*sitePool
var recentLock sync.RWMutex

type sitePool struct {
	lock sync.RWMutex
	pool map[string]*dataTypePool
}

type dataTypePool struct {
	lock sync.RWMutex
	pool map[int]*stationPool
}

type stationPool struct {
	lock sync.RWMutex
	pool map[int]data.IData
}

func (p *stationPool) clone() map[int]data.IData {
	p.lock.RLock()
	defer p.lock.RUnlock()

	result := make(map[int]data.IData)
	for k, v := range p.pool {
		result[k] = v
	}
	return result
}

func ClearCache(siteID string, stationID int) error {
	recentLock.RLock()
	defer recentLock.RUnlock()

	site, exists := recentCache[siteID]
	if !exists {
		return nil
	}

	site.lock.RLock()
	defer site.lock.RUnlock()

	for _, stationPool := range site.pool {
		stationPool.lock.Lock()
		delete(stationPool.pool, stationID)
		stationPool.lock.Unlock()
	}

	return nil
}

func GetRecentData(siteID string, actionAuth authority.ActionAuthSet, dataType string, stationID ...int) (result map[int]map[int]data.IData, e error) {

	defer func() {

		fetchedStationIDs := make([]int, 0)
		for sid := range result {
			fetchedStationIDs = append(fetchedStationIDs, sid)
		}

		stationMonitors, err := monitor.GetStationMonitors(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, fetchedStationIDs...)
		if err != nil {
			log.Println("error get station monitor")
			return
		}

		for _, sid := range stationID {

			entry := result[sid]

			monitorIDs := stationMonitors[sid]
			filtered := make(map[int]data.IData)

			for _, mid := range monitorIDs {
				d := entry[mid]
				if d != nil {
					filtered[mid] = d
				}
			}

			result[sid] = filtered
		}

	}()

	result = make(map[int]map[int]data.IData)

	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, stationID, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return nil, err
	}

	stationID = make([]int, 0)
	for f, ok := range filtered {
		if ok {
			stationID = append(stationID, f)
		}
	}

	if len(stationID) == 0 {
		return result, nil
	}

	recentLock.RLock()
	site, exists := recentCache[siteID]
	recentLock.RUnlock()
	if !exists {
		recentLock.Lock()
		site, exists = recentCache[siteID]
		defer recentLock.Unlock()
		if !exists {
			return fetchRecentData(siteID, dataType, result, stationID...)
		}
	}

	site.lock.RLock()
	data, exists := site.pool[dataType]
	site.lock.RUnlock()
	if !exists {
		site.lock.Lock()
		defer site.lock.Unlock()
		data, exists = site.pool[dataType]
		if !exists {
			return fetchRecentData(siteID, dataType, result, stationID...)
		}
	}

	data.lock.RLock()
	toFetch := make([]int, 0)
	for _, id := range stationID {
		station, exists := data.pool[id]
		if !exists {
			toFetch = append(toFetch, id)
			continue
		}
		result[id] = station.clone()
	}
	data.lock.RUnlock()

	if len(toFetch) == 0 {
		return result, nil
	}

	return fetchRecentData(siteID, dataType, result, toFetch...)
}

func UpdateRecentData(siteID string, d data.IData) (isUpdated bool, site *sitePool, dataPool *dataTypePool, station *stationPool) {
	exists := false
	defer func() {

		if site != nil && dataPool != nil && station != nil {
			return
		}

		log.Println("defer generate recent new entry: ", siteID, d.GetDataType(), d.GetStationID())

		if site == nil {
			recentLock.Lock()
			defer recentLock.Unlock()
			if recentCache == nil {
				recentCache = make(map[string]*sitePool)
			}
			site, exists = recentCache[siteID]
			if !exists {
				site = &sitePool{pool: make(map[string]*dataTypePool)}
				recentCache[siteID] = site
			}
		} else {
			recentLock.RLock()
			defer recentLock.RUnlock()
		}

		if dataPool == nil {
			site.lock.Lock()
			defer site.lock.Unlock()

			dataPool, exists = site.pool[d.GetDataType()]
			if !exists {
				dataPool = &dataTypePool{pool: make(map[int]*stationPool)}
				site.pool[d.GetDataType()] = dataPool
			}
		} else {
			site.lock.RLock()
			defer site.lock.RUnlock()
		}

		if station == nil {
			dataPool.lock.Lock()
			defer dataPool.lock.Unlock()

			station, exists = dataPool.pool[d.GetStationID()]
			if !exists {
				station = &stationPool{pool: make(map[int]data.IData)}
				dataPool.pool[d.GetStationID()] = station
			}

			if d.GetMonitorID() > 0 {
				station.lock.Lock()
				defer station.lock.Unlock()

				if station.pool[d.GetMonitorID()] == nil || time.Time(station.pool[d.GetMonitorID()].GetDataTime()).Before(time.Time(d.GetDataTime())) {
					isUpdated = true
					station.pool[d.GetMonitorID()] = d
				}
			}
		} else {
			dataPool.lock.RLock()
			defer dataPool.lock.RUnlock()
		}
	}()

	recentLock.RLock()
	defer recentLock.RUnlock()
	site, exists = recentCache[siteID]
	if !exists {
		return
	}

	site.lock.RLock()
	defer site.lock.RUnlock()
	dataPool, exists = site.pool[d.GetDataType()]
	if !exists {
		return
	}

	dataPool.lock.RLock()
	defer dataPool.lock.RUnlock()
	station, exists = dataPool.pool[d.GetStationID()]
	if !exists {
		return
	}

	sm, err := data.GetModule(siteID)
	if err != nil {
		return
	}

	if sm.MonitorField == "monitor_id" {
		if d.GetMonitorID() > 0 {
			station.lock.Lock()
			defer station.lock.Unlock()
			if station.pool[d.GetMonitorID()] == nil || time.Time(station.pool[d.GetMonitorID()].GetDataTime()).Before(time.Time(d.GetDataTime())) {
				station.pool[d.GetMonitorID()] = d
				isUpdated = true
			}
		}
	} else {
		if d.GetMonitorCodeID() > 0 {
			station.lock.Lock()
			defer station.lock.Unlock()
			if station.pool[d.GetMonitorCodeID()] == nil || time.Time(station.pool[d.GetMonitorCodeID()].GetDataTime()).Before(time.Time(d.GetDataTime())) {
				station.pool[d.GetMonitorCodeID()] = d
				isUpdated = true
			}
		}
	}

	return
}

func fetchRecentData(siteID, dataType string, appendix map[int]map[int]data.IData, stationID ...int) (map[int]map[int]data.IData, error) {

	if len(stationID) == 0 {
		return appendix, nil
	}

	sm, err := data.GetModule(siteID)
	if err != nil {
		return nil, err
	}

	table := data.TableName(siteID, dataType)

	var columns []string
	var traceTime time.Time

	var instance func() data.IData

	switch dataType {
	case data.REAL_TIME:
		columns = append(data.SelectColumn(siteID), data.RTD)
		traceTime = time.Now().Add(-1 * time.Hour * 24)
		instance = func() data.IData { return new(data.RealTimeData) }
	case data.MINUTELY:
		columns = append(data.SelectColumn(siteID), data.IntervalColumn...)
		traceTime = time.Now().Add(-1 * time.Hour * 24)
		instance = func() data.IData { return new(data.MinutelyData) }
	case data.HOURLY:
		columns = append(data.SelectColumn(siteID), data.IntervalColumn...)
		traceTime = time.Now().Add(-1 * time.Hour * 24 * 7)
		instance = func() data.IData { return new(data.HourlyData) }
	case data.DAILY:
		columns = append(data.SelectColumn(siteID), data.IntervalColumn...)
		traceTime = time.Now().Add(-1 * time.Hour * 24 * 31)
		instance = func() data.IData { return new(data.DailyData) }
	}

	fields := make([]string, len(columns))
	for i, c := range columns {
		fields[i] = fmt.Sprintf("data.%s", c)
	}

	values := make([]interface{}, 0)
	values = append(values, traceTime)

	queryIDs := make(map[int]byte)
	ids := make([]any, 0)
	var idField string
	if len(stationID) == 1 {
		idField = " = ?"
		queryIDs[stationID[0]] = 1
	} else {
		placeholder := make([]string, 0)
		for _, id := range stationID {
			if _, exists := queryIDs[id]; !exists {
				placeholder = append(placeholder, "?")
			}
			queryIDs[id] = 1
		}
		idField = " IN (" + strings.Join(placeholder, ",") + ")"
	}

	for id := range queryIDs {
		ids = append(ids, id)
	}

	values = append(values, ids...)
	values = append(values, ids...)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s data
		JOIN
		(
			SELECT
				MAX(id) as id
			FROM
				%s
			WHERE
				%s >= ? AND %s %s
			GROUP BY %s, %s
		) max
		ON
			data.id = max.id
		WHERE
			%s %s
	`, strings.Join(fields, ","), table, table, data.DATA_TIME, data.STATION_ID, idField, sm.MonitorField, data.STATION_ID, data.STATION_ID, idField)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get recent data: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	list := make([]data.IData, 0)

	for rows.Next() {
		d := instance()
		if data.Scan(siteID, rows, d, columns); err != nil {
			log.Println("error scan recent data: ", err)
			return nil, err
		}

		list = append(list, d)

		delete(queryIDs, d.GetStationID())

		if _, exists := appendix[d.GetStationID()]; !exists {
			appendix[d.GetStationID()] = make(map[int]data.IData)
		}

		if sm.MonitorField == "monitor_id" {
			if d.GetMonitorID() > 0 {
				appendix[d.GetStationID()][d.GetMonitorID()] = d
			}
		} else {
			if d.GetMonitorCodeID() > 0 {
				appendix[d.GetStationID()][d.GetMonitorCodeID()] = d
			}
		}
	}

	go func() {
		for _, d := range list {
			UpdateRecentData(siteID, d)
		}
		for empty := range queryIDs {
			d := instance()
			d.SetStationID(empty)
			UpdateRecentData(siteID, d)
		}
	}()

	return appendix, nil
}
