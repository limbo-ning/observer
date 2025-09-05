package ipcclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/subscription"
	"obsessiontech/environment/push"
	"obsessiontech/environment/role"
	"obsessiontech/environment/websocket"
)

var stationBroadcastInstance *stationBroadcast

const (
	statusBroadcast  = "status"
	offlineBroadcast = "offline"
)

func init() {
	stationBroadcastInstance = new(stationBroadcast)
	stationBroadcastInstance.pool = make(map[string]*stationBroadcastSitePool)
	websocket.RegisterHandler("stationBroadcast", stationBroadcastInstance)
}

type stationBroadcastSitePool struct {
	Lock sync.RWMutex
	Pool map[int64]*stationBroadcastFeed
}
type stationBroadcastFeed struct {
	StationID  []int
	Type       []string
	WriteCH    chan interface{}
	ActionAuth authority.ActionAuthSet
}

type stationBroadcast struct {
	lock sync.RWMutex
	pool map[string]*stationBroadcastSitePool
}

type stationBroadcastAction struct {
	websocket.Action
	Operation string `json:"operation"`
	StationID string `json:"stationID"`
	Type      string `json:"type"`
}

func (b *stationBroadcast) subscribe(siteID, session string, uid int, action *stationBroadcastAction, id int64, writeCh chan interface{}) error {
	if action.StationID == "" {
		return errors.New("需要指定监测点")
	}

	if uid <= 0 {
		return websocket.E_need_login
	}

	actionAuth, err := role.GetAuthorityActions(siteID, entity.MODULE_ENTITY, session, "", uid, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return err
	}
	if len(actionAuth) == 0 {
		return websocket.E_need_login
	}

	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()

	feed := new(stationBroadcastFeed)
	feed.Type = strings.Split(action.Type, ",")

	for i := range feed.Type {
		t := feed.Type[i]
		switch t {
		case statusBroadcast:
		case offlineBroadcast:
		default:
			feed.Type[i] = statusBroadcast
		}
	}

	ids := strings.Split(action.StationID, ",")
	feed.StationID = make([]int, 0)
	for _, idstr := range ids {
		id, err := strconv.Atoi(idstr)
		if err != nil {
			return err
		}
		feed.StationID = append(feed.StationID, id)
	}
	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, feed.StationID, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		return err
	}

	feed.StationID = make([]int, 0)
	for sid, ok := range filtered {
		if ok {
			feed.StationID = append(feed.StationID, sid)
		}
	}
	if len(feed.StationID) == 0 {
		return errors.New("无权限监听监测点")
	}

	feed.WriteCH = writeCh
	feed.ActionAuth = actionAuth

	siteP.Pool[id] = feed

	return nil
}

func (b *stationBroadcast) getSitePool(siteID string) *stationBroadcastSitePool {
	b.lock.RLock()
	siteP, exists := b.pool[siteID]
	b.lock.RUnlock()

	if !exists {
		b.lock.Lock()
		defer b.lock.Unlock()

		siteP, exists = b.pool[siteID]

		if !exists {
			siteP = new(stationBroadcastSitePool)
			siteP.Pool = make(map[int64]*stationBroadcastFeed)

			b.pool[siteID] = siteP
		}
	}
	return siteP
}

func (b *stationBroadcast) OnMessage(siteID, session string, uid int, id int64, msg []byte, writeCh chan interface{}) error {
	var action stationBroadcastAction
	if err := json.Unmarshal(msg, &action); err != nil {
		return err
	}
	switch action.Operation {
	case "subscribe":
		if err := b.subscribe(siteID, session, uid, &action, id, writeCh); err != nil {
			return err
		}
	case "unsubscribe":
		siteP := b.getSitePool(siteID)
		siteP.Lock.Lock()
		defer siteP.Lock.Unlock()
		delete(siteP.Pool, id)
	default:
		return errors.New("operation not support")
	}

	return nil
}

func (b *stationBroadcast) OnClose(siteID string, id int64) {
	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()
	delete(siteP.Pool, id)
}

