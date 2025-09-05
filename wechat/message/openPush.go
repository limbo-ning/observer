package message

import (
	"encoding/xml"
	"errors"
)

var NOT_SUPPORT = errors.New("message not support")
var INVALID_MSG = errors.New("invalid message")

const MSG_EVENT = "event"
const MSG_TEXT = "text"
const MSG_IMAGE = "image"
const MSG_VOICE = "voice"
const MSG_VIDEO = "video"
const MSG_SHORTVIDEO = "shortvideo"

type Message interface {
	GetMsgType() string
	GetEvent() string
}

type BaseMessage struct {
	XMLName      xml.Name `xml:"xml"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int      `xml:"CreateTime"`
	MsgID        int      `xml:"MsgID,omitempty"`
}

func (this *BaseMessage) GetMsgType() string {
	return this.MsgType
}
func (this *BaseMessage) GetEvent() string {
	return this.Event
}

type TextMessage struct {
	BaseMessage
	Content string `xml:"Content"`
}

type EventMessage struct {
	BaseMessage
	Status string `xml:"Status"`
}

func Receive(data []byte) (Message, error) {
	var filter BaseMessage
	err := xml.Unmarshal(data, &filter)
	if err != nil {
		return nil, err
	}

	switch filter.MsgType {
	case MSG_TEXT:
		var text TextMessage
		err := xml.Unmarshal(data, &text)
		if err != nil {
			return nil, err
		}
		return &text, nil
	case MSG_EVENT:
		var event EventMessage
		err := xml.Unmarshal(data, &event)
		if err != nil {
			return nil, err
		}
		return &event, nil
	default:
		return nil, NOT_SUPPORT
	}
}
