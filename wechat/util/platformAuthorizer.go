package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type AuthorizerAccessCodeRet struct {
	ErrCode           int    `json:"errcode"`
	ErrMsg            string `json:"errmsg"`
	AuthorizationInfo struct {
		AuthorizationAppID       string                   `json:"authorizer_appid"`
		AuthorizationAccessToken string                   `json:"authorizer_access_token"`
		ExpiresIn                time.Duration            `json:"expires_in"`
		AuthorizerRrefreshToken  string                   `json:"authorizer_refresh_token"`
		FuncInfo                 []map[string]interface{} `json:"func_info"`
	} `json:"authorization_info"`
}

func GetAuthorizerAccessCode(authorizationCode string) (*AuthorizerAccessCodeRet, error) {

	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var ret AuthorizerAccessCodeRet

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID
	req["authorization_code"] = authorizationCode

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_query_auth?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get authorizer access code:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get authorizer access code:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

type RefreshAuthorizerAccessCodeRet struct {
	ErrCode                  int           `json:"errcode"`
	ErrMsg                   string        `json:"errmsg"`
	AuthorizationAccessToken string        `json:"authorizer_access_token"`
	AuthorizerRrefreshToken  string        `json:"authorizer_refresh_token"`
	ExpiresIn                time.Duration `json:"expires_in"`
}

func RefreshAuthorizerAccessCode(authorizerAppID, refreshAuthorizerAccessCode string) (*RefreshAuthorizerAccessCodeRet, error) {
	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var ret RefreshAuthorizerAccessCodeRet

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID
	req["authorizer_appid"] = authorizerAppID
	req["authorizer_refresh_token"] = refreshAuthorizerAccessCode

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_authorizer_token?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error refresh authorizer access code:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error refresh authorizer access code:", err)
		return nil, err
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}

	if ret.ErrCode > 0 {
		log.Println("error refresh authorizer access code: ", ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

type AuthorizerListRet struct {
	ErrCode    int           `json:"errcode"`
	ErrMsg     string        `json:"errmsg"`
	TotalCount int           `json:"total_count"`
	List       []*Authorizer `json:"list"`
}

type Authorizer struct {
	AuthorizerAppID string `json:"authorizer_appid"`
	RefreshToken    string `json:"refresh_token"`
	AuthTime        int    `json:"auth_time"`
}

func GetPlatformAuthorizerList(offset, count int) (*AuthorizerListRet, error) {
	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var ret AuthorizerListRet

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID
	req["offset"] = offset
	req["count"] = count

	reqBytes, _ := json.Marshal(req)

	log.Println("wechat platform get authorizer list: ", offset, count)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_list?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get authorizer list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get authorizer list:", err)
		return nil, err
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}

	if ret.ErrCode > 0 {
		log.Println("error get authorizer list: ", ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}
