package ipcclient

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/subscription"
	"obsessiontech/environment/push"
	"obsessiontech/environment/role"
	"obsessiontech/environment/websocket"
)

var dataBroadcastInstance *dataBroadcast

func init() {
	dataBroadcastInstance = new(dataBroadcast)
	dataBroadcastInstance.pool = make(map[string]*dataBroadcastSitePool)
	websocket.RegisterHandler("dataBroadcast", dataBroadcastInstance)
}

type dataBroadcastSitePool struct {
	Lock sync.RWMutex
	Pool map[int64]*dataBroadcastFeed
}
type dataBroadcastFeed struct {
	DataType   []string
	Flag       []string
	StationID  []int
	MonitorID  []int
	WriteCH    chan interface{}
	ActionAuth authority.ActionAuthSet
}

type dataBroadcast struct {
	lock sync.RWMutex
	pool map[string]*dataBroadcastSitePool
}

type dataBroadcastAction struct {
	websocket.Action
	Operation string `json:"operation"`
	DataType  string `json:"dataType"`
	Flag      string `json:"flag"`
	StationID string `json:"stationID"`
	MonitorID string `json:"monitorID"`
}

func (b *dataBroadcast) subscribe(siteID, session string, uid int, action *dataBroadcastAction, id int64, writeCh chan interface{}) error {
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

	feed := new(dataBroadcastFeed)

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
	if action.DataType != "" {
		feed.DataType = strings.Split(action.DataType, ",")
	}
	if action.Flag != "" {
		feed.Flag = strings.Split(action.Flag, ",")
	}

	if action.MonitorID != "" {
		feed.MonitorID = make([]int, 0)
		ids := strings.Split(action.MonitorID, ",")
		for _, idstr := range ids {
			id, err := strconv.Atoi(idstr)
			if err != nil {
				return err
			}
			feed.MonitorID = append(feed.MonitorID, id)
		}
	}

	feed.WriteCH = writeCh
	feed.ActionAuth = actionAuth

	siteP.Pool[id] = feed

	return nil
}

func (b *dataBroadcast) getSitePool(siteID string) *dataBroadcastSitePool {
	b.lock.RLock()
	siteP, exists := b.pool[siteID]
	b.lock.RUnlock()

	if !exists {
		b.lock.Lock()
		defer b.lock.Unlock()

		siteP, exists = b.pool[siteID]

		if !exists {
			siteP = new(dataBroadcastSitePool)
			siteP.Pool = make(map[int64]*dataBroadcastFeed)

			b.pool[siteID] = siteP
		}
	}
	return siteP
}