func BroadcastStation(siteID string, stationID int, online bool) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("fatal error broadcast panic: ", err)
		}
	}()

	toSend := make(map[int64]chan interface{})

	stationBroadcastInstance.lock.RLock()
	defer stationBroadcastInstance.lock.RUnlock()
	siteP, exists := stationBroadcastInstance.pool[siteID]

	if !exists {
		log.Println("no station subscription pool")
		return
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for id, feed := range siteP.Pool {

		checked := false
		for _, t := range feed.Type {
			if t == statusBroadcast {
				checked = true
				break
			}
		}
		if !checked {
			continue
		}

		checked = false
		for _, s := range feed.StationID {
			if stationID == s {
				checked = true
				break
			}
		}
		if !checked {
			continue
		}

		toSend[id] = feed.WriteCH
	}

	if len(toSend) == 0 {
		return
	}

	data := make(map[string]interface{})
	data["type"] = "stationBroadcast"
	data["online"] = online
	data["stationID"] = stationID

	for _, writeCh := range toSend {
		writeCh <- data
	}
}

func BroadcastStationOffline(siteID string, stationID int) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("fatal error broadcast panic: ", err)
		}
	}()

	toSend := make(map[int64]chan interface{})

	stationBroadcastInstance.lock.RLock()
	defer stationBroadcastInstance.lock.RUnlock()
	siteP, exists := stationBroadcastInstance.pool[siteID]

	if !exists {
		log.Println("no station subscription pool")
		return
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for id, feed := range siteP.Pool {
		checked := false
		for _, t := range feed.Type {
			if t == offlineBroadcast {
				checked = true
				break
			}
		}
		if !checked {
			continue
		}

		checked = false
		for _, s := range feed.StationID {
			if stationID == s {
				checked = true
				break
			}
		}
		if !checked {
			continue
		}

		toSend[id] = feed.WriteCH
	}

	if len(toSend) == 0 {
		return
	}

	stations, err := entity.GetStation(siteID, stationID)
	if err != nil {
		log.Println("error get station: ", err)
		return
	}

	if len(stations) == 0 {
		log.Println("error get station: station not found: ", stationID)
		return
	}
	station := stations[0]

	duration := "预警时间间隔"

	environmentModule, err := environment.GetModule(siteID)
	if err != nil {
		log.Println("error get environment module: ", err)
	} else {
		for _, p := range environmentModule.Protocols {
			if station.Protocol == p.Protocol {
				duration = fmt.Sprintf("%.1f小时", (time.Minute*p.OfflineCountdownMin).Hours()+(time.Minute*p.OfflineDelayMin).Hours())
				break
			}
		}
	}

	data := make(map[string]interface{})
	data["type"] = "stationOfflineBroadcast"
	data["offline"] = true
	data["stationID"] = stationID
	data["duration"] = duration

	for _, writeCh := range toSend {
		writeCh <- data
	}
}

var stationOfflinePushed map[string]*siteStationPushed
var stationOfflinePushedLock sync.RWMutex

type siteStationPushed struct {
	lock     sync.RWMutex
	silenced map[int]*time.Timer
	tocease  map[int]byte
}

func getOfflinePush(siteID string, stationID int) (stationOffliner *siteStationPushed) {

	var exists bool
	defer func() {
		if stationOffliner == nil {
			stationOfflinePushedLock.Lock()
			defer stationOfflinePushedLock.Unlock()

			stationOffliner, exists = stationOfflinePushed[siteID]
			if !exists {
				stationOffliner = &siteStationPushed{silenced: make(map[int]*time.Timer), tocease: make(map[int]byte)}
				if stationOfflinePushed == nil {
					stationOfflinePushed = make(map[string]*siteStationPushed)
				}
				stationOfflinePushed[siteID] = stationOffliner
			}
		}
	}()

	stationOfflinePushedLock.RLock()
	defer stationOfflinePushedLock.RUnlock()

	stationOffliner, exists = stationOfflinePushed[siteID]
	return
}

