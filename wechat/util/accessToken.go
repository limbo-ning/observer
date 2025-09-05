package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

type UserAccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

func GetUserCodeSilentRedirectURL(redirectURL string) string {
	url := fmt.Sprintf("https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_base#wechat_redirect", Config.WechatAppID, url.QueryEscape(redirectURL))
	return url
}

func GetUserCodeRedirectURL(redirectURL string) string {
	url := fmt.Sprintf("https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo#wechat_redirect", Config.WechatAppID, url.QueryEscape(redirectURL))
	return url
}
func PlatformGetUserCodeSilentRedirectURL(appID, redirectURL string) string {
	return fmt.Sprintf("https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_base&state=STATE&component_appid=%s#wechat_redirect", appID, url.QueryEscape(redirectURL), Config.WechatPlatformAppID)
}
func PlatformGetUserCodeRedirectURL(appID, redirectURL string) string {
	log.Println("generate platform redirect url: ", appID, redirectURL)
	return fmt.Sprintf("https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=STATE&component_appid=%s#wechat_redirect", appID, url.QueryEscape(redirectURL), Config.WechatPlatformAppID)
}
func GetOpenUserAccessToken(appID, secret, code string) (*UserAccessToken, error) {
	var accessToken UserAccessToken

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", appID, secret, code)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error open get user access token:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error open get user access token:", err)
		return nil, err
	}

	json.Unmarshal(body, &accessToken)

	return &accessToken, nil
}

func GetUserAccessToken(code string) (*UserAccessToken, error) {

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", Config.WechatAppID, Config.WechatAppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get user access token:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get user access token:", err)
		return nil, err
	}

	var accessToken UserAccessToken
	json.Unmarshal(body, &accessToken)

	return &accessToken, nil
}

func PlatformGetUserAccessToken(appID, code string) (*UserAccessToken, error) {

	componentAccessToken, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var accessToken UserAccessToken

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/component/access_token?appid=%s&code=%s&grant_type=authorization_code&component_appid=%s&component_access_token=%s", appID, code, Config.WechatPlatformAppID, componentAccessToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error platform get user access token:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error platform get user access token:", err)
		return nil, err
	}

	json.Unmarshal(body, &accessToken)

	return &accessToken, nil
}

func RefreshUserAccessToken(refreshToken string) (*UserAccessToken, error) {

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/refresh_token?appid=%s&grant_type=refresh_token&refresh_token=%s", Config.WechatAppID, refreshToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error refresh user access token:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error refresh user access token:", err)
		return nil, err
	}

	var accessToken UserAccessToken
	json.Unmarshal(body, &accessToken)

	return &accessToken, nil
}

var openAccessToken string

type ServerAccessToken struct {
	ErrCode     int           `json:"errcode"`
	ErrMsg      string        `json:"errmsg"`
	AccessToken string        `json:"access_token"`
	ExpiresIn   time.Duration `json:"expires_in"`
}

func ExpireOpenAccessToken() {
	openAccessToken = ""
}

func GetOpenAccessToken() string {
	if openAccessToken == "" {

		url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", Config.WechatAppID, Config.WechatAppSecret)
		resp, err := http.Get(url)
		if err != nil {
			log.Panic(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panic(err)
		}

		var ret ServerAccessToken
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to get open access token. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		}

		openAccessToken = ret.AccessToken

		refreshTimeout := float64(time.Second*ret.ExpiresIn) * 0.9

		time.AfterFunc(time.Duration(refreshTimeout), func() {
			openAccessToken = ""
		})

		log.Println("request access token: ", ret)
	}

	return openAccessToken
}

var miniAppAccessToken string

func ExpireMiniAppAccessToken() {
	miniAppAccessToken = ""
}

func GetMiniAppAccessToken() string {
	if miniAppAccessToken == "" {

		url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", Config.WechatMiniAppID, Config.WechatMiniAppSecret)
		resp, err := http.Get(url)
		if err != nil {
			log.Panic(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panic(err)
		}

		var ret ServerAccessToken
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to get open access token. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		}

		miniAppAccessToken = ret.AccessToken

		refreshTimeout := float64(time.Second*ret.ExpiresIn) * 0.9

		time.AfterFunc(time.Duration(refreshTimeout), func() {
			miniAppAccessToken = ""
		})

	}

	return miniAppAccessToken
}

type UserSessionKey struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode,omitempty"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

func GetUserSessionKey(code string) (*UserSessionKey, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", Config.WechatMiniAppID, Config.WechatMiniAppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var sessionKey UserSessionKey
	json.Unmarshal(body, &sessionKey)

	log.Println("miniapp session key ", string(body), sessionKey)

	if sessionKey.ErrCode > 0 {
		return nil, errors.New(sessionKey.ErrMsg)
	}

	return &sessionKey, nil
}

func PlatformGetUserSessionKey(appID, code string) (*UserSessionKey, error) {
	componentAccessToken, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/component/jscode2session?appid=%s&js_code=%s&grant_type=authorization_code&component_appid=%s&component_access_token=%s", appID, code, Config.WechatPlatformAppID, componentAccessToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var sessionKey UserSessionKey
	json.Unmarshal(body, &sessionKey)

	log.Println("miniapp session key ", string(body), sessionKey)

	if sessionKey.ErrCode > 0 {
		log.Println("error get miniapp session key: ", sessionKey.ErrMsg)
		return nil, errors.New(sessionKey.ErrMsg)
	}

	return &sessionKey, nil
}
