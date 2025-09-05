package echo

import (
	"log"
	"math/rand"
	"obsessiontech/wechat/message"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

var reg *regexp.Regexp

var pool map[string]*[]string
var size map[string]*int32
var unique map[string]*map[string]byte

var random *rand.Rand

func init() {
	reg = regexp.MustCompile(`[\w|\d]`)
	pool = make(map[string]*[]string)
	size = make(map[string]*int32)
	unique = make(map[string]*map[string]byte)
	for _, setting := range echoConfig.Echo {
		list := make([]string, setting.PoolSize, setting.PoolSize)
		pool[setting.OpenAccount] = &list
		var i int32
		size[setting.OpenAccount] = &i
		u := make(map[string]byte)
		unique[setting.OpenAccount] = &u
	}

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func DealTextEcho(msg *message.TextMessage) *message.TextMessage {
	for _, setting := range echoConfig.Echo {
		if shouldEcho(msg, setting) {
			return doEcho(msg, setting)
		}
	}

	return nil
}

func shouldEcho(msg *message.TextMessage, setting *Setting) bool {
	if msg.ToUserName != setting.OpenAccount {
		log.Printf("to not match. to[%s] setting[%s]", msg.ToUserName, setting.OpenAccount)
		return false
	}

	if !setting.Enable {
		log.Printf("not enabled")
		return false
	}

	if len(setting.EnableInterval) > 0 {
		now := time.Now()
		if now.Hour() < setting.EnableInterval[0] || now.Hour() >= setting.EnableInterval[1] {
			log.Printf("enable time not match. begin[%d] end[%d]", setting.EnableInterval[0], setting.EnableInterval[1])
			return false
		}
	}

	if !strings.Contains(msg.Content, setting.Key) {
		log.Printf("not contain key [%s]", setting.Key)
		return false
	}

	if len(strings.Replace(msg.Content, setting.Key, "", -1)) < 5 {
		log.Printf("too short")
		return false
	}

	if len(reg.FindAllString(msg.Content, -1)) > 5 {
		log.Printf("too many letter or number")
		return false
	}

	return true
}

func doEcho(msg *message.TextMessage, setting *Setting) *message.TextMessage {

	var res message.TextMessage
	res.FromUserName = msg.ToUserName
	res.ToUserName = msg.FromUserName
	res.MsgType = msg.MsgType
	res.CreateTime = msg.CreateTime + 1

	list := (*pool[setting.OpenAccount])
	u := (*unique[setting.OpenAccount])

	curSize := atomic.LoadInt32(size[setting.OpenAccount])
	var r int
	if curSize > 0 {
		r = random.Intn(int(curSize))
	}

	if _, exists := u[msg.Content]; !exists {
		if curSize < setting.PoolSize {
			atomic.AddInt32(size[setting.OpenAccount], 1)
			list[curSize] = msg.Content
			if curSize == 0 {
				return nil
			}
			res.Content = list[r]
		} else {
			res.Content = list[r]
			list[r] = msg.Content
			delete(u, res.Content)
		}
		u[msg.Content] = 1
	} else {
		res.Content = list[r]
	}

	log.Printf("current size:%d", (*size[setting.OpenAccount]))

	return &res
}
