package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"obsessiontech/wechat/util"
)

const e_invalid_access_token = 40001

func PlatformPushOpenTemplate(accessToken, templateID, openID, first, remark, url, miniappAppID, miniappPage string, keywords ...string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", accessToken), bytes.NewReader(packContent(templateID, openID, first, remark, url, miniappAppID, miniappPage, keywords...)))
	if err != nil {
		log.Println("error push wechat template: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return err
	}
	var ret PushRet
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return err
	}
	if ret.ErrCode != 0 {
		log.Println("error push wechat template: ", ret.ErrCode, ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PushTemplate(templateID, openID, first, remark, url string, keywords ...string) {
	var count = 0
try:
	count++
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", util.GetOpenAccessToken()), bytes.NewReader(packContent(templateID, openID, first, remark, url, "", "", keywords...)))
	if err != nil {
		log.Println("error push wechat template: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return
	}
	var ret PushRet
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Println("error push wechat template: ", err)
		return
	}
	if ret.ErrCode != 0 {
		log.Println("error push wechat template: ", ret.ErrCode, ret.ErrMsg)

		if ret.ErrCode == e_invalid_access_token && count < 3 {
			util.ExpireOpenAccessToken()
			goto try
		}

		return
	}
}

type PushRet struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	MsgID   int    `json:"msgid"`
}

func packContent(templateID, openID, first, remark, url, miniappAppID, miniappPage string, keywords ...string) []byte {
	content := make(map[string]interface{})

	content["touser"] = openID
	content["template_id"] = templateID

	if url != "" {
		content["url"] = url
	}

	if miniappAppID != "" {
		miniapp := make(map[string]interface{})
		miniapp["appid"] = miniappAppID
		if miniappPage != "" {
			miniapp["pagepath"] = miniappPage
		}
		content["miniprogram"] = miniapp
	}

	data := make(map[string]map[string]string)

	data["first"] = map[string]string{"value": first}
	data["remark"] = map[string]string{"value": remark}
	for i, keyword := range keywords {
		data[fmt.Sprintf("keyword%d", i+1)] = map[string]string{"value": keyword}
	}

	content["data"] = data

	result, err := json.Marshal(content)
	if err != nil {
		log.Panic("error parse content: ", err)
	}

	log.Printf("template content parsed: %v", string(result))

	return result
}