func PushStationStatus(siteID string, stationID int, online bool) error {

	sm, err := subscription.GetModule(siteID)
	if err != nil {
		log.Println("error push offline get subscription module: ", err)
		return nil
	}

	pushSetting := sm.PushSettings[subscription.STATION_STATUS]
	if pushSetting == nil {
		log.Println("station status push setting not set: ", siteID)
		return nil
	}

	if online {
		if pushSetting.Cease != nil {
			stationOffliner := getOfflinePush(siteID, stationID)
			stationOffliner.lock.Lock()
			defer stationOffliner.lock.Unlock()

			if _, exists := stationOffliner.tocease[stationID]; !exists {
				return nil
			}

			stations, err := entity.GetStation(siteID, stationID)
			if err != nil {
				log.Println("error set  ceasestation: ", err)
				return err
			}

			if len(stations) == 0 {
				log.Println("error set  ceasestation: station not found: ", stationID)
				return nil
			}
			station := stations[0]

			if station.Status != entity.ACTIVE {
				log.Println("station not active")
				return nil
			}

			entities, err := entity.GetEntities(siteID, station.EntityID)
			if err != nil {
				log.Println("error set cease station: entity not found: ", station.EntityID)
				return err
			}
			if len(entities) == 0 {
				log.Println("error set cease station: entity not found: ", station.EntityID)
				return nil
			}

			entity := entities[0]

			subscriptionList, err := subscription.GetSubscriptionsToPush(siteID, station.EntityID, station.ID, subscription.STATION_STATUS)
			if err != nil {
				log.Println("error get subscription list to push cease: ", err)
				return err
			}

			if len(subscriptionList) == 0 {
				return nil
			}

			for _, sub := range subscriptionList {
				stationSub := new(subscription.StationSubscription)
				stationSub.Subscription = *sub
				stationSub.Entity = entity
				stationSub.Station = station
				stationSub.Time = time.Now()
				stationSub.IsCease = true

				if err := push.Push(siteID, stationSub); err != nil {
					log.Println("error push station offline cease: ", err)
				}
			}

			delete(stationOffliner.tocease, stationID)

			return nil
		}
	}

	return nil
}

func PushOffline(siteID string, stationID int) error {

	sm, err := subscription.GetModule(siteID)
	if err != nil {
		log.Println("error push offline get subscription module: ", siteID, err)
		return err
	}

	pushSetting := sm.PushSettings[subscription.STATION_STATUS]
	if pushSetting == nil || pushSetting.Trigger == nil {
		log.Println("station status push setting not set: ", siteID)
		return nil
	}

	stationOffliner := getOfflinePush(siteID, stationID)
	stationOffliner.lock.Lock()
	defer stationOffliner.lock.Unlock()

	if stationOffliner.silenced[stationID] != nil {
		log.Println("push station offline already pushed")
		return nil
	}

	stations, err := entity.GetStation(siteID, stationID)
	if err != nil {
		log.Println("error set station: ", err)
		return err
	}

	if len(stations) == 0 {
		log.Println("error set station: station not found: ", stationID)
		return nil
	}
	station := stations[0]

	if station.Status != entity.ACTIVE {
		log.Println("station not active")
		return nil
	}

	entities, err := entity.GetEntities(siteID, station.EntityID)
	if err != nil {
		log.Println("error set station: entity not found: ", station.EntityID)
		return err
	}
	if len(entities) == 0 {
		log.Println("error set station: entity not found: ", station.EntityID)
		return nil
	}

	offlineTime := time.Now()

	environmentModule, err := environment.GetModule(siteID)
	if err != nil {
		log.Println("error get environment module: ", err)
	} else {
		for _, p := range environmentModule.Protocols {
			if station.Protocol == p.Protocol {
				offlineTime = offlineTime.Add(-1 * time.Minute * p.OfflineCountdownMin).Add(-1 * time.Minute * p.OfflineDelayMin)
				break
			}
		}
	}

	entity := entities[0]

	subscriptionList, err := subscription.GetSubscriptionsToPush(siteID, station.EntityID, station.ID, subscription.STATION_STATUS)
	if err != nil {
		log.Println("error get subscription list to push: ", err)
		return err
	}

	if len(subscriptionList) == 0 {
		return nil
	}

	for _, sub := range subscriptionList {
		stationSub := new(subscription.StationSubscription)
		stationSub.Subscription = *sub
		stationSub.Entity = entity
		stationSub.Station = station
		stationSub.Time = offlineTime

		if err := push.Push(siteID, stationSub); err != nil {
			log.Println("error push station offline: ", err)
		}
	}

	if pushSetting.Trigger.CooldownMin > 0 {
		stationOffliner.silenced[stationID] = time.AfterFunc(pushSetting.Trigger.CooldownMin*time.Minute, func() {
			stationOffliner := getOfflinePush(siteID, stationID)
			stationOffliner.lock.Lock()
			defer stationOffliner.lock.Unlock()

			delete(stationOffliner.silenced, stationID)
		})
	}

	if pushSetting.Cease != nil {
		stationOffliner.tocease[stationID] = 1
	}

	return nil
}

