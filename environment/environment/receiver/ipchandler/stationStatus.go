package ipchandler

import (
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/ipcmessage"
)

func ReportRequestedStation(stationIDs []int) *ipcmessage.StationStatusRes {
	var result ipcmessage.StationStatusRes
	result = make(map[int]bool)

	if len(stationIDs) == 0 {
		return nil
	}

	onlineLock.RLock()
	defer onlineLock.RUnlock()

	for _, id := range stationIDs {
		s := entity.GetCacheStationByID(Config.SiteID, id)
		if s != nil {
			if _, exists := onlineCountdown[s.MN]; exists {
				result[id] = true
				continue
			}
		}
		result[id] = false
	}

	return &result
}

func ReportStation(mn string, online bool) {
	station := entity.GetCacheStationByMN(Config.SiteID, mn)
	if station != nil {
		var toSend ipcmessage.StationStatusRes
		toSend = map[int]bool{station.ID: online}
		broadcast(&toSend)
	}
}
