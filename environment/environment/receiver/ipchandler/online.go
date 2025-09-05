package ipchandler

import (
	"log"
	"sync"
	"time"

	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/ipcmessage"
)

var onlineCountdown = make(map[string]chan byte)
var onlineLock sync.RWMutex

func Online(mn, proto string) {

	log.Println("online:", mn, proto)

	offlineLock.RLock()
	offlineTimer, exists := offlineCountdown[mn]
	if exists {
		offlineLock.RUnlock()
		offlineLock.Lock()
		offlineTimer, exists = offlineCountdown[mn]
		if exists {
			if !offlineTimer.Stop() {
				select {
				case <-offlineTimer.C:
				default:
				}
			}
			delete(offlineCountdown, mn)
		}
		offlineLock.Unlock()
	} else {
		offlineLock.RUnlock()
	}

	log.Println("online offline timer removed:", mn, proto)

	sm, err := environment.GetModule(Config.SiteID)
	if err != nil {
		log.Println("error get environment module: ", err)
		return
	}

	for _, p := range sm.Protocols {
		if p.Protocol == proto {
			if p.OfflineDelayMin <= 0 {
				onlineLock.RLock()
				_, exists := onlineCountdown[mn]
				onlineLock.RUnlock()
				if !exists {
					onlineLock.Lock()
					onlineCountdown[mn] = make(chan byte)
					onlineLock.Unlock()
				}

				log.Println("online without countdown:", mn, proto)

				return
			}
			onlineLock.RLock()

			reset, exists := onlineCountdown[mn]
			if exists {
				select {
				case reset <- 1:
				default:
					log.Println("fail to reset online count down:", mn, proto)
				}
				onlineLock.RUnlock()
				log.Println("online with countdown reseted:", mn, proto)
				return
			}

			onlineLock.RUnlock()
			onlineLock.Lock()
			defer onlineLock.Unlock()

			reset, exists = onlineCountdown[mn]
			if exists {
				select {
				case reset <- 1:
				default:
					log.Println("fail to reset online count down:", mn, proto)
				}
				log.Println("online with countdown reseted 2:", mn, proto)
				return
			}

			reset = make(chan byte)
			onlineCountdown[mn] = reset

			delay := time.Minute * p.OfflineDelayMin

			go func() {
				t := time.NewTimer(delay)
				for {
					select {
					case <-reset:
						if !t.Stop() {
							select {
							case <-t.C:
							default:
							}
						}
						t.Reset(delay)
					case <-t.C:
						log.Println("offline countdown triggered: ", mn, proto)
						onlineLock.Lock()
						defer onlineLock.Unlock()
						delete(onlineCountdown, mn)

						log.Println("offline report")
						ReportStation(mn, false)
						offlineCountdownStart(mn, proto)

						return
					}
				}
			}()

			log.Println("online with countdown:", mn, proto)

			return
		}
	}

	log.Println("error online proto not found: ", mn, proto)
}

func Offline(mn, proto string) {
	log.Println("offline:", mn, proto)

	sm, err := environment.GetModule(Config.SiteID)
	if err != nil {
		log.Println("error get environment module: ", err)
		return
	}

	for _, p := range sm.Protocols {
		if p.Protocol == proto {
			if p.OfflineDelayMin <= 0 {
				onlineLock.Lock()
				defer onlineLock.Unlock()
				delete(onlineCountdown, mn)
				ReportStation(mn, false)
				offlineCountdownStart(mn, proto)
			}
		}
	}

	return
}

var offlineCountdown = make(map[string]*time.Timer)
var offlineLock sync.RWMutex

func offlineCountdownStart(mn, proto string) {
	log.Println("offline countdown:", mn, proto)

	sm, err := environment.GetModule(Config.SiteID)
	if err != nil {
		log.Println("error get environment module: ", err)
		return
	}

	for _, p := range sm.Protocols {
		if p.Protocol == proto {
			if p.OfflineCountdownMin <= 0 {

				log.Println("offline withount countdown triggered:", mn, proto)

				station := entity.GetCacheStationByMN(Config.SiteID, mn)
				if station != nil {
					toSend := ipcmessage.StationOffline(station.ID)
					broadcast(&toSend)
				}
				return
			}

			offlineLock.RLock()

			_, exists := offlineCountdown[mn]
			if exists {
				offlineLock.RUnlock()
				return
			}

			offlineLock.RUnlock()
			offlineLock.Lock()
			defer offlineLock.Unlock()

			_, exists = offlineCountdown[mn]
			if exists {
				return
			}

			countdown := time.Minute * p.OfflineCountdownMin
			offlineCountdown[mn] = time.AfterFunc(countdown, func() {
				log.Println("offline with countdown triggered:", mn, proto)
				offlineLock.Lock()
				defer offlineLock.Unlock()
				delete(offlineCountdown, mn)
				station := entity.GetCacheStationByMN(Config.SiteID, mn)
				if station != nil {
					toSend := ipcmessage.StationOffline(station.ID)
					broadcast(&toSend)
				}
			})

			log.Println("offline with countdown:", mn, proto)

			return
		}
	}

	return
}
