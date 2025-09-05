package alipay

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"
)

const (
	Grant_type_authorization = "authorization_code"
	Grant_type_refresh       = "refresh_token"
)

func CreateAuthorizationLink(redirectURL string) string {
	return fmt.Sprintf("https://openauth.alipay.com/oauth2/appToAppBatchAuth.htm?app_id=%s&application_type=WEBAPP,MOBILEAPP,TINYAPP,PUBLICAPP&redirect_uri=%s", Config.AlipayPlatformAppID, redirectURL)
}

type AppAuthResponse struct {
	Code    string          `json:"code"`
	Msg     string          `json:"msg"`
	SubCode string          `json:"sub_code"`
	SubMsg  string          `json:"sub_msg"`
	Tokens  []*AppAuthToken `json:"tokens"`
}

type AppAuthToken struct {
	AppAuthToken    string        `json:"app_auth_token"`
	UserID          string        `json:"user_id"`
	AuthAppID       string        `json:"auth_app_id"`
	ExpiresIn       time.Duration `json:"expires_in"`
	ReExpiresIn     time.Duration `json:"re_expires_in"`
	AppRefreshToken string        `json:"app_refresh_token"`
}

type AppAuthRet struct {
	Response *AppAuthResponse `json:"alipay_open_auth_token_app_response"`
	Sign     string           `json:"sign"`
}

func GetAppAuth(grantType, code string) ([]*AppAuthToken, error) {
	param := getPublicParam(Config.AlipayPlatformAppID, "alipay.open.auth.token.app", "", "", "")

	bizContent := make(map[string]interface{})
	bizContent["grant_type"] = grantType
	switch grantType {
	case Grant_type_authorization:
		bizContent["code"] = code
	case Grant_type_refresh:
		bizContent["refresh_token"] = code
	default:
		return nil, errors.New("invalid grant type")
	}

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, Config.AlipayPlatformPrivateKey)

	form := url.Values{}
	for k, v := range param {
		form.Set(k, v)
	}

	body, err := execute(param)

	var ret AppAuthRet
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Println("error request alipay authorization: ", err)
		return nil, err
	}

	if ret.Response.Code != success_code {
		return nil, errors.New(ret.Response.Msg)
	}

	return ret.Response.Tokens, nil
}
