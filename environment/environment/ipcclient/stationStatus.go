package ipcclient

import (
	"errors"
	"log"
	"sync"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/ipcmessage"
)

var statusHistory map[string]*stationStatusHistoryPool
var statusHistoryLock sync.RWMutex

type stationStatusHistoryPool struct {
	Lock    sync.RWMutex
	History []*StationStatusHistory
	Timer   *time.Timer
}

func (p *stationStatusHistoryPool) clean(siteID string) {

	log.Println("clean station status history pool")

	p.Lock.Lock()
	defer p.Lock.Unlock()

	mod, err := environment.GetModule(siteID)
	if err != nil {
		log.Println("error get environment module: ", err)
		return
	}

	if mod.StationStatusCacheTimeMin <= 0 {
		p.History = make([]*StationStatusHistory, 0)
		return
	}

	head := p.History[0]
	for !time.Time(head.Time).Add(mod.StationStatusCacheTimeMin).After(time.Now()) {
		p.History = append([]*StationStatusHistory{}, p.History[1:]...)
		if len(p.History) > 0 {
			head = p.History[0]
		} else {
			head = nil
			break
		}
	}

	if head != nil {
		p.Timer = time.AfterFunc(time.Time(head.Time).Add(mod.StationStatusCacheTimeMin*time.Minute).Sub(time.Now()), func() { p.clean(siteID) })
	} else {
		p.Timer = nil
	}
}

type StationStatusHistory struct {
	StationID int       `json:"stationID"`
	Online    bool      `json:"online"`
	Time      util.Time `json:"time"`
}

func stationHistoryInput(siteID string, stationID int, online bool) (pool *stationStatusHistoryPool) {

	log.Println("station history input: ", stationID, online)

	mod, err := environment.GetModule(siteID)
	if err != nil {
		log.Println("error get environment module: ", err)
		return
	}

	if mod.StationStatusCacheTimeMin <= 0 {
		log.Println("no station status cache setting")
		return
	}

	defer func() {
		if pool == nil {
			log.Println("init station status cache pool")
			statusHistoryLock.Lock()
			defer statusHistoryLock.Unlock()

			pool = new(stationStatusHistoryPool)
			pool.Lock.Lock()
			defer pool.Lock.Unlock()

			pool.History = make([]*StationStatusHistory, 0)
			pool.History = append(pool.History, &StationStatusHistory{
				StationID: stationID,
				Online:    online,
				Time:      util.Time(time.Now()),
			})
			pool.Timer = time.AfterFunc(mod.StationStatusCacheTimeMin*time.Minute, func() { pool.clean(siteID) })

			if statusHistory == nil {
				statusHistory = make(map[string]*stationStatusHistoryPool)
			}
			statusHistory[siteID] = pool
		}
	}()

	statusHistoryLock.RLock()
	defer statusHistoryLock.RUnlock()

	if statusHistory == nil {
		return
	}

	var exists bool
	pool, exists = statusHistory[siteID]

	if !exists {
		pool = nil
		return
	}

	pool.Lock.Lock()
	defer pool.Lock.Unlock()

	pool.History = append(pool.History, &StationStatusHistory{
		StationID: stationID,
		Online:    online,
		Time:      util.Time(time.Now()),
	})

	if pool.Timer == nil {
		pool.Timer = time.AfterFunc(mod.StationStatusCacheTimeMin*time.Minute, func() { pool.clean(siteID) })
	}

	return
}

func GetStationStatusHistory(siteID string, stationID ...int) []*StationStatusHistory {
	result := make([]*StationStatusHistory, 0)

	if len(stationID) == 0 {
		return result
	}

	ids := make(map[int]byte)
	for _, id := range stationID {
		ids[id] = 1
	}

	statusHistoryLock.RLock()
	defer statusHistoryLock.RUnlock()

	if statusHistory == nil {
		return result
	}

	pool := statusHistory[siteID]
	if pool == nil {
		return result
	}

	pool.Lock.RLock()
	defer pool.Lock.RUnlock()

	for _, h := range pool.History {
		if _, exists := ids[h.StationID]; exists {
			result = append([]*StationStatusHistory{h}, result...)
		}
	}

	return result
}

var E_receiver_request_failure = errors.New("服务器接收端未连接")

func RequestStationStatus(siteID string, stationID ...int) map[int]bool {
	result := make(map[int]bool)

	if len(stationID) == 0 {
		return result
	}

	var req ipcmessage.StationStatusReq
	req = stationID

	for _, hostAddr := range Config.EnvironmentReceiverAddrs {
		if hostAddr.SiteID == siteID {
			send := make(chan ipcmessage.IMessage)
			receive, err := StartRequestClient(siteID, hostAddr.ConnType, hostAddr.Addr, send)
			if err != nil {
				log.Println("error request station status start client: ", err)
				continue
			}

			send <- &req

			select {
			case res, ok := <-receive:
				if !ok {
					log.Println("error request station status receiving closed")
				} else {
					if status, ok := res.(*ipcmessage.StationStatusRes); ok {
						for stationID, online := range *status {
							if previous, exists := result[stationID]; !exists || !previous {
								result[stationID] = online
							}
						}
					} else {
						log.Println("error request station status unexpected res type: ", res.GetIPCMessageType(), res)
					}
				}
			case <-time.After(Config.EnvironmentReceiverRequestTimeOutSec * time.Second):
				log.Println("error request station status time out ")
			}

			close(send)
		}
	}

	return result
}
