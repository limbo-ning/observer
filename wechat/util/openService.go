package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type OpenServiceAccountRet struct {
	ErrorRet
	OpenAppID string `json:"open_appid"`
}

func PlatformCreateOpenServiceAccount(accessToken, appID string) (string, error) {

	var ret OpenServiceAccountRet

	req := make(map[string]interface{})
	req["appid"] = appID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/open/create?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error create open service account:", err)
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error create open service account:", err)
		return "", err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return "", errors.New(ret.ErrMsg)
	}

	return ret.OpenAppID, nil
}

func PlatformGetOpenServiceAccount(accessToken, appID string) (string, error) {
	var ret OpenServiceAccountRet

	req := make(map[string]interface{})
	req["appid"] = appID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/open/get?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get open service account:", err)
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get open service account:", err)
		return "", err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return "", errors.New(ret.ErrMsg)
	}

	return ret.OpenAppID, nil
}

func PlatformBindOpenServiceAccount(accessToken, appID, openAppID string) error {
	var ret ErrorRet

	req := make(map[string]interface{})
	req["appid"] = appID
	req["open_appid"] = openAppID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/open/bind?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error bind open service account:", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error bind open service account:", err)
		return err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PlatformUnbindOpenServiceAccount(accessToken, appID, openAppID string) error {
	var ret ErrorRet

	req := make(map[string]interface{})
	req["appid"] = appID
	req["open_appid"] = openAppID

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/open/unbind?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error unbind open service account:", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error unbind open service account:", err)
		return err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return errors.New(ret.ErrMsg)
	}

	return nil
}
