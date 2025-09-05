package speaker

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/event"
	"obsessiontech/environment/peripheral"
	"obsessiontech/environment/role"
	"obsessiontech/environment/websocket"
)

var e_speaker_not_connected = errors.New("无在线设备")

var speakerBroadcastInstance *speakerBroadcast

func init() {
	speakerBroadcastInstance = new(speakerBroadcast)
	speakerBroadcastInstance.pool = make(map[string]*speakerBroadcastSitePool)
	websocket.RegisterHandler("speakerBroadcast", speakerBroadcastInstance)
}

type speakerBroadcastSitePool struct {
	Lock sync.RWMutex
	Pool map[int64]*speakerBroadcastFeed
}
type speakerBroadcastFeed struct {
	DeviceID   int
	WriteCH    chan interface{}
	ActionAuth authority.ActionAuthSet
}

type speakerBroadcast struct {
	lock sync.RWMutex
	pool map[string]*speakerBroadcastSitePool
}

type speakerBroadcastAction struct {
	websocket.Action
	Operation string `json:"operation"`
	DeviceID  int    `json:"deviceID"`
	EventID   int    `json:"eventID"`
	Msg       string `json:"msg"`
}

func (b *speakerBroadcast) subscribe(siteID, session string, uid int, action *speakerBroadcastAction, id int64, writeCh chan interface{}) error {
	if action.DeviceID <= 0 {
		return errors.New("需要指定设备ID")
	}

	if uid <= 0 {
		return websocket.E_need_login
	}

	actionAuth, err := role.GetAuthorityActions(siteID, peripheral.MODULE_PEREPHERAL, session, "", uid, peripheral.ACTION_ADMIN_VIEW, peripheral.ACTION_VIEW)
	if err != nil {
		return err
	}
	if len(actionAuth) == 0 {
		return websocket.E_need_login
	}

	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()

	feed := new(speakerBroadcastFeed)

	feed.DeviceID = action.DeviceID

	feed.WriteCH = writeCh
	feed.ActionAuth = actionAuth

	siteP.Pool[id] = feed

	return nil
}

func (b *speakerBroadcast) getSitePool(siteID string) *speakerBroadcastSitePool {
	b.lock.RLock()
	siteP, exists := b.pool[siteID]
	b.lock.RUnlock()

	if !exists {
		b.lock.Lock()
		defer b.lock.Unlock()

		siteP, exists = b.pool[siteID]

		if !exists {
			siteP = new(speakerBroadcastSitePool)
			siteP.Pool = make(map[int64]*speakerBroadcastFeed)

			b.pool[siteID] = siteP
		}
	}
	return siteP
}

func (b *speakerBroadcast) feedback(siteID string, eventID int, eventStatus, msg string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		events, err := event.GetEventWithTxn(siteID, txn, true, eventID)
		if err != nil {
			panic(err)
		}

		if len(events) == 0 {
			panic(errors.New("事件不存在"))
		}

		e := events[0]
		e.Feedback(eventStatus, map[string]interface{}{
			"at":  util.Time(time.Now()),
			"msg": msg,
		})

		if err := e.UpdateStatusWithTxn(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func (b *speakerBroadcast) OnMessage(siteID, session string, uid int, id int64, msg []byte, writeCh chan interface{}) error {
	var action speakerBroadcastAction
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
	case "inprogress":
		return b.feedback(siteID, action.EventID, event.IN_PROGRESS, action.Msg)
	case "success":
		return b.feedback(siteID, action.EventID, event.SUCCESS, action.Msg)
	case "fail":
		return b.feedback(siteID, action.EventID, event.FAIL, action.Msg)
	default:
		return errors.New("operation not support")
	}

	return nil
}

func (b *speakerBroadcast) OnClose(siteID string, id int64) {
	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()
	delete(siteP.Pool, id)
}

func BroadcastSound(siteID string, eventID, deviceID int, resourceURL, resourceURI string, repeat int) error {

	defer func() {
		if err := recover(); err != nil {
			log.Println("fatal error broadcast panic: ", err)
		}
	}()

	toSend := make(map[int64]chan interface{})

	speakerBroadcastInstance.lock.RLock()
	defer speakerBroadcastInstance.lock.RUnlock()
	siteP, exists := speakerBroadcastInstance.pool[siteID]

	if !exists {
		return e_speaker_not_connected
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for id, feed := range siteP.Pool {
		if feed.DeviceID == deviceID {
			toSend[id] = feed.WriteCH
		}
	}

	if len(toSend) == 0 {
		return e_speaker_not_connected
	}

	data := make(map[string]interface{})
	data["type"] = "speakerBroadcast"
	data["eventID"] = eventID
	if resourceURL != "" {
		data["resourceURL"] = resourceURL
	}
	if resourceURI != "" {
		data["resourceURI"] = resourceURI
	}
	data["repeat"] = repeat

	for _, writeCh := range toSend {
		writeCh <- data
	}

	return nil
}

func GetSpeakerStatus(siteID string, deviceID ...int) (map[int]bool, error) {
	result := make(map[int]bool)

	speakerBroadcastInstance.lock.RLock()
	defer speakerBroadcastInstance.lock.RUnlock()
	siteP, exists := speakerBroadcastInstance.pool[siteID]

	if !exists {
		return result, nil
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for _, did := range deviceID {
		for _, feed := range siteP.Pool {
			if feed.DeviceID == did {
				result[did] = true
			}
		}
	}

	return result, nil
}
