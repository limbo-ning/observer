package stats

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/entity"
	"strings"
	"time"
)

const (
	HISTORY_STATS_DATA_QUALITY = "dataquality"

	HISTORY_STATS_INTERVAL_DAILY = "daily"
)

var e_duplicate = errors.New("重复")

type HistoryStats struct {
	ID           int                    `json:"ID"`
	StationID    int                    `json:"stationID"`
	StatsTime    util.Time              `json:"statsTime"`
	Type         string                 `json:"type"`
	IntervalType string                 `json:"intervalType"`
	Stats        map[string]interface{} `json:"stats"`
}

const historyStatsColumn = "stats.id,stats.station_id,stats.stats_time,stats.type,stats.interval_type,stats.stats"

func historyStatsTable(siteID string) string {
	return siteID + "_historystats"
}

func (h *HistoryStats) scan(rows *sql.Rows) error {

	var stats string

	if err := rows.Scan(&h.ID, &h.StationID, &h.StatsTime, &h.Type, &h.IntervalType, &stats); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(stats), &h.Stats); err != nil {
		return err
	}

	return nil
}

func (h *HistoryStats) validate() error {
	switch h.Type {
	case HISTORY_STATS_DATA_QUALITY:
	default:
		return errors.New("未知历史数据")
	}

	switch h.IntervalType {
	case HISTORY_STATS_INTERVAL_DAILY:
	default:
		return errors.New("未知周期")
	}

	if time.Time(h.StatsTime).IsZero() {
		return errors.New("无历史数据时间")
	}

	if h.Stats == nil || len(h.Stats) == 0 {
		return errors.New("无历史数据内容")
	}
	return nil
}

func (h *HistoryStats) add(siteID string, txn *sql.Tx) error {

	if err := h.validate(); err != nil {
		return err
	}

	stats, _ := json.Marshal(&h.Stats)

	if ret, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(station_id,stats_time,type,interval_type,stats)
		VALUES
			(?,?,?,?,?)
	`, historyStatsTable(siteID)), h.StationID, time.Time(h.StatsTime), h.Type, h.IntervalType, string(stats)); err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return e_duplicate
		}

		log.Println("error add history stats: ", err)

		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add history stats get id: ", err)
		return err
	} else {
		h.ID = int(id)
	}

	return nil
}

func (h *HistoryStats) addUpdate(siteID string, txn *sql.Tx) error {
	if err := h.validate(); err != nil {
		return err
	}

	stats, _ := json.Marshal(&h.Stats)

	if ret, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(station_id,stats_time,type,interval_type,stats)
		VALUES
			(?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			station_id=VALUES(station_id), stats_time=VALUES(stats_time), type=VALUES(type), interval_type=VALUES(interval_type), stats=VALUES(stats)
	`, historyStatsTable(siteID)), h.StationID, time.Time(h.StatsTime), h.Type, h.IntervalType, string(stats)); err != nil {
		log.Println("error add history stats: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add history stats get id: ", err)
		return err
	} else {
		h.ID = int(id)
	}

	return nil
}

func GetHistoryStats(siteID string, actionAuth authority.ActionAuthSet, statsType, intervalType string, beginTime, endTime time.Time, stationID ...int) (map[int][]*HistoryStats, error) {

	result := make(map[int][]*HistoryStats)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	switch statsType {
	case HISTORY_STATS_DATA_QUALITY:
	default:
		return nil, errors.New("未知历史数据")
	}

	switch intervalType {
	case HISTORY_STATS_INTERVAL_DAILY:
	default:
		return nil, errors.New("未知周期")
	}

	if beginTime.IsZero() || endTime.IsZero() {
		return nil, errors.New("无历史数据时间")
	}

	whereStmts = append(whereStmts, "stats.type = ?", "stats.interval_type = ?", "stats.stats_time >= ?", "stats.stats_time <= ?")
	values = append(values, statsType, intervalType, beginTime, endTime)

	if len(stationID) > 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "stats.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("stats.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s stats
		WHERE
			%s
	`, historyStatsColumn, historyStatsTable(siteID), strings.Join(whereStmts, " AND "))

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get history stats: ", err)
		return nil, err
	}

	defer rows.Close()

	stationIDs := make([]int, 0)
	for rows.Next() {
		var h HistoryStats
		if err := h.scan(rows); err != nil {
			log.Println("error scan history stats: ", err)
			return nil, err
		}

		list, exists := result[h.StationID]
		if !exists {
			stationIDs = append(stationIDs, h.StationID)
			list = make([]*HistoryStats, 0)
		}
		list = append(list, &h)
		result[h.StationID] = list
	}

	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, stationIDs, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return nil, err
	}

	for _, sid := range stationIDs {
		if !filtered[sid] {
			delete(result, sid)
		}
	}

	return result, nil
}
