package wechat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type WechatAccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

func GetAccessToken(code string) WechatAccessToken {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", Config.WechatAppID, Config.WechatAppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var accessToken WechatAccessToken
	json.Unmarshal(body, &accessToken)

	log.Println("wechat access token ", string(body), accessToken)

	return accessToken
}

type UserInfo struct {
	Openid     string `json:"open_id"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	Headimgurl string `json:"headimgurl"`
	Unionid    string `json:"unionid"`
}

func GetUserInfo(token *WechatAccessToken, userInfo *UserInfo) bool {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s", token.AccessToken, token.OpenID)

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
		return false
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
		return false
	}

	json.Unmarshal(body, &userInfo)

	log.Println("user info:", string(body), userInfo)

	return true
}
