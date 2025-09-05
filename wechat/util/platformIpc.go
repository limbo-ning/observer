package util

import (
	"encoding/json"
	"errors"
	"log"
	"obsessiontech/common/ipc"
	"time"
)

const (
	componentAccessTokenReq = iota
	componentAccessTokenRes
)

type IMessage interface {
	GetIPCMessageType() int
}

type ComponentAccessTokenReq struct{}

func (m *ComponentAccessTokenReq) GetIPCMessageType() int { return componentAccessTokenReq }

type ComponentAccessTokenRes string

func (m *ComponentAccessTokenRes) GetIPCMessageType() int { return componentAccessTokenRes }

var E_unmarshal_failure = errors.New("json unmarshal failed")
var E_message_type_unknown = errors.New("message type unknown")

func ParseMessage(input []byte) (IMessage, error) {
	var datagram struct {
		MessageType int             `json:"type"`
		Message     json.RawMessage `json:"message"`
	}

	if err := json.Unmarshal(input, &datagram); err != nil {
		return nil, E_unmarshal_failure
	}

	var message IMessage

	switch datagram.MessageType {
	case componentAccessTokenReq:
		message = new(ComponentAccessTokenReq)
	case componentAccessTokenRes:
		message = new(ComponentAccessTokenRes)
	}

	if err := json.Unmarshal(datagram.Message, &message); err != nil {
		return nil, E_unmarshal_failure
	}

	return message, nil
}

func WrapMessage(message IMessage) ([]byte, error) {

	datagram := map[string]interface{}{
		"type":    message.GetIPCMessageType(),
		"message": message,
	}

	return json.Marshal(datagram)
}

func listen(conn *ipc.Connection) {
	for {
		select {
		case <-conn.Ctx.Done():
			return
		default:
		}

		datagrams, _, err := ipc.Receive(conn.Conn)
		if err != nil {
			log.Println("error read wechat platform ipc request: ", err)
			conn.Cancel()
			return
		}

		for _, d := range datagrams {
			message, err := ParseMessage(d)
			if err != nil {
				log.Println("error wechat platform ipc message: ", err)
				continue
			}

			log.Println("receive wechat platform ipc message:", message.GetIPCMessageType())

			var res IMessage

			switch message.(type) {
			case (*ComponentAccessTokenReq):
				token, err := GetComponentAccessToken()
				if err != nil {
					conn.Cancel()
					return
				}

				tokenRes := ComponentAccessTokenRes(token)
				res = &tokenRes
			default:
				continue
			}

			data, err := WrapMessage(res)
			if err != nil {
				log.Println("error wechat platform wrap ipc res: ", res.GetIPCMessageType(), err)
				continue
			}
			if err := ipc.Write(conn.Conn, data); err != nil {
				conn.Cancel()
				return
			}
		}
	}
}

func client(connType, addr string, output <-chan IMessage) (<-chan IMessage, error) {
	log.Println("wechat platform request client connecting: ", addr)

	conn, err := ipc.StartClient(connType, addr)
	if err != nil {
		log.Println("error start wechat platform request client: ", addr, err)
		select {
		case <-output:
		default:
		}
		return nil, err
	}

	input := make(chan IMessage)

	go func() {
		for {
			select {
			case <-conn.Ctx.Done():
				log.Println("wechat platform  request client ctx done")
				conn.Cancel()
				return
			case toSend, ok := <-output:
				if !ok {
					log.Println("wechat platform  request client output closed")
					conn.Cancel()
					return
				}
				data, err := WrapMessage(toSend)
				if err != nil {
					log.Println("error wrap wechat platform ipc data: ", toSend.GetIPCMessageType(), err)
					continue
				}
				if err := ipc.Write(conn.Conn, data); err != nil {
					conn.Cancel()
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-conn.Ctx.Done():
				log.Println("wechat platform  request client ctx done")
				conn.Cancel()
				close(input)
				return
			default:
			}

			datagrams, _, err := ipc.Receive(conn.Conn)
			if err != nil {
				log.Println("request wechat platform client receive err: ", err)
				return
			}

			for _, d := range datagrams {
				message, err := ParseMessage(d)
				if err != nil {
					log.Println("error wechat platform  ipc message: ", err)
					continue
				}

				select {
				case input <- message:
				case <-time.After(time.Second * 10):
					log.Println("wechat platform request client input channel receive timeout")
				}
			}

		}
	}()

	log.Println("wechat platform  request client established: ", addr)

	return input, nil
}
