package stats

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/event"
)

const EVENT_ENVIRONMENT_HISTORY_DATAQUALITY = "environment_history_dataquality"

func init() {
	event.Register(EVENT_ENVIRONMENT_HISTORY_DATAQUALITY, func() event.IEvent {
		return new(HistoryDataQualityEvent)
	})
}

type HistoryDataQualityEvent struct{}

func (e *HistoryDataQualityEvent) ValidateEvent(siteID string, eventInstance *event.Event) error {

	intervalType := eventInstance.MainRelateID

	switch intervalType {
	case HISTORY_STATS_INTERVAL_DAILY:
	default:
		return errors.New("未知时间周期")
	}

	return nil
}

func (e *HistoryDataQualityEvent) ExecuteEvent(siteID string, txn *sql.Tx, eventInstance *event.Event) error {

	intervalType := eventInstance.MainRelateID

	var isReplace bool
	if replace, exists := eventInstance.SubRelateID["replace"]; exists {
		isReplace, _ = strconv.ParseBool(replace)
	}

	var beginTime time.Time
	var endTime time.Time

	switch intervalType {
	case HISTORY_STATS_INTERVAL_DAILY:
		beginTime = util.GetDate(time.Now().AddDate(0, 0, -1))
		endTime = beginTime.AddDate(0, 0, 1)
	default:
		return errors.New("未知时间周期")
	}

	stationDataQualities, err := GetStationDataQuality(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, &beginTime, &endTime)
	if err != nil {
		return err
	}

	var saved int

	for sid, dataQuality := range stationDataQualities {
		history := new(HistoryStats)
		history.StationID = sid
		history.Type = HISTORY_STATS_DATA_QUALITY
		history.IntervalType = intervalType
		history.StatsTime = util.Time(beginTime)
		history.Stats = make(map[string]interface{})
		history.Stats["dataQuality"] = dataQuality

		if isReplace {
			if err := history.addUpdate(siteID, txn); err != nil {
				return err
			}
		} else {
			if err := history.add(siteID, txn); err != nil {
				if err == e_duplicate {
					log.Println("历史记录重复:", sid, history.Type, history.IntervalType, history.StatsTime)
					continue
				}
				return err
			}
		}

		saved++
	}

	eventInstance.Feedback(event.SUCCESS, map[string]interface{}{
		"at":    util.Time(time.Now()),
		"saved": saved,
	})

	if err := eventInstance.UpdateStatusWithTxn(siteID, txn); err != nil {
		log.Println("error update event after finish data quality stats: ", err)
	}

	return nil
}
