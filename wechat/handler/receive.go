package handler

import (
	"log"
	"obsessiontech/wechat/handler/echo"
	"obsessiontech/wechat/message"
)

func Receive(data []byte) interface{} {

	msg, err := message.Receive(data)
	if err != nil {
		log.Println(err)
		return nil
	}

	switch msg.GetMsgType() {
	case message.MSG_TEXT:
		text := msg.(*message.TextMessage)
		log.Println(text)
		return echo.DealTextEcho(text)
	}

	return nil
}
