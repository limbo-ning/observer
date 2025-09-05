package entity

import (
	"log"
	"sync"
)

var cache = make(map[string]*sitePool)
var cacheLock sync.RWMutex

type sitePool struct {
	stations      map[int]*Station
	stationLock   sync.RWMutex
	mnStations    map[string]*Station
	mnStationLock sync.RWMutex
}

func getSitePool(siteID string, createIfNotExists bool) (result *sitePool) {

	defer func() {
		if result == nil {
			if createIfNotExists {
				cacheLock.Lock()
				defer cacheLock.Unlock()

				result = cache[siteID]
				if result == nil {
					result = new(sitePool)
					cache[siteID] = result

					result.stations = make(map[int]*Station)
					result.mnStations = make(map[string]*Station)
				}
			}
		}
	}()

	cacheLock.RLock()
	defer cacheLock.RUnlock()

	result = cache[siteID]
	return
}

func setStationCache(siteID string, clearAll bool, station ...*Station) {

	sp := getSitePool(siteID, true)

	sp.stationLock.Lock()
	sp.mnStationLock.Lock()
	defer sp.stationLock.Unlock()
	defer sp.mnStationLock.Unlock()

	if clearAll {
		sp.stations = make(map[int]*Station)
		sp.mnStations = make(map[string]*Station)
	}

	for _, s := range station {
		sp.stations[s.ID] = s
		if s.Protocol != "" && s.MN != "" {
			sp.mnStations[s.MN] = s
		}
	}

	log.Println("set station cache: ", clearAll, len(station))
}

func getStationCache(siteID string, stationID ...int) []*Station {

	sp := getSitePool(siteID, false)

	if sp == nil {
		return nil
	}

	sp.stationLock.RLock()
	defer sp.stationLock.RUnlock()

	result := make([]*Station, 0)
	if len(stationID) == 0 {
		for _, s := range sp.stations {
			result = append(result, s)
		}
	} else {
		for _, id := range stationID {
			if s, exists := sp.stations[id]; exists {
				result = append(result, s)
			}
		}
	}

	return result
}

func GetCacheStationByID(siteID string, id int) *Station {
	sp := getSitePool(siteID, true)

	if sp == nil {
		return nil
	}

	sp.stationLock.RLock()
	defer sp.stationLock.RUnlock()

	if station, exists := sp.stations[id]; exists {
		return station
	}

	return nil
}

func GetCacheStationByMN(siteID, mn string) *Station {
	sp := getSitePool(siteID, true)

	if sp == nil {
		return nil
	}

	sp.mnStationLock.RLock()
	defer sp.mnStationLock.RUnlock()

	if station, exists := sp.mnStations[mn]; exists {
		return station
	}

	return nil
}

func LoadStation(siteID string, stationID ...int) error {

	log.Println("load station: ", stationID)

	stationList, err := GetStations(siteID, nil, nil, "", "", "", stationID...)
	if err != nil {
		return err
	}

	//如果是删除了 需要全部重新加载
	if len(stationID) != 0 && len(stationList) == 0 {
		return LoadStation(siteID)
	}

	clearAll := len(stationID) == 0
	setStationCache(siteID, clearAll, stationList...)

	log.Println("load station done")

	return nil
}
