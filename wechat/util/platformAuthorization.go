package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"obsessiontech/common/ipc"
	"os"
	"time"
)

var platformAesKey []byte

var componentVerifyTicket string
var componentAccessTicket string

func init() {
	var err error
	if Config.WechatPlatformEncryptKey != "" {
		platformAesKey, err = base64.StdEncoding.DecodeString(Config.WechatPlatformEncryptKey + "=")
		if err != nil {
			panic(err)
		}
	}

	if Config.WechatPlatformVerifyTicketPath != "" {
		if vt, err := ioutil.ReadFile(Config.WechatPlatformVerifyTicketPath); err != nil {
			log.Println("error read wechat platform verify ticket: ", err)
		} else {
			componentVerifyTicket = string(vt)
		}
	}

	if Config.IsWechatPlatformHost {
		connChan, err := ipc.StartHost(Config.WechatPlatformHostType, Config.WechatPlatformHost)
		if err != nil {
			log.Println("error establish wechat platform host: ", err)
			panic(err)
		}

		go func() {
			for {
				select {
				case conn, ok := <-connChan:
					if !ok {
						log.Println("wechat platform host closed")
						return
					}

					go listen(conn)
				}
			}
		}()

	}
}

const PLATFORM_VERIFY_TICKET = "component_verify_ticket"
const PLATFORM_AUTHORIZED = "authorized"
const PLATFORM_UNAUTHORIZED = "unauthorized"
const PLATFORM_UPDATEAUTHORIZED = "updateauthorized"

type AuthorizationPush struct {
	AppID                        string        `xml:"AppID"`
	CreateTime                   string        `xml:"CreateTime"`
	InfoType                     string        `xml:"InfoType"`
	AuthorizerAppid              string        `xml:"AuthorizerAppid"`
	AuthorizationCode            string        `xml:"AuthorizationCode"`
	AuthorizationCodeExpiredTime time.Duration `xml:"AuthorizationCodeExpiredTime"`
	PreAuthCode                  string        `xml:"PreAuthCode"`
	ComponentVerifyTicket        string        `xml:"ComponentVerifyTicket"`
}

type NotifyEncrypted struct {
	AppID   string `xml:"AppID"`
	Encrypt string `xml:"Encrypt"`
}

func ReceivePlatformAuthorizationPush(timestamp int, msgSignature, nonce, encryptType string, data []byte) (*AuthorizationPush, error) {
	if encryptType != "aes" {
		log.Println("error unsupported platform authorization push encrypt type: ", encryptType)
	}

	var encryted NotifyEncrypted

	if err := xml.Unmarshal(data, &encryted); err != nil {
		log.Println("error unmarshal platform authorization push: ", err)
		return nil, err
	}

	if selfSigned := PlatformSign(timestamp, nonce, encryted.Encrypt); selfSigned != msgSignature {
		log.Println("error signature not match: ", msgSignature, selfSigned)
	}

	decrypted, err := PlatformDecrypt(encryted.Encrypt)
	if err != nil {
		return nil, err
	}

	log.Println("decrypted: ", string(decrypted))

	var push AuthorizationPush
	if err := xml.Unmarshal(decrypted, &push); err != nil {
		log.Println("error unmarshal decrypted platform authorization push", err)
		return nil, err
	}

	if push.InfoType == PLATFORM_VERIFY_TICKET {
		componentVerifyTicket = push.ComponentVerifyTicket

		if Config.WechatPlatformVerifyTicketPath != "" {
			go func() {
				if err := ioutil.WriteFile(Config.WechatPlatformVerifyTicketPath, []byte(componentVerifyTicket), os.ModePerm); err != nil {
					log.Println("error save verify ticket: ", err)
				}
			}()
		}

	}

	return &push, nil
}

type ComponentAccessTokenRet struct {
	ErrCode              int           `json:"errcode"`
	ErrMsg               string        `json:"errmsg"`
	ComponentAccessToken string        `json:"component_access_token"`
	ExpiresIn            time.Duration `json:"expires_in"`
}

func GetComponentAccessToken() (string, error) {

	if !Config.IsWechatPlatformHost && Config.WechatPlatformHost != "" {

		send := make(chan IMessage)
		receive, err := client(Config.WechatPlatformHostType, Config.WechatPlatformHost, send)
		if err != nil {
			log.Println("wechat platform component accesstoken ipc err establishing: ", err)
			return "", err
		}

		defer close(send)

		log.Println("wechat platform component accesstoken ipc established")
		send <- new(ComponentAccessTokenReq)

		select {
		case res, ok := <-receive:
			log.Println("wechat platform component accesstoken ipc received: ", res, ok)
			if !ok {
				return "", errors.New("no ipc reply")
			} else {
				if token, ok := res.(*ComponentAccessTokenRes); ok && *token != "" {
					return string(*token), nil
				} else {
					return "", errors.New("invalid ipc reply")
				}
			}
		case <-time.After(Config.WechatPlatformTimeoutSec * time.Second):
			log.Println("wechat platform component accesstoken ipc timeout")
			return "", errors.New("ipc timeout")
		}
	}

	if componentAccessTicket == "" {

		if componentVerifyTicket == "" {
			return "", errors.New("系统没有可用的componentVerifyTicket")
		}

		var accessToken ComponentAccessTokenRet

		req := make(map[string]interface{})
		req["component_appid"] = Config.WechatPlatformAppID
		req["component_appsecret"] = Config.WechatPlatformAppSecret
		req["component_verify_ticket"] = componentVerifyTicket

		reqBytes, _ := json.Marshal(req)

		resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_component_token", "application/json", bytes.NewReader(reqBytes))
		if err != nil {
			log.Println("error get component access token:", err)
			return "", err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("error get component access token:", err)
			return "", err
		}

		json.Unmarshal(body, &accessToken)

		log.Println("wechat component access token ", string(body), accessToken)

		if accessToken.ErrCode > 0 {
			return "", errors.New(accessToken.ErrMsg)
		}

		componentAccessTicket = accessToken.ComponentAccessToken

		refreshTimeout := float64(time.Second*accessToken.ExpiresIn) * 0.9

		time.AfterFunc(time.Duration(refreshTimeout), func() {
			log.Println("expire component access token")
			componentAccessTicket = ""
		})
	}

	return componentAccessTicket, nil
}

type PreAuthCodeRet struct {
	ErrCode     int           `json:"errcode"`
	ErrMsg      string        `json:"errmsg"`
	PreAuthCode string        `json:"pre_auth_code"`
	ExpiresIn   time.Duration `json:"expires_in"`
}

func GetPreAuthCode() (string, error) {

	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return "", err
	}

	var ret PreAuthCodeRet

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_create_preauthcode?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get pre auth code:", err)
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get pre auth code:", err)
		return "", err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return "", errors.New(ret.ErrMsg)
	}

	return ret.PreAuthCode, nil

}

func CreateAuthorizationLink(redirectURL, appID string) (pcLink string, wechatLink string, err error) {
	preAuthCode, err := GetPreAuthCode()
	if err != nil {
		return "", "", err
	}

	var bizAppID string
	if appID != "" {
		bizAppID = "&biz_appid=" + appID
	}

	pcLink = fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s%s", Config.WechatPlatformAppID, preAuthCode, redirectURL, bizAppID)
	wechatLink = fmt.Sprintf("https://mp.weixin.qq.com/safe/bindcomponent?action=bindcomponent&auth_type=3&no_scan=1&component_appid=%s&pre_auth_code=%s&redirect_uri=%s%s#wechat_redirect", Config.WechatPlatformAppID, preAuthCode, redirectURL, bizAppID)

	return
}
