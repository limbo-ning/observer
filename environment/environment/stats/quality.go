package stats

import (
	"errors"
	"log"
	"time"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
)

var e_need_datatime = errors.New("需要时间")

type QualityRates struct {
	TransRate       float64 `json:"transRate"`
	EffectRate      float64 `json:"effectRate"`
	EffectTransRate float64 `json:"effectTransRate"`
	TransCount      int     `json:"transCount"`
	EffectCount     int     `json:"effectCount"`
	SlotCount       int     `json:"slotCount"`
}

func calculateRates(siteID string, slot int, counts map[string]int) (*QualityRates, error) {

	sm, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	result := &QualityRates{}
	result.SlotCount = slot

	if slot == 0 {
		log.Println("warn slot zero")
		return result, nil
	}

	for f, c := range counts {
		flag, err := monitor.GetFlag(siteID, f)
		if err != nil {
			return nil, err
		}

		result.TransCount += c

		if flag != nil {
			if !monitor.CheckFlag(monitor.FLAG_TRANSMISSION, flag.Bits) {
				result.SlotCount -= c
			}
			if monitor.CheckFlag(monitor.FLAG_EFFECTIVE, flag.Bits) {
				result.EffectCount += c
			}
		}
	}

	if result.SlotCount < 0 {
		log.Println("warn slot count < 0: ", slot, counts)
		result.SlotCount = 0
	}

	if result.SlotCount > 0 {
		switch sm.TransEffectRateVersion {
		case 0:
			result.TransRate = float64(result.TransCount) / float64(result.SlotCount)
			if result.TransRate > 0 {
				result.EffectRate = float64(result.EffectCount) / float64(result.TransCount)
			}
			result.EffectTransRate = float64(result.EffectCount) / float64(result.SlotCount)
		case 1:
			result.TransRate = float64(result.TransCount) / float64(result.SlotCount)
			result.EffectRate = float64(result.EffectCount) / float64(result.SlotCount)
			result.EffectTransRate = result.TransRate * result.EffectRate
		default:
			result.TransRate = float64(result.TransCount) / float64(result.SlotCount)
			if result.TransRate > 0 {
				result.EffectRate = float64(result.EffectCount) / float64(result.TransCount)
			}
			result.EffectTransRate = float64(result.EffectCount) / float64(result.SlotCount)
		}
	}

	// 	result.TransCount = counts[1]
	// 	result.EffectCount = counts[0]

	// 	if result.TransCount > result.SlotCount {
	// 		result.TransCount = slot
	// 	}
	// 	if result.EffectCount > result.TransCount {
	// 		result.EffectCount = result.TransCount
	// 	}

	// 	switch sm.TransEffectRateVersion {
	// 	case 0:
	// 		result.TransRate = float64(result.TransCount) / float64(slot)
	// 		if result.TransRate > 0 {
	// 			result.EffectRate = float64(result.EffectCount) / float64(result.TransCount)
	// 		}
	// 		result.EffectTransRate = float64(result.EffectCount) / float64(slot)
	// 	case 1:
	// 		result.TransRate = float64(result.TransCount) / float64(slot)
	// 		result.EffectRate = float64(result.EffectCount) / float64(slot)
	// 		result.EffectTransRate = result.TransRate * result.EffectRate
	// 	}
	// }

	if result.TransRate > 1 {
		log.Println("warn transrate > 1: ", slot, counts)
		result.TransRate = 1
	}
	if result.EffectRate > 1 {
		log.Println("warn effectrate > 1: ", slot, counts)
		result.EffectRate = 1
	}
	if result.EffectTransRate > 1 {
		log.Println("warn effecttransrate > 1: ", slot, counts)
		result.EffectTransRate = 1
	}

	return result, nil
}

func GetStationDataQuality(siteID string, actionAuth authority.ActionAuthSet, beginDate, endDate *time.Time, stationID ...int) (map[int]*QualityRates, error) {

	if beginDate == nil || endDate == nil {
		return nil, e_need_datatime
	}
	types := []int{monitor.MONITOR_DATA}

	result := make(map[int]*QualityRates)
	stationMonitorCount, err := monitor.CountStationMonitor(siteID, actionAuth, types, stationID...)
	if err != nil {
		return nil, err
	}

	stationID = make([]int, 0)
	for sid := range stationMonitorCount {
		stationID = append(stationID, sid)
	}

	stationList, err := entity.GetStation(siteID, stationID...)
	if err != nil {
		return nil, err
	}

	stations := make(map[int]*entity.Station)
	for _, s := range stationList {
		stations[s.ID] = s
	}

	stationStats, err := getStationStatistics(siteID, beginDate, endDate, stationID...)
	if err != nil {
		return nil, err
	}

	for sid, monitorCount := range stationMonitorCount {

		stationBeginDate := time.Time(*beginDate)

		station := stations[sid]
		if station == nil {
			continue
		}

		if station.Status == entity.INACTIVE {
			continue
		}

		if time.Time(station.OnlineTime).After(stationBeginDate) {
			if time.Time(station.OnlineTime).After(*endDate) {
				continue
			}

			stationBeginDate = time.Time(station.OnlineTime)
		}

		slotsPerMonitor := countHourlySlot(&stationBeginDate, endDate)

		result[sid], err = calculateRates(siteID, slotsPerMonitor*monitorCount, stationStats[sid])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func GetMonitorDataQuality(siteID string, actionAuth authority.ActionAuthSet, stationID int, beginDate, endDate *time.Time) (map[int]*QualityRates, error) {

	if beginDate == nil || endDate == nil {
		return nil, e_need_datatime
	}

	stationList, err := entity.GetStation(siteID, stationID)
	if err != nil {
		return nil, err
	}

	if len(stationList) == 0 {
		return nil, errors.New("站点不存在")
	}

	result := make(map[int]*QualityRates)

	station := stationList[0]

	if time.Time(station.OnlineTime).After(*beginDate) {
		if time.Time(station.OnlineTime).After(*endDate) {
			return result, nil
		}

		beginDate = (*time.Time)(&station.OnlineTime)
	}

	slotsPerMonitor := countHourlySlot(beginDate, endDate)

	stationMonitors, err := monitor.GetStationMonitors(siteID, actionAuth, stationID)
	if err != nil {
		return nil, err
	}

	if _, exists := stationMonitors[stationID]; !exists {
		return result, nil
	}

	monitorStats, err := getMonitorStatistics(siteID, beginDate, endDate, stationID, stationMonitors[stationID]...)
	if err != nil {
		return nil, err
	}

	for monitorID, stats := range monitorStats {
		result[monitorID], err = calculateRates(siteID, slotsPerMonitor, stats)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
