package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	myContext "obsessiontech/common/context"
)

var e_need_wait = errors.New("need waiting")
var e_lock_timeout = errors.New("lock timeout")
var e_lock_fail = errors.New("fail to acquire lock")

type GlobalLocker struct {
	siteLockers map[string]*siteGlobalLocker
	lock        sync.RWMutex
}

type siteGlobalLocker struct {
	pool map[string]*lockerContext
	lock sync.RWMutex
}

type lockerContext struct {
	requestID string
	ctx       context.Context
	count     uint64
	cancel    context.CancelFunc
}

func (locker *siteGlobalLocker) Wait(ctx context.Context, requestID string, key ...string) {

	list := make([]context.Context, 0)

	defer func() {
		if len(list) > 0 {
			log.Println("wait for locked key: ", len(list))
		}
		for _, wait := range list {
			select {
			case <-ctx.Done():
				log.Println("wait timeout")
				return
			case <-wait.Done():
				log.Println("wait done")
			}
		}
	}()

	locker.lock.RLock()
	defer locker.lock.RUnlock()

	for _, k := range key {
		if ctx, exists := locker.pool[k]; exists && ctx.requestID != requestID {
			log.Println("found exists lock: ", k, ctx.requestID)
			list = append(list, ctx.ctx)
		}
	}
}

func (locker *siteGlobalLocker) Lock(ctx context.Context, requestID string, key []string) (map[string]*lockerContext, error) {
	select {
	case <-ctx.Done():
		log.Println("timeout before get lock")
		return nil, e_lock_timeout
	default:
	}

	locker.lock.Lock()
	defer locker.lock.Unlock()

	select {
	case <-ctx.Done():
		log.Println("timeout after get lock")
		return nil, e_lock_timeout
	default:
	}

	for _, k := range key {
		if ctx, exists := locker.pool[k]; exists && ctx.requestID != requestID {
			return nil, e_need_wait
		}
	}

	result := make(map[string]*lockerContext)

	for _, k := range key {
		if ctx, exists := locker.pool[k]; exists {
			log.Println("lock target exists")
			if ctx.requestID == requestID {
				result[k] = ctx
			} else {
				return nil, fmt.Errorf("fail to lock: lock exists: key[%s] requestID[%s]", k, requestID)
			}
		} else {
			log.Println("lock target not exists")
			ctx, cancel := myContext.GetContext()
			lockerCtx := &lockerContext{requestID: requestID, ctx: ctx, cancel: cancel}
			locker.pool[k] = lockerCtx
			result[k] = lockerCtx
		}

		atomic.AddUint64(&(result[k].count), 1)
	}

	return result, nil
}

func (locker *siteGlobalLocker) UnLock(locked map[string]*lockerContext) {

	locker.lock.Lock()
	defer locker.lock.Unlock()

	for key, lockedCtx := range locked {
		current, exists := locker.pool[key]
		if !exists {
			log.Printf("warning unlock global lock not exists: key[%s] target[%s]", key, lockedCtx.requestID)
			continue
		}

		if current != lockedCtx {
			log.Printf("warning unlock global lock not same: current[%s], key[%s], target[%s]", current.requestID, key, lockedCtx.requestID)
			continue
		}

		remain := atomic.AddUint64(&(current.count), ^uint64(0))

		if remain == 0 {
			current.cancel()
			delete(locker.pool, key)
			log.Println("unlock global lock: ", current.requestID, key)
		}
	}
}

func (g *GlobalLocker) Host(connType, local string) error {
	connChan, err := StartHost(connType, local)
	if err != nil {
		return err
	}

	g.siteLockers = make(map[string]*siteGlobalLocker)

	go func() {
		for {
			select {
			case conn, ok := <-connChan:
				if !ok {
					log.Println("global lock host closed")
					return
				}

				go g.serve(conn)
			}
		}
	}()

	return nil
}

type globalLockReq struct {
	SiteID    string        `json:"siteID"`
	RequestID string        `json:"requestID"`
	LockKeys  []string      `json:"lockKeys"`
	Timeout   time.Duration `json:"timeout"`
}

type globalLockRet struct {
	RetCode   int      `json:"retCode"`
	RetMsg    string   `json:"retMsg,omitempty"`
	SiteID    string   `json:"siteID,omitempty"`
	RequestID string   `json:"requestID,omitempty"`
	LockKeys  []string `json:"lockKeys,omitempty"`
}

