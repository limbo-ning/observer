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

func PlatformPushMiniAppSubscription(accessToken, templateID, openID, page string, data map[string]interface{}) error {

	toSend := make(map[string]interface{})

	toSend["touser"] = openID
	toSend["template_id"] = templateID
	toSend["page"] = page
	toSend["data"] = data

	dataToSend, err := json.Marshal(toSend)
	if err != nil {
		log.Println("error marshal platform push mini app subscription")
		return err
	}

	log.Println("platform push miniapp subscription: ", string(dataToSend))

	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", accessToken), bytes.NewReader(dataToSend))
	if err != nil {
		log.Println("error push wechat miniapp subscription: ", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error push wechat miniapp subscription: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error push wechat miniapp subscription: ", err)
	}
	var ret PushRet
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Println("error push wechat miniapp subscription: ", err)
		return err
	}
	if ret.ErrCode != 0 {
		log.Println("error push wechat miniapp subscription: ", ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PushMiniAppTemplate(templateID, openID, formID, page string, keywords ...string) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/wxopen/template/send?access_token=%s", util.GetMiniAppAccessToken()), bytes.NewReader(packMiniAppContent(templateID, openID, formID, page, keywords...)))
	if err != nil {
		log.Println("error push wechat miniapp template: ", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error push wechat miniapp template: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error push wechat miniapp template: ", err)
	}
	var ret PushRet
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Println("error push wechat miniapp template: ", err)
	}
	if ret.ErrCode != 0 {
		log.Println("error push wechat miniapp template: ", ret.ErrMsg)
	}
}

func packMiniAppContent(templateID, openID, formID, page string, keywords ...string) []byte {
	content := make(map[string]interface{})

	content["touser"] = openID
	content["template_id"] = templateID
	content["form_id"] = formID

	if page != "" {
		content["page"] = page
	}

	data := make(map[string]map[string]string)

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