func (b *dataBroadcast) OnMessage(siteID, session string, uid int, id int64, msg []byte, writeCh chan interface{}) error {
	var action dataBroadcastAction
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

func (b *dataBroadcast) OnClose(siteID string, id int64) {
	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()
	delete(siteP.Pool, id)
}

func BroadcastData(siteID string, d data.IData) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("fatal error broadcast panic: ", err)
		}
	}()

	toSend := make(map[int64]chan interface{})

	dataBroadcastInstance.lock.RLock()
	defer dataBroadcastInstance.lock.RUnlock()
	siteP, exists := dataBroadcastInstance.pool[siteID]

	if !exists {
		return
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for id, feed := range siteP.Pool {
		checked := false
		for _, s := range feed.StationID {
			if d.GetStationID() == s {
				checked = true
				break
			}
		}
		if !checked {
			continue
		}
		if feed.Flag != nil && len(feed.Flag) > 0 {
			checked = false
			for _, f := range feed.Flag {
				if d.GetFlag() == f {
					checked = true
					break
				}
			}
		}
		if !checked {
			continue
		}
		if feed.DataType != nil && len(feed.DataType) > 0 {
			checked = false
			for _, t := range feed.DataType {
				if d.GetDataType() == t {
					checked = true
					break
				}
			}
		}
		if !checked {
			continue
		}
		if feed.MonitorID != nil && len(feed.MonitorID) > 0 {
			checked = false
			for _, t := range feed.MonitorID {
				if d.GetMonitorID() == t {
					checked = true
					break
				}
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
	data["type"] = "dataBroadcast"
	data["dataType"] = d.GetDataType()
	data["data"] = d

	for _, writeCh := range toSend {
		writeCh <- data
	}
}

var reducerLock sync.RWMutex

var reducer = make(map[string]*siteReducer)

type siteReducer struct {
	lock sync.RWMutex
	pool map[int]*stationReducer
}

type stationReducer struct {
	lock sync.RWMutex
	pool map[string]*dataReducer
}

type dataReducer struct {
	lock     sync.Mutex
	topush   map[int][]data.IData
	silenced map[int]byte
	tocease  map[int][]data.IData

	schedule *time.Timer
}

func (r *dataReducer) scheduleConsume(siteID, dataType string, stationID int) {
	log.Println("schedule consume: ", siteID, dataType, stationID)
	if r.schedule == nil {
		r.schedule = time.AfterFunc(time.Second*5, func() {
			consumePush(siteID, stationID, dataType)
		})
	} else {
		if !r.schedule.Stop() {
			select {
			case <-r.schedule.C:
			default:
			}
		}
		r.schedule.Reset(time.Second * 5)
	}
}

func getPushReducer(siteID, dataType string, stationID int) (siteReduce *siteReducer, stationReduce *stationReducer, dataReduce *dataReducer) {

	var exists bool

	defer func() {
		if siteReduce != nil && stationReduce != nil && dataReduce != nil {
			return
		}
		if siteReduce == nil {
			reducerLock.Lock()
			defer reducerLock.Unlock()

			siteReduce, exists = reducer[siteID]
			if !exists {
				siteReduce = &siteReducer{pool: make(map[int]*stationReducer)}
				reducer[siteID] = siteReduce
			}
		} else {
			reducerLock.RLock()
			defer reducerLock.RUnlock()
		}

		if stationReduce == nil {
			siteReduce.lock.Lock()
			defer siteReduce.lock.Unlock()

			stationReduce, exists = siteReduce.pool[stationID]
			if !exists {
				stationReduce = &stationReducer{pool: make(map[string]*dataReducer)}
				siteReduce.pool[stationID] = stationReduce
			}
		} else {
			siteReduce.lock.RLock()
			defer siteReduce.lock.RUnlock()
		}

		if dataReduce == nil {
			stationReduce.lock.Lock()
			defer stationReduce.lock.Unlock()

			dataReduce, exists = stationReduce.pool[dataType]
			if !exists {
				dataReduce = &dataReducer{silenced: make(map[int]byte), topush: make(map[int][]data.IData), tocease: make(map[int][]data.IData)}
				stationReduce.pool[dataType] = dataReduce
			}
		}
	}()

	reducerLock.RLock()
	defer reducerLock.RUnlock()
	siteReduce, exists = reducer[siteID]
	if !exists {
		return
	}

	siteReduce.lock.RLock()
	defer siteReduce.lock.RUnlock()
	stationReduce, exists = siteReduce.pool[stationID]
	if !exists {
		return
	}

	stationReduce.lock.RLock()
	defer stationReduce.lock.RUnlock()
	dataReduce, exists = stationReduce.pool[dataType]

	return
}

func PushData(siteID string, d data.IData) error {
	flag, err := monitor.GetFlag(siteID, d.GetFlag())
	if err != nil {
		log.Println("error get flag: ", siteID, err)
		return err
	}

	if flag == nil {
		log.Println("flag nil")
		return nil
	}

	m, err := subscription.GetModule(siteID)
	if err != nil {
		return err
	}

	var pushSetting *subscription.PushSetting
	switch d.GetDataType() {
	case data.REAL_TIME:
		pushSetting = m.PushSettings[subscription.DATA_REAL_TIME]
	case data.MINUTELY:
		pushSetting = m.PushSettings[subscription.DATA_MINUTELY]
	case data.HOURLY:
		pushSetting = m.PushSettings[subscription.DATA_HOURLY]
	case data.DAILY:
		pushSetting = m.PushSettings[subscription.DATA_DAILY]
	}

	if pushSetting == nil || pushSetting.Trigger == nil {
		log.Println("no push setting: ", siteID, d.GetDataType())
		return nil
	}

	siteReduce, stationReduce, dataReduce := getPushReducer(siteID, d.GetDataType(), d.GetStationID())
	siteReduce.lock.RLock()
	defer siteReduce.lock.RUnlock()

	stationReduce.lock.RLock()
	defer stationReduce.lock.RUnlock()

	dataReduce.lock.Lock()
	defer dataReduce.lock.Unlock()

	if !monitor.CheckFlag(monitor.FLAG_PUSH, flag.Bits) {
		delete(dataReduce.topush, d.GetMonitorID())

		if pushSetting.Cease != nil {
			if list, exists := dataReduce.tocease[d.GetMonitorID()]; exists {
				if len(list) > 0 {
					if time.Time(d.GetDataTime()).Before(time.Time(list[0].GetDataTime())) {
						return nil
					}
				}
				list = append(list, d)

				dataReduce.tocease[d.GetMonitorID()] = list
				dataReduce.scheduleConsume(siteID, d.GetDataType(), d.GetStationID())
			}
		}

		return nil
	}

	if pushSetting.Cease != nil {
		if list, exists := dataReduce.tocease[d.GetMonitorID()]; exists && len(list) > 0 {
			dataReduce.tocease[d.GetMonitorID()] = make([]data.IData, 0)
		}
	}

	topush, exists := dataReduce.topush[d.GetMonitorID()]
	if !exists {
		topush = make([]data.IData, 0)
	}

	if len(topush) > 0 {
		if time.Time(d.GetDataTime()).Before(time.Time(topush[0].GetDataTime())) {
			log.Println("discard old data: ", siteID, d.GetDataTime(), topush[0].GetDataTime())
			return nil
		}
	}
	topush = append(topush, d)

	dataReduce.topush[d.GetMonitorID()] = topush

	dataReduce.scheduleConsume(siteID, d.GetDataType(), d.GetStationID())

	return nil
}

func consumePush(siteID string, stationID int, dataType string) error {

	m, err := subscription.GetModule(siteID)
	if err != nil {
		return err
	}

	var pushSetting *subscription.PushSetting
	switch dataType {
	case data.REAL_TIME:
		pushSetting = m.PushSettings[subscription.DATA_REAL_TIME]
	case data.MINUTELY:
		pushSetting = m.PushSettings[subscription.DATA_MINUTELY]
	case data.HOURLY:
		pushSetting = m.PushSettings[subscription.DATA_HOURLY]
	case data.DAILY:
		pushSetting = m.PushSettings[subscription.DATA_DAILY]
	}

	siteReduce, stationReduce, dataReduce := getPushReducer(siteID, dataType, stationID)
	siteReduce.lock.RLock()
	defer siteReduce.lock.RUnlock()

	stationReduce.lock.RLock()
	defer stationReduce.lock.RUnlock()

	dataReduce.lock.Lock()
	defer dataReduce.lock.Unlock()

	consumeTrigger(pushSetting, dataReduce, siteID, dataType, stationID)
	consumeCease(pushSetting, dataReduce, siteID, dataType, stationID)

	return nil
}

func consumeTrigger(pushSetting *subscription.PushSetting, dataReduce *dataReducer, siteID string, dataType string, stationID int) error {
	if pushSetting == nil || pushSetting.Trigger == nil {
		dataReduce.topush = make(map[int][]data.IData)
		return nil
	}

	list := make([]data.IData, 0)
	triggerMonitors := make(map[int]byte)

	for mid, pool := range dataReduce.topush {
		flags := make(map[string][]data.IData)

		for _, d := range pool {
			if _, exists := flags[d.GetFlag()]; !exists {
				flags[d.GetFlag()] = make([]data.IData, 0)
			}

			flags[d.GetFlag()] = append(flags[d.GetFlag()], d)
		}

		for f, ds := range flags {
			if pushSetting.Trigger.FlagThresholdCount == nil || len(ds) >= pushSetting.Trigger.FlagThresholdCount[f] {
				if _, pushed := dataReduce.silenced[mid]; !pushed {
					triggerMonitors[mid] = 1
					list = append(list, ds...)
				} else {
					log.Println("consume trigger silenced: ", siteID, dataType, stationID, mid)
				}
			} else {
				log.Println("consume trigger under threshold: ", siteID, dataType, stationID, mid, f, len(ds))
			}
		}
	}

	if len(triggerMonitors) == 0 {
		return nil
	}

	stations, err := entity.GetStation(siteID, stationID)
	if err != nil {
		log.Println("error push overproof data: ", err)
		return err
	}
	if len(stations) == 0 {
		log.Println("error push overproof data: station not found ", stationID)
		return nil
	}

	station := stations[0]

	if station.Status != entity.ACTIVE {
		log.Println("station not active")
		return nil
	}

	entities, err := entity.GetEntities(siteID, station.EntityID)
	if err != nil {
		log.Println("error push overproof data: ", err)
		return err
	}
	if len(entities) == 0 {
		log.Println("error push overproof data: entity not found ", station.EntityID)
		return nil
	}
	entity := entities[0]

	var subscriptionType string
	switch dataType {
	case data.DAILY:
		subscriptionType = subscription.DATA_DAILY
	case data.HOURLY:
		subscriptionType = subscription.DATA_HOURLY
	case data.MINUTELY:
		subscriptionType = subscription.DATA_MINUTELY
	case data.REAL_TIME:
		subscriptionType = subscription.DATA_REAL_TIME
	}
	subscriptionList, err := subscription.GetSubscriptionsToPush(siteID, station.EntityID, station.ID, subscriptionType)
	if err != nil {
		log.Println("error get subscription list to push: ", err)
		return err
	}

	if len(subscriptionList) == 0 {
		return nil
	}

	monitor.LoadMonitor(siteID)
	monitor.LoadFlagLimit(siteID)

	for _, sub := range subscriptionList {
		monitorSub := new(subscription.MonitorSubscription)
		monitorSub.Time = time.Time(list[0].GetDataTime())
		monitorSub.Subscription = *sub
		monitorSub.Entity = entity
		monitorSub.Station = station
		monitorSub.DataList = list

		if err := push.Push(siteID, monitorSub); err != nil {
			log.Println("error push overproof data: ", err)
		}
	}

	if pushSetting.Trigger.CooldownMin > 0 {
		for mid := range triggerMonitors {
			dataReduce.silenced[mid] = 1
			time.AfterFunc(pushSetting.Trigger.CooldownMin*time.Minute, func() {
				siteReduce, stationReduce, dataReduce := getPushReducer(siteID, dataType, stationID)
				siteReduce.lock.RLock()
				defer siteReduce.lock.RUnlock()

				stationReduce.lock.RLock()
				defer stationReduce.lock.RUnlock()

				dataReduce.lock.Lock()
				defer dataReduce.lock.Unlock()

				delete(dataReduce.silenced, mid)
			})

			if pushSetting.Cease != nil {
				dataReduce.tocease[mid] = make([]data.IData, 0)
			}
		}
	}

	return nil
}

func consumeCease(pushSetting *subscription.PushSetting, dataReduce *dataReducer, siteID string, dataType string, stationID int) error {
	log.Println("consume cease: ", siteID, stationID, dataType, len(dataReduce.tocease))
	if pushSetting == nil || pushSetting.Cease == nil {
		log.Println("consume cease setting nil: ", siteID, stationID, dataType)
		dataReduce.tocease = make(map[int][]data.IData)
		return nil
	}

	list := make([]data.IData, 0)
	triggerMonitors := make(map[int]byte)

	for mid, pool := range dataReduce.tocease {
		flags := make(map[string][]data.IData)

		for _, d := range pool {
			if _, exists := flags[d.GetFlag()]; !exists {
				flags[d.GetFlag()] = make([]data.IData, 0)
			}

			flags[d.GetFlag()] = append(flags[d.GetFlag()], d)
		}

		for f, ds := range flags {
			if pushSetting.Cease.FlagThresholdCount == nil || len(ds) >= pushSetting.Cease.FlagThresholdCount[f] {
				triggerMonitors[mid] = 1
				list = append(list, ds...)
			} else {
				log.Println("consume cease under threshold: ", siteID, dataType, stationID, mid, f, len(ds))
			}
		}
	}

	log.Println("consume cease monitors: ", len(triggerMonitors))
	if len(triggerMonitors) == 0 {
		return nil
	}

	stations, err := entity.GetStation(siteID, stationID)
	if err != nil {
		log.Println("error push overproof cease data: ", err)
		return err
	}
	if len(stations) == 0 {
		log.Println("error push overproof cease data: station not found ", stationID)
		return nil
	}

	station := stations[0]

	if station.Status != entity.ACTIVE {
		log.Println("station not active")
		return nil
	}

	entities, err := entity.GetEntities(siteID, station.EntityID)
	if err != nil {
		log.Println("error push overproof cease data: ", err)
		return err
	}
	if len(entities) == 0 {
		log.Println("error push overproof cease data: entity not found ", station.EntityID)
		return nil
	}
	entity := entities[0]

	var subscriptionType string
	switch dataType {
	case data.DAILY:
		subscriptionType = subscription.DATA_DAILY
	case data.HOURLY:
		subscriptionType = subscription.DATA_HOURLY
	case data.MINUTELY:
		subscriptionType = subscription.DATA_MINUTELY
	case data.REAL_TIME:
		subscriptionType = subscription.DATA_REAL_TIME
	}
	subscriptionList, err := subscription.GetSubscriptionsToPush(siteID, station.EntityID, station.ID, subscriptionType)
	if err != nil {
		log.Println("error get subscription list to push cease: ", err)
		return err
	}

	monitor.LoadMonitor(siteID)
	monitor.LoadFlagLimit(siteID)

	for _, sub := range subscriptionList {
		monitorSub := new(subscription.MonitorSubscription)
		monitorSub.Time = time.Time(list[0].GetDataTime())
		monitorSub.Subscription = *sub
		monitorSub.Entity = entity
		monitorSub.Station = station
		monitorSub.DataList = list
		monitorSub.IsCease = true

		if err := push.Push(siteID, monitorSub); err != nil {
			log.Println("error push cease overproof cease data: ", err)
		}
	}

	for mid := range triggerMonitors {
		delete(dataReduce.tocease, mid)
	}

	return nil
}