// func PushOffline(siteID string, stationID int) (sitePool *siteStationPushed, pushed bool) {

// 	log.Println("push offline: ", siteID, stationID)

// 	sm, err := subscription.GetModule(siteID)
// 	if err != nil {
// 		log.Println("error push offline get subscription module: ", err)
// 		return
// 	}

// 	pushSetting := sm.PushSettings[subscription.STATION_STATUS]
// 	if pushSetting == nil {
// 		log.Println("station status push setting not set: ", siteID)
// 		return
// 	}

// 	var exists bool

// 	defer func() {
// 		if pushed {

// 			if pushSetting.Trigger.CooldownMin <= 0 {
// 				return
// 			}

// 			if sitePool == nil {
// 				stationOfflinePushedLock.Lock()
// 				defer stationOfflinePushedLock.Unlock()

// 				sitePool, exists = stationOfflinePushed[siteID]
// 				if !exists {
// 					sitePool = &siteStationPushed{silenced: make(map[int]*time.Timer), tocease: make(map[int]byte)}
// 					if stationOfflinePushed == nil {
// 						stationOfflinePushed = make(map[string]*siteStationPushed)
// 					}
// 					stationOfflinePushed[siteID] = sitePool
// 				}

// 				sitePool.lock.Lock()
// 				defer sitePool.lock.Unlock()

// 				var t *time.Timer
// 				t = time.AfterFunc(pushSetting.Trigger.CooldownMin*time.Minute, func() {

// 					log.Println("push offline timer")

// 					sitePool.lock.Lock()
// 					defer sitePool.lock.Unlock()

// 					if sitePool.silenced[stationID] != t {
// 						log.Println("push offline timer different")
// 						return
// 					}

// 					delete(sitePool.silenced, stationID)

// 					if len(sitePool.silenced) == 0 {
// 						stationOfflinePushedLock.Lock()
// 						defer stationOfflinePushedLock.Unlock()

// 						delete(stationOfflinePushed, siteID)
// 					}
// 				})

// 				sitePool.silenced[stationID] = t
// 			}
// 		}
// 	}()

// 	stationOfflinePushedLock.RLock()
// 	defer stationOfflinePushedLock.RUnlock()

// 	sitePool, exists = stationOfflinePushed[siteID]
// 	if exists {
// 		sitePool.lock.RLock()
// 		defer sitePool.lock.RUnlock()

// 		if _, exists = sitePool.silenced[stationID]; exists {
// 			log.Println("push offline not timeout: ", stationID)
// 			return
// 		}
// 	}

// 	stations, err := entity.GetStation(siteID, stationID)
// 	if err != nil {
// 		log.Println("error set station: ", err)
// 		return
// 	}

// 	if len(stations) == 0 {
// 		log.Println("error set station: station not found: ", stationID)
// 		return
// 	}
// 	station := stations[0]

// 	entities, err := entity.GetEntities(siteID, station.EntityID)
// 	if err != nil {
// 		log.Println("error set station: entity not found: ", station.EntityID)
// 		return
// 	}
// 	if len(entities) == 0 {
// 		log.Println("error set station: entity not found: ", station.EntityID)
// 		return
// 	}

// 	offlineTime := time.Now()

// 	environmentModule, err := environment.GetModule(siteID)
// 	if err != nil {
// 		log.Println("error get environment module: ", err)
// 	} else {
// 		for _, p := range environmentModule.Protocols {
// 			if station.Protocol == p.Protocol {
// 				offlineTime = offlineTime.Add(-1 * time.Minute * p.OfflineCountdownMin).Add(-1 * time.Minute * p.OfflineDelayMin)
// 				break
// 			}
// 		}
// 	}

// 	entity := entities[0]

// 	subscriptionList, err := subscription.GetSubscriptionsToPush(siteID, station.EntityID, station.ID, subscription.STATION_STATUS)
// 	if err != nil {
// 		log.Println("error get subscription list to push: ", err)
// 		return
// 	}

// 	for _, sub := range subscriptionList {
// 		stationSub := new(subscription.StationSubscription)
// 		stationSub.Subscription = *sub
// 		stationSub.Entity = entity
// 		stationSub.Station = station
// 		stationSub.Time = offlineTime

// 		if err := push.Push(siteID, stationSub); err != nil {
// 			log.Println("error push station offline: ", err)
// 		}
// 	}

// 	pushed = true

// 	return
// }