func (g *GlobalLocker) getSitePool(siteID string) (result *siteGlobalLocker) {

	defer func() {
		if result == nil {
			g.lock.Lock()
			defer g.lock.Unlock()

			result = g.siteLockers[siteID]
			if result == nil {
				result = new(siteGlobalLocker)
				result.pool = make(map[string]*lockerContext)
				g.siteLockers[siteID] = result
			}
		}
	}()

	g.lock.RLock()
	defer g.lock.RUnlock()

	result = g.siteLockers[siteID]
	return
}

func (g *GlobalLocker) respond(conn *Connection, data interface{}) error {
	res, _ := json.Marshal(data)
	return Write(conn.Conn, res)
}

func (g *GlobalLocker) serve(conn *Connection) {
	for {
		select {
		case <-conn.Ctx.Done():
			return
		default:
		}

		data, client, err := Receive(conn.Conn)
		if err != nil {
			log.Println("error: ", client, err)
			g.respond(conn, &globalLockRet{RetCode: 500, RetMsg: err.Error()})
			conn.Cancel()
			return
		}

		for _, d := range data {

			dd := d
			go func() {
				log.Println("global lock serve: ", string(dd))
				var req globalLockReq
				if err := json.Unmarshal(dd, &req); err != nil {
					log.Println("error: ", client, err)
					g.respond(conn, &globalLockRet{RetCode: 500, RetMsg: err.Error()})
					conn.Cancel()
					return
				}

				if err := g.processLockReq(conn, &req); err != nil {
					log.Println("error: ", client, err)
					g.respond(conn, &globalLockRet{RetCode: 500, RetMsg: err.Error()})
					conn.Cancel()
					return
				}
			}()
		}
	}
}

func (g *GlobalLocker) processLockReq(conn *Connection, req *globalLockReq) error {
	if req.SiteID == "" {
		return errors.New("no site id")
	}

	if len(req.LockKeys) == 0 {
		return errors.New("no key")
	}

	locker := g.getSitePool(req.SiteID)

	log.Println("locker: ", len(locker.pool))

	if req.Timeout == 0 {
		req.Timeout = time.Second * 30
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, req.Timeout)
	defer func() {
		cancel()
	}()

	deadline, ok := ctx.Deadline()
	log.Println("request lock ctx: ", time.Now(), deadline, ok)

	for {
		locker.Wait(ctx, req.RequestID, req.LockKeys...)
		lockedCtx, err := locker.Lock(ctx, req.RequestID, req.LockKeys)
		if err != nil {
			log.Println("error serve global lock: ", req.RequestID, err)
			if err == e_need_wait {
				continue
			}
			return err
		}

		defer func() {
			locker.UnLock(lockedCtx)
			log.Println("unlock global lock: ", req.RequestID)
		}()
		break
	}

	log.Println("lock global lock: ", req.RequestID)
	g.respond(conn, &globalLockRet{RetCode: 0})

	<-conn.Ctx.Done()
	log.Println("global lock client done: ", req.RequestID)

	return nil
}

func RequestGlobalLock(connType, remote string, siteID, requestID string, lockKey []string, timeout time.Duration, action func() error) error {

	log.Println("request lock: ", requestID)

	if siteID == "" {
		return errors.New("no site id")
	}

	if requestID == "" {
		return errors.New("no request id")
	}

	if lockKey == nil || len(lockKey) == 0 {
		return errors.New("no lock keys")
	}

	conn, err := StartClient(connType, remote)
	if err != nil {
		return err
	}

	log.Println("request lock client established: ", requestID)

	defer func() {
		conn.Cancel()
	}()

	req := new(globalLockReq)
	req.SiteID = siteID
	req.RequestID = requestID
	req.LockKeys = make([]string, 0)
	req.Timeout = timeout

	for _, k := range lockKey {
		req.LockKeys = append(req.LockKeys, k)
	}

	data, _ := json.Marshal(req)

	if err := Write(conn.Conn, data); err != nil {
		return err
	}

	log.Println("request lock sent: ", requestID)

	res := make(chan Datagram)

	go func() {
		data, _, err := Receive(conn.Conn)
		if err != nil {
			conn.Cancel()
			return
		}

		for _, d := range data {
			res <- d
		}
	}()

	select {
	case <-conn.Ctx.Done():
		return e_lock_fail
	case data := <-res:
		var ret globalLockRet
		if err := json.Unmarshal(data, &ret); err != nil {
			return err
		}

		if ret.RetCode != 0 {
			return errors.New(ret.RetMsg)
		}

		if err := action(); err != nil {
			return err
		}

		select {
		case <-conn.Ctx.Done():
			log.Println("lost lock connection after action")
			return e_lock_fail
		default:
		}

	}

	return nil
}
