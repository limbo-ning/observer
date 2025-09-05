package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

const WECHAT_APP_OPEN = "OPEN"
const WECHAT_APP_WEB = "WEB"
const WECHAT_APP_MINIAPP = "MINIAPP"

type AuthorizationInfo struct {
	AuthorizationAppID string                   `json:"authorization_appid"`
	FuncInfo           []map[string]interface{} `json:"func_info"`
}

type OpenInfo struct {
	ErrCode           int                 `json:"errcode,omitempty"`
	ErrMsg            string              `json:"errmsg,omitempty"`
	AuthorizorInfo    *OpenAuthorizerInfo `json:"authorizer_info"`
	AuthorizationInfo *AuthorizationInfo  `json:"authorization_info"`
}

type OpenAuthorizerInfo struct {
	NickName        string                 `json:"nick_name"`
	HeadImg         string                 `json:"head_img"`
	ServiceTypeInfo map[string]interface{} `json:"service_type_info"`
	VerifyTypeInfo  map[string]interface{} `json:"verify_type_info"`
	Uername         string                 `json:"user_name"`
	PrincipalName   string                 `json:"principal_name"`
	Businessinfo    map[string]interface{} `json:"business_info"`
	Alias           string                 `json:"alias"`
	QrCodeURL       string                 `json:"qrcode_url"`
}

type MiniAppInfo struct {
	ErrCode           int                 `json:"errcode,omitempty"`
	ErrMsg            string              `json:"errmsg,omitempty"`
	AuthorizorInfo    *MiniAuthorizerInfo `json:"authorizer_info"`
	AuthorizationInfo *AuthorizationInfo  `json:"authorization_info"`
}

type MiniAuthorizerInfo struct {
	NickName        string                 `json:"nick_name"`
	HeadImg         string                 `json:"head_img"`
	ServiceTypeInfo map[string]interface{} `json:"service_type_info"`
	VerifyTypeInfo  map[string]interface{} `json:"verify_type_info"`
	Uername         string                 `json:"user_name"`
	PrincipalName   string                 `json:"principal_name"`
	Businessinfo    map[string]interface{} `json:"business_info"`
	Signature       string                 `json:"signature"`
	MiniProgramInfo map[string]interface{} `json:"MiniProgramInfo"`
}

func GetAuthorizerOpenInfo(appID string) (*OpenInfo, error) {

	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var ret OpenInfo

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID
	req["authorizer_appid"] = appID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_info?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get authorizer open info:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get authorizer open info:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

func GetAuthorizerMiniAppInfo(appID string) (*MiniAppInfo, error) {
	accessTicket, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	var ret MiniAppInfo

	req := make(map[string]interface{})
	req["component_appid"] = Config.WechatPlatformAppID
	req["authorizer_appid"] = appID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_info?component_access_token="+accessTicket, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get authorizer mini app info:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get authorizer mini app info:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}
