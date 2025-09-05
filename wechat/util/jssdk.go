package util

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"obsessiontech/common/random"
)

type WxConfig struct {
	AppID     string `json:"appID"`
	Timestamp int64  `json:"timestamp"`
	Noncestr  string `json:"noncestr"`
	Signature string `json:"signature"`
}

var JS_API_TICKET string

type JsApiTicketRet struct {
	ErrCode   int           `json:"errcode"`
	ErrMsg    string        `json:"errmsg"`
	Ticket    string        `json:"ticket"`
	ExpiresIn time.Duration `json:"expires_in"`
}

func GetJsApiTicket() string {
	if JS_API_TICKET == "" {
		url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", GetOpenAccessToken())
		resp, err := http.Get(url)
		if err != nil {
			log.Panic(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panic(err)
		}

		var ret JsApiTicketRet
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to get js api ticket. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		}

		JS_API_TICKET = ret.Ticket

		refreshTimeout := (time.Second * ret.ExpiresIn).Seconds() * 0.9

		time.AfterFunc(time.Duration(refreshTimeout), func() {
			JS_API_TICKET = ""
		})

	}

	return JS_API_TICKET
}

func GetWxConfig(url string) *WxConfig {
	param := make(map[string]interface{})
	param["url"] = url
	param["jsapi_ticket"] = GetJsApiTicket()
	param["noncestr"] = random.GenerateNonce(16)
	param["timestamp"] = time.Now().Unix()
	param["signature"] = Sign(param)

	return &WxConfig{
		Noncestr:  param["noncestr"].(string),
		Timestamp: param["timestamp"].(int64),
		Signature: param["signature"].(string),
		AppID:     Config.WechatAppID,
	}
}

func Sign(params map[string]interface{}) string {
	pairs := make([]string, 0)

	for k, v := range params {
		if vs, ok := v.(string); ok {
			if vs == "" {
				continue
			}
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(pairs)

	var toSign = strings.Join(pairs, "&")

	log.Println("js api content to sign", toSign)

	t := sha1.New()
	t.Write([]byte(toSign))

	signed := fmt.Sprintf("%x", t.Sum(nil))
	signed = strings.ToLower(signed)

	log.Println("js api signed", signed)

	return signed
}
