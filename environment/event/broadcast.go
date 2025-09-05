package event

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/peripheral"
	"obsessiontech/environment/role"
	"obsessiontech/environment/websocket"
)

var eventBroadcastInstance *eventBroadcast

func init() {
	eventBroadcastInstance = new(eventBroadcast)
	eventBroadcastInstance.pool = make(map[string]*eventBroadcastSitePool)
	websocket.RegisterHandler("eventBroadcast", eventBroadcastInstance)
}

type eventBroadcastSitePool struct {
	Lock sync.RWMutex
	Pool map[int64]*eventBroadcastFeed
}
type eventBroadcastFeed struct {
	EventID    []int
	WriteCH    chan interface{}
	ActionAuth authority.ActionAuthSet
}

type eventBroadcast struct {
	lock sync.RWMutex
	pool map[string]*eventBroadcastSitePool
}

type eventBroadcastAction struct {
	websocket.Action
	Operation string `json:"operation"`
	EventID   []int  `json:"eventID"`
}

func (b *eventBroadcast) getSitePool(siteID string) *eventBroadcastSitePool {
	b.lock.RLock()
	siteP, exists := b.pool[siteID]
	b.lock.RUnlock()

	if !exists {
		b.lock.Lock()
		defer b.lock.Unlock()

		siteP, exists = b.pool[siteID]

		if !exists {
			siteP = new(eventBroadcastSitePool)
			siteP.Pool = make(map[int64]*eventBroadcastFeed)

			b.pool[siteID] = siteP
		}
	}
	return siteP
}

func (b *eventBroadcast) subscribe(siteID, session string, uid int, action *eventBroadcastAction, id int64, writeCh chan interface{}) error {
	if len(action.EventID) == 0 {
		return errors.New("需要指定事件ID")
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

	feed := new(eventBroadcastFeed)

	feed.EventID = action.EventID

	feed.WriteCH = writeCh
	feed.ActionAuth = actionAuth

	siteP.Pool[id] = feed

	return nil
}

func (b *eventBroadcast) OnMessage(siteID, session string, uid int, id int64, msg []byte, writeCh chan interface{}) error {
	var action eventBroadcastAction
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

func (b *eventBroadcast) OnClose(siteID string, id int64) {
	siteP := b.getSitePool(siteID)
	siteP.Lock.Lock()
	defer siteP.Lock.Unlock()
	delete(siteP.Pool, id)
}

func BroadcastEvent(siteID string, event *Event) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("fatal error broadcast panic: ", err)
		}
	}()

	toSend := make(map[int64]chan interface{})

	eventBroadcastInstance.lock.RLock()
	defer eventBroadcastInstance.lock.RUnlock()
	siteP, exists := eventBroadcastInstance.pool[siteID]

	if !exists {
		return
	}

	siteP.Lock.RLock()
	defer siteP.Lock.RUnlock()

	for id, feed := range siteP.Pool {
		for _, eid := range feed.EventID {
			if eid == event.ID {
				toSend[id] = feed.WriteCH
				break
			}
		}
	}

	if len(toSend) == 0 {
		return
	}

	data := make(map[string]interface{})
	data["type"] = "eventBroadcast"
	data["event"] = event

	for _, writeCh := range toSend {
		writeCh <- data
	}
}
