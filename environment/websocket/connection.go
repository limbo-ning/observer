package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"obsessiontech/environment/user/auth"

	"github.com/gorilla/websocket"
)

var E_need_login = errors.New("请登录")
var E_not_authorized = errors.New("权限不足")
var E_uri_not_support = errors.New("不支持的服务")

func upgrade() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
}

type MessageHandler interface {
	OnMessage(siteID, session string, uid int, id int64, msg []byte, writeCh chan interface{}) error
	OnClose(siteID string, id int64)
}

var handlers = make(map[string]MessageHandler)

type Action struct {
	URI string `json:"uri"`
}

func RegisterHandler(uri string, handler MessageHandler) {
	handlers[uri] = handler
}

func closeConn(siteID string, id int64) {
	for _, handler := range handlers {
		handler.OnClose(siteID, id)
	}
}

func Handle(w http.ResponseWriter, r *http.Request, siteID, clientIP, headerToken string) {

	var session string
	var uid int

	upgrader := upgrade()

	clientSecToken := r.Header.Get("Sec-WebSocket-Protocol")
	if clientSecToken != "" {
		upgrader.Subprotocols = []string{clientSecToken}
		uid, session = auth.IsLogined(siteID, clientIP, clientSecToken, auth.REFERER_NA)
	}

	if uid <= 0 && headerToken != "" {
		uid, session = auth.IsLogined(siteID, clientIP, headerToken, auth.REFERER_NA)
	}

	log.Println("handle socket: ", uid, session)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error fail to establish websocket: ", err)
		return
	}

	id := time.Now().UnixNano()

	defer func() {
		log.Println("close connection")
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close"), time.Now().Add(time.Second))
		conn.Close()
		if err := recover(); err != nil {
			log.Println("fatal error recover: ", err)
			closeConn(siteID, id)
		}
	}()

	conn.SetPingHandler(func(message string) error {
		conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(time.Second))
		return nil
	})

	writeCh := make(chan interface{})
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("fatal error recover: ", err)
				closeConn(siteID, id)
			}
		}()

		for {
			select {
			case out, isOpen := <-writeCh:
				if out != nil {
					err := conn.WriteJSON(out)
					if err != nil {
						if isOpen {
							close(writeCh)
						}
						return
					}
				}
				if !isOpen {
					return
				}
			case <-time.After(time.Second * 30):
				conn.WriteControl(websocket.PingMessage, []byte(""), time.Now().Add(time.Second))
			}

		}
	}()

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("read message error: ", err)
			closeConn(siteID, id)
			close(writeCh)
			return
		}
		log.Printf("incoming msg type:%d msg:%s", msgType, string(msg))

		switch msgType {
		case websocket.TextMessage, websocket.BinaryMessage:
			route(siteID, session, uid, msg, id, writeCh)
		case websocket.CloseMessage:
			closeConn(siteID, id)
			close(writeCh)
			return
		case websocket.PingMessage:
		case websocket.PongMessage:
		}
	}
}

func route(siteID, session string, uid int, msg []byte, id int64, writeCh chan interface{}) {

	var uri Action

	if err := json.Unmarshal(msg, &uri); err != nil {
		writeCh <- badRequest(err)
	}

	if handler, exists := handlers[uri.URI]; exists {
		if err := handler.OnMessage(siteID, session, uid, id, msg, writeCh); err != nil {
			writeCh <- badRequest(err)
		}
	} else {
		writeCh <- badRequest(E_uri_not_support)
	}
}

func badRequest(err error) *map[string]interface{} {
	result := make(map[string]interface{})
	if err == E_need_login {
		result["retCode"] = 1
	} else if err == E_not_authorized {
		result["retCode"] = 403
	} else {
		result["retCode"] = 500
	}
	result["retMsg"] = err.Error()

	return &result
}
