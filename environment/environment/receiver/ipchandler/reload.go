package ipchandler

import (
	"log"

	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/ipcmessage"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/receiver/connection"
)

func ReloadModule() *ipcmessage.Ack {

	var result ipcmessage.Ack

	result = 0

	if _, err := environment.GetModule(Config.SiteID, false, true); err != nil {
		result = 1
	}

	if _, err := monitor.GetModule(Config.SiteID, false, true); err != nil {
		result = 1
	}

	log.Println("load module: ", result)

	return &result
}

func ReloadStation(stationID int) *ipcmessage.Ack {

	station := entity.GetCacheStationByID(Config.SiteID, stationID)

	var result ipcmessage.Ack

	if err := entity.LoadStation(Config.SiteID, stationID); err != nil {
		result = 1
	} else {
		result = 0
	}

	if station != nil {
		running, exists := connection.GetRunningProtocol(station.MN)
		if exists {
			log.Println("reload station running previously: ", stationID)
			connection.RemoveConnection(station.MN, station.Protocol, running)
		} else {
			log.Println("reload station not running previously: ", stationID)
		}
	} else {
		log.Println("reload station not exists previously: ", stationID)
	}

	return &result
}

func ReloadMonitor() *ipcmessage.Ack {
	var result ipcmessage.Ack

	if err := monitor.LoadMonitor(Config.SiteID); err != nil {
		result = 1
	} else {
		result = 0
	}

	return &result
}

func ReloadMonitorCode() *ipcmessage.Ack {

	var result ipcmessage.Ack

	if err := monitor.LoadMonitorCode(Config.SiteID); err != nil {
		result = 1
	} else {
		result = 0
	}

	return &result
}

func ReloadFlagLimit() *ipcmessage.Ack {

	var result ipcmessage.Ack

	if err := monitor.LoadFlagLimit(Config.SiteID); err != nil {
		result = 1
	} else {
		result = 0
	}

	return &result
}
