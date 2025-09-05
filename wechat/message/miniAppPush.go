package message

import (
	"log"
	"encoding/json"
)

const MINIAPP_MSG_TXT = "text"
const MINIAPP_MSG_IMG = "image"
const MINIAPP_MSG_PG = "miniprogrampage"
const MINIAPP_MSG_EVENT = "event"

const MINIAPP_EVENT_CONTACT = "user_enter_tempsession"

type MiniAppMessage interface {}

type MiniAppTxtMessage struct {
	ToUserName string
    FromUserName string
    CreateTime int
    MsgType string
    Content string
    MsgID int
}

type MiniAppImgMessage struct {
	ToUserName string
    FromUserName string
    CreateTime int
    MsgType string
    PicURL string
    MediaID string
    MsgID int
}

type MiniAppPgMessage struct {
	ToUserName string
    FromUserName string
    CreateTime int
    MsgType string
    MsgID int
    Title string
    AppID string
    PagePath string
    ThumbURL string
    ThumbMediaID string
}

type MiniAppEvent struct {
	ToUserName string
    FromUserName string
    CreateTime int
    MsgType string
    MsgID int
    Event string
    SessionFrom string
}

func ReceiveMiniAppPush(data []byte) (MiniAppMessage, error) {
	log.Println("receive mini app push: ", string(data))
	filter := make(map[string]interface{})
	err := json.Unmarshal(data, &filter)
	if err != nil {
		return nil, err
	}

	msgType, exists := filter["MsgType"]
	if !exists {
		return nil, INVALID_MSG
	}

	switch msgType {
		case MINIAPP_MSG_TXT:
			var ret MiniAppTxtMessage
			if err := json.Unmarshal(data, &ret); err != nil {
				return nil, err
			}
			return ret, nil
		case MINIAPP_MSG_IMG:
			var ret MiniAppImgMessage
			if err := json.Unmarshal(data, &ret); err != nil {
				return nil, err
			}
			return ret, nil
		case MINIAPP_MSG_PG:
			var ret MiniAppPgMessage
			if err := json.Unmarshal(data, &ret); err != nil {
				return nil, err
			}
			return ret, nil
		case MINIAPP_MSG_EVENT:
			var ret MiniAppEvent
			if err := json.Unmarshal(data, &ret); err != nil {
				return nil, err
			}
			return ret, nil
		default:
			return nil, NOT_SUPPORT
	}
}