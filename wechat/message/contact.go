package message

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	commonUtil "obsessiontech/common/util"
)

type IContactMessage interface {
	GetToUser() string
	GetMsgType() string
}
type BaseContactMessage struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
}

func (b *BaseContactMessage) GetToUser() string {
	return b.ToUser
}
func (b *BaseContactMessage) GetMsgType() string {
	return b.MsgType
}

const CONTACT_MSG_TXT = "text"
const CONTACT_MSG_IMG = "image"
const CONTACT_MSG_LINK = "link"
const CONTACT_MSG_PG = "miniprogrampage"

type ContactText struct {
	BaseContactMessage
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
}

type ContactImage struct {
	BaseContactMessage
	Image struct {
		MediaID string `json:"media_id"`
	} `json:"image"`
}

type ContactLink struct {
	BaseContactMessage
	Link struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		ThumbURL    string `json:"thumb_url"`
	} `json:"link"`
}

type MiniAppContactProgramPage struct {
	BaseContactMessage
	ProgramPage struct {
		Title        string `json:"title"`
		PagePath     string `json:"pagepath"`
		ThumbMediaID string `json:"thumb_media_id"`
	} `json:"miniprogrampage"`
}

func SendContactMessage(msg IContactMessage, accessToken string) error {
	data, err := commonUtil.UnsafeJsonString(msg)

	if err != nil {
		log.Println("error pack contact message: ", err)
		return err
	}

	log.Println("push contact message data: ", string(data))

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", accessToken), bytes.NewReader(data))
	if err != nil {
		log.Println("error push wechat miniapp contact msg: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("error push wechat miniapp contact msg: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error push wechat miniapp contact msg: ", err)
		return err
	}
	log.Println("contact push ret: ", string(body))

	return nil
}
