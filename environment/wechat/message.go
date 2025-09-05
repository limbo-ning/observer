package wechat

import (
	"log"
	"strings"
	"time"

	"obsessiontech/wechat/message"
	"obsessiontech/wechat/util"
)

func ReceiveMessagePush(appID string, timestamp int, msgSignature, nonce, encryptType string, data []byte) (string, []byte, error) {

	decrypted, err := message.PlatformReceive(timestamp, msgSignature, nonce, encryptType, data)

	agent, err := GetAgent(appID)
	if err != nil {
		log.Println("error get agent when message pushed: ", err)

		if err == e_not_exist {
			return "", []byte(""), nil
		}

		return "text/plain", nil, err
	}

	if agent.Type == util.WECHAT_APP_OPEN {
		msg, err := message.Receive(decrypted)
		if err != nil {
			log.Println("error receive wechat open push: ", err)
			return "", nil, err
		}
		switch msg.(type) {
		case *message.TextMessage:
			txt := msg.(*message.TextMessage)
			if txt.Content == "TESTCOMPONENT_MSG_TYPE_TEXT" {
				log.Printf("wechat platform test 1: %+v", *txt)
				reply := new(message.TextMessage)
				reply.MsgType = txt.MsgType
				reply.FromUserName = txt.ToUserName
				reply.CreateTime = int(time.Now().Unix())
				reply.ToUserName = txt.FromUserName
				reply.Content = "TESTCOMPONENT_MSG_TYPE_TEXT_callback"

				replyData, err := message.PlatformReplyOpen(reply)
				if err != nil {
					log.Println("error wechat platform test1: ", err)
				}

				return "application/xml", replyData, nil
			} else if strings.HasPrefix(txt.Content, "QUERY_AUTH_CODE:") {
				log.Printf("wechat platform test 2: %+v", *txt)
				go func() {
					authCode := strings.Split(txt.Content, ":")[1]

					reply := new(message.ContactText)
					reply.MsgType = message.CONTACT_MSG_TXT
					reply.ToUser = txt.FromUserName
					reply.Text.Content = authCode + "_from_api"

					time.Sleep(5 * time.Second)

					accessToken, err := GetAgentAccessToken(appID)
					if err != nil {
						log.Println("error wechat platform test2: ", err)
						return
					}

					if err := message.SendContactMessage(reply, accessToken); err != nil {
						log.Println("error wechat platform test 2: ", err)
					}
				}()

				return "text/plain", []byte(""), nil
			} else {
				return "text/plain", []byte(""), nil
			}
		default:
			return "text/plain", []byte(""), nil
		}
	} else if agent.Type == util.WECHAT_APP_MINIAPP {
		return "text/plain", []byte(""), nil
	}

	return "", nil, nil
}
