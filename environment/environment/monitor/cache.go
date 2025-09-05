package monitor

import (
	"log"
	"strings"
	"sync"
)

var siteCache = make(map[string]*sitePool)
var siteCacheLock sync.RWMutex

type sitePool struct {
	monitorLock     sync.RWMutex
	monitorMap      map[int]*Monitor
	codeLock        sync.RWMutex
	codeMap         map[int]map[string]*MonitorCode
	monitorCodeLock sync.RWMutex
	monitorCodeMap  map[int]map[int]*MonitorCode
	codeIdLock      sync.RWMutex
	codeIdMap       map[int]*MonitorCode
	flagLimitLock   sync.RWMutex
	flagLimitMap    map[int]map[int]map[string]*FlagLimit
}

func getCacheSitePool(siteID string) *sitePool {
	siteCacheLock.RLock()
	siteP, exists := siteCache[siteID]
	siteCacheLock.RUnlock()

	if !exists {
		siteCacheLock.Lock()
		defer siteCacheLock.Unlock()

		siteP, exists = siteCache[siteID]

		if !exists {
			siteP = new(sitePool)
			siteCache[siteID] = siteP

			siteP.monitorMap = make(map[int]*Monitor)
			siteP.codeMap = make(map[int]map[string]*MonitorCode)
			siteP.monitorCodeMap = make(map[int]map[int]*MonitorCode)
			siteP.codeIdMap = make(map[int]*MonitorCode)
			siteP.flagLimitMap = make(map[int]map[int]map[string]*FlagLimit)
		}
	}
	return siteP
}

func LoadMonitor(siteID string) error {
	siteP := getCacheSitePool(siteID)
	return siteP.loadMonitor(siteID)
}

func LoadMonitorCode(siteID string) error {
	siteP := getCacheSitePool(siteID)
	return siteP.loadMonitorCode(siteID)
}

func LoadFlagLimit(siteID string) error {
	siteP := getCacheSitePool(siteID)
	return siteP.loadFlagLimit(siteID)
}

func (p *sitePool) loadMonitor(siteID string) error {
	log.Println("load monitor: ", siteID)
	monitorList, err := GetMonitors(siteID, nil, nil)
	if err != nil {
		return err
	}

	p.monitorLock.Lock()
	defer p.monitorLock.Unlock()

	p.monitorMap = make(map[int]*Monitor)

	for _, m := range monitorList {
		p.monitorMap[m.ID] = m
	}

	log.Println("load monitor done: ", len(p.monitorMap))

	return nil
}

func (p *sitePool) loadMonitorCode(siteID string) error {
	log.Println("load monitor code")

	monitorCodeList, err := GetMonitorCodes(siteID, -1, "")
	if err != nil {
		return err
	}

	p.codeIdLock.Lock()
	p.codeLock.Lock()
	p.monitorCodeLock.Lock()
	defer p.codeIdLock.Unlock()
	defer p.codeLock.Unlock()
	defer p.monitorCodeLock.Unlock()

	p.codeIdMap = make(map[int]*MonitorCode)
	p.codeMap = make(map[int]map[string]*MonitorCode)
	p.monitorCodeMap = make(map[int]map[int]*MonitorCode)

	for _, m := range monitorCodeList {
		entry := MonitorCode(*m)

		p.codeIdMap[m.ID] = m

		if _, exists := p.codeMap[m.StationID]; !exists {
			p.codeMap[m.StationID] = make(map[string]*MonitorCode)
		}
		p.codeMap[m.StationID][m.Code] = &entry

		if _, exists := p.monitorCodeMap[m.StationID]; !exists {
			p.monitorCodeMap[m.StationID] = make(map[int]*MonitorCode)
		}
		p.monitorCodeMap[m.StationID][m.MonitorID] = &entry
	}

	log.Println("load monitor code done")

	return nil
}

func (p *sitePool) loadFlagLimit(siteID string) error {

	log.Println("load monitor flag limit")

	limitList, err := GetFlagLimits(siteID, nil, nil, nil)
	if err != nil {
		return err
	}

	p.flagLimitLock.Lock()
	defer p.flagLimitLock.Unlock()

	p.flagLimitMap = make(map[int]map[int]map[string]*FlagLimit)

	for _, m := range limitList {
		stationMapping, exists := p.flagLimitMap[m.StationID]
		if !exists {
			stationMapping = make(map[int]map[string]*FlagLimit)
			p.flagLimitMap[m.StationID] = stationMapping
		}
		monitorMapping, exists := stationMapping[m.MonitorID]
		if !exists {
			monitorMapping = make(map[string]*FlagLimit)
			stationMapping[m.MonitorID] = monitorMapping
		}
		monitorMapping[m.Flag] = m
	}

	log.Println("load monitor limit done: ", len(p.flagLimitMap))

	return nil
}

func GetMonitorCodeByID(siteID string, codeID int) *MonitorCode {
	siteP := getCacheSitePool(siteID)

	siteP.codeIdLock.RLock()
	defer siteP.codeIdLock.RUnlock()

	return siteP.codeIdMap[codeID]
}

func GetMonitorCodeByStationMonitor(siteID string, stationID, monitorID int) *MonitorCode {
	siteP := getCacheSitePool(siteID)

	siteP.monitorCodeLock.RLock()
	defer siteP.monitorCodeLock.RUnlock()

	if stationCodes, exits := siteP.monitorCodeMap[stationID]; exits {
		if monitorCodes, exists := stationCodes[monitorID]; exists {
			return monitorCodes
		}
	}

	if stationID > 0 {
		return GetMonitorCodeByStationMonitor(siteID, 0, monitorID)
	}

	return nil
}

func GetMonitorCodeByCode(siteID string, stationID int, code string) *MonitorCode {
	siteP := getCacheSitePool(siteID)

	siteP.codeLock.RLock()
	defer siteP.codeLock.RUnlock()

	if !strings.HasPrefix(code, CODE_DEFAULT) {
		if codes, exists := siteP.codeMap[stationID]; exists {
			if entry, exists := codes[code]; exists {
				return entry
			}
		}
	}

	if codes, exists := siteP.codeMap[0]; exists {
		if entry, exists := codes[code]; exists {
			return entry
		}
	}

	return nil
}

func GetAllMonitor(siteID string) []*Monitor {
	siteP := getCacheSitePool(siteID)

	siteP.monitorLock.RLock()
	defer siteP.monitorLock.RUnlock()

	result := make([]*Monitor, 0)
	for _, m := range siteP.monitorMap {
		result = append(result, m)
	}

	return result
}

func GetMonitor(siteID string, monitorID int) *Monitor {

	siteP := getCacheSitePool(siteID)

	siteP.monitorLock.RLock()
	defer siteP.monitorLock.RUnlock()

	return siteP.monitorMap[monitorID]
}

func GetFlagLimit(siteID string, stationID, monitorID int, flag string) (result *FlagLimit) {

	defer func() {
		if result == nil && stationID > 0 {
			result = GetFlagLimit(siteID, 0, monitorID, flag)
		}
	}()

	siteP := getCacheSitePool(siteID)

	siteP.flagLimitLock.RLock()
	defer siteP.flagLimitLock.RUnlock()

	stationMapping, exists := siteP.flagLimitMap[stationID]
	if exists {
		monitorMapping, exist := stationMapping[monitorID]
		if exist {
			if entry, exists := monitorMapping[flag]; exists {
				result = entry
				return
			}
		}
	}

	return nil
}
