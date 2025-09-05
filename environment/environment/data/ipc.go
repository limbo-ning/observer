package data

import (
	"encoding/json"
	"errors"
	"log"
	"obsessiontech/common/ipc"
)

const (
	ack = iota

	listenReq

	triggerRotationReq
	archiveActivateReq
	triggerArchiveRollbackReq
	clearArchiveEntryReq
)

type IMessage interface {
	GetIPCMessageType() int
}

type Ack int

func (m *Ack) GetIPCMessageType() int { return ack }

type ListenReq struct{}

func (m *ListenReq) GetIPCMessageType() int { return listenReq }

type TriggerRotationReq struct {
	SiteID    string `json:"siteID"`
	Immediate bool   `json:"immediate"`
}

func (m *TriggerRotationReq) GetIPCMessageType() int { return triggerRotationReq }

type ArchiveActivateReq struct {
	SiteID   string `json:"siteID"`
	DataType string `json:"dataType"`
	Table    string `json:"table"`
}

func (m *ArchiveActivateReq) GetIPCMessageType() int { return archiveActivateReq }

type TriggerArchiveRollbackReq struct {
	SiteID   string `json:"siteID"`
	DataType string `json:"dataType"`
	Table    string `json:"table"`
}

func (m *TriggerArchiveRollbackReq) GetIPCMessageType() int { return triggerArchiveRollbackReq }

type ClearArchiveEntryReq struct {
	SiteID   string `json:"siteID"`
	DataType string `json:"dataType"`
}

func (m *ClearArchiveEntryReq) GetIPCMessageType() int { return clearArchiveEntryReq }

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
	case ack:
		message = new(Ack)
	case listenReq:
		message = new(ListenReq)
	case triggerRotationReq:
		message = new(TriggerRotationReq)
	case archiveActivateReq:
		message = new(ArchiveActivateReq)
	case triggerArchiveRollbackReq:
		message = new(TriggerArchiveRollbackReq)
	case clearArchiveEntryReq:
		message = new(ClearArchiveEntryReq)
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

func send(conn *ipc.Connection, message IMessage) error {
	data, err := WrapMessage(message)
	if err != nil {
		return err
	}
	_, err = conn.Conn.Write(data)
	return err
}

func hostListen(conn *ipc.Connection) {
	for {
		select {
		case <-conn.Ctx.Done():
			rotationPersistentClientLock.Lock()
			delete(rotationPersistentClient, conn)
			rotationPersistentClientLock.Unlock()
			return
		default:
		}

		datagrams, _, err := ipc.Receive(conn.Conn)
		if err != nil {
			log.Println("error read environment archive ipc request: ", err)
			return
		}

		for _, d := range datagrams {
			message, err := ParseMessage(d)
			if err != nil {
				log.Println("error environment archive ipc message: ", err)
				continue
			}

			log.Println("receive environment archive ipc message:", message.GetIPCMessageType())

			var res IMessage

			switch req := message.(type) {
			case (*Ack):
				continue
			case (*ListenReq):
				var ackRes Ack
				ackRes = 0

				rotationPersistentClientLock.Lock()
				rotationPersistentClient[conn] = true
				rotationPersistentClientLock.Unlock()

				res = &ackRes
			case (*TriggerRotationReq):
				var ackRes Ack
				ackRes = 0
				TriggerRotation(req.SiteID, req.Immediate)
				res = &ackRes
			case (*ArchiveActivateReq):
				var ackRes Ack
				ackRes = 0
				if err := ActivateArchive(req.SiteID, req.DataType, req.Table); err != nil {
					ackRes = 1
				}
				res = &ackRes
			case (*TriggerArchiveRollbackReq):
				var ackRes Ack
				ackRes = 0
				TriggerArchiveRollback(req.SiteID, req.DataType, req.Table)
				res = &ackRes
			default:
				continue
			}

			if res != nil {
				data, err := WrapMessage(res)
				if err != nil {
					log.Println("error environment archive wrap ipc res: ", res.GetIPCMessageType(), err)
					continue
				}
				if err := ipc.Write(conn.Conn, data); err != nil {
					conn.Cancel()
					return
				}
			}
		}
	}
}

func clientListen(conn *ipc.Connection) {
	for {
		select {
		case <-conn.Ctx.Done():
			return
		default:
		}

		datagrams, _, err := ipc.Receive(conn.Conn)
		if err != nil {
			log.Println("error read environment archive ipc request: ", err)
			return
		}

		for _, d := range datagrams {
			message, err := ParseMessage(d)
			if err != nil {
				log.Println("error environment archive ipc message: ", err)
				continue
			}

			log.Println("receive environment archive ipc message:", message.GetIPCMessageType())

			var res IMessage

			switch req := message.(type) {
			case *Ack:
				continue
			case *ClearArchiveEntryReq:
				var ackRes Ack
				ackRes = 0
				ClearArchiveTable(req.SiteID, req.DataType)
				res = &ackRes
			default:
				continue
			}

			if res != nil {
				data, err := WrapMessage(res)
				if err != nil {
					log.Println("error environment archive wrap ipc res: ", res.GetIPCMessageType(), err)
					continue
				}
				if err := ipc.Write(conn.Conn, data); err != nil {
					conn.Cancel()
					return
				}
			}
		}
	}
}
