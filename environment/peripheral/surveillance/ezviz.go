package surveillance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"obsessiontech/common/util"
)

var e_ezviz_config_invalid = errors.New("萤石云设置不正确")

type EzvizConfig struct {
	AppKey      string `json:"appKey"`
	AppSecret   string `json:"appSecret"`
	AccessToken string `json:"accessToken"`
	ExpireTsMs  int64  `json:"expireTsMs"`
}

type baseRet struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type accessTokenResult struct {
	AccessToken string `json:"accessToken"`
	ExpireTsMs  int64  `json:"expireTime"`
}

func accessToken(siteID string) (string, error) {

	surveillanceModule, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	if surveillanceModule.Ezviz == nil {
		return "", e_ezviz_config_invalid
	}

	if surveillanceModule.Ezviz.AccessToken != "" && time.Unix(surveillanceModule.Ezviz.ExpireTsMs/1000, surveillanceModule.Ezviz.ExpireTsMs%1000).After(time.Now()) {
		return surveillanceModule.Ezviz.AccessToken, nil
	}

	result, err := GetAccessToken(siteID, surveillanceModule.Ezviz.AppKey, surveillanceModule.Ezviz.AppSecret)
	if err != nil {
		return "", err
	}

	surveillanceModule.Ezviz.AccessToken = result.AccessToken
	surveillanceModule.Ezviz.ExpireTsMs = result.ExpireTsMs

	surveillanceModule.Save(siteID)

	return result.AccessToken, nil
}

func GetAccessToken(siteID, appKey, appSecret string) (*accessTokenResult, error) {

	req := url.Values{}
	req.Set("appKey", appKey)
	req.Set("appSecret", appSecret)

	resp, err := http.Post("https://open.ys7.com/api/lapp/token/get", "application/x-www-form-urlencoded", bytes.NewReader([]byte(req.Encode())))
	if err != nil {
		log.Println("error get ezviz access token:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get ezviz access token:", err)
		return nil, err
	}

	var ret struct {
		baseRet
		Data *accessTokenResult `json:"data"`
	}

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return nil, errors.New(ret.Msg)
	}

	ts := time.Unix(ret.Data.ExpireTsMs/100, ret.Data.ExpireTsMs%100)
	log.Println("get ezviz access token: ", ret.Data.AccessToken, ret.Data.ExpireTsMs, ts.Sub(time.Now()).Hours())

	return ret.Data, nil
}

type liveURLResult struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	ExpireTime util.Time `json:"expireTime"`
}

func GetLiveURL(siteID, deviceSerial, code string, channelNo int) (string, string, error) {

	token, err := accessToken(siteID)
	if err != nil {
		return "", "", err
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("deviceSerial", deviceSerial)

	if code != "" {
		req.Set("code", code)
	}

	if channelNo > 0 {
		req.Set("channelNo", fmt.Sprintf("%d", channelNo))
	}

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/v2/live/address/get", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get ezviz live url:", err)
		return "", "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get ezviz live url:", err)
		return "", "", err
	}

	var ret struct {
		baseRet
		Data *liveURLResult `json:"data"`
	}

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return "", "", errors.New(ret.Msg)
	}

	log.Println("get ezviz live url: ", ret.Data.ID, ret.Data.URL, time.Time(ret.Data.ExpireTime).Sub(time.Now()).Hours())

	return ret.Data.URL, token, nil
}

type deviceListResultItem struct {
	DeviceSerial  string `json:"deviceSerial"`
	DeviceName    string `json:"deviceName"`
	DeviceType    string `json:"deviceType"`
	Status        int    `json:"status"`
	Defence       int    `json:"defence"`
	DeviceVersion string `json:"deviceVersion"`
}

func GetDeviceList(siteID string, pageNo, pageSize int) ([]*deviceListResultItem, int, error) {
	token, err := accessToken(siteID)
	if err != nil {
		return nil, 0, err
	}

	if pageNo <= 0 {
		pageNo = 1
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("pageStart", fmt.Sprintf("%d", pageNo-1))
	req.Set("pageSize", fmt.Sprintf("%d", pageSize))

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/device/list", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get ezviz device list:", err)
		return nil, 0, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get ezviz device list:", err)
		return nil, 0, err
	}

	var ret struct {
		baseRet
		Data []*deviceListResultItem `json:"data"`
		Page struct {
			Total int `json:"total"`
			Page  int `json:"page"`
			Size  int `json:"size"`
		} `json:"page"`
	}

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return nil, 0, errors.New(ret.Msg)
	}

	return ret.Data, ret.Page.Total, nil
}

type deviceCameraListResultItem struct {
	DeviceSerial string `json:"deviceSerial"`
	DeviceName   string `json:"deviceName"`
	IPCSerial    string `json:"ipcSerial"`
	ChannelNo    int    `json:"channelNo"`
	ChannelName  string `json:"channelName"`
	Status       int    `json:"status"`
	IsShared     string `json:"isShared"`
	PicURL       string `json:"picUrl"`
	IsEncrypt    int    `json:"isEncrypt"`
	VideoLevel   int    `json:"videoLevel"`
	RelatedIPC   bool   `json:"relatedIpc"`
}

func GetDeviceCameraList(siteID, deviceSerial string) ([]*deviceCameraListResultItem, error) {

	token, err := accessToken(siteID)
	if err != nil {
		return nil, err
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("deviceSerial", deviceSerial)

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/device/camera/list", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error get ezviz device camera list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get ezviz device camera list:", err)
		return nil, err
	}

	var ret struct {
		baseRet
		Data []*deviceCameraListResultItem `json:"data"`
	}

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return nil, errors.New(ret.Msg)
	}

	return ret.Data, nil
}

func AddDevice(siteID, deviceSerial, validateCode string) error {

	token, err := accessToken(siteID)
	if err != nil {
		return err
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("deviceSerial", deviceSerial)
	req.Set("validateCode", validateCode)

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/device/add", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error ezviz add device:", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error ezviz add device:", err)
		return err
	}

	var ret baseRet

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return errors.New(ret.Msg)
	}

	return nil
}

func DeleteDevice(siteID, deviceSerial string) error {
	token, err := accessToken(siteID)
	if err != nil {
		return err
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("deviceSerial", deviceSerial)

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/device/delete", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error ezviz delete device:", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error ezviz delete device:", err)
		return err
	}

	var ret baseRet

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return errors.New(ret.Msg)
	}

	return nil
}

func UpdateDeviceName(siteID, deviceSerial, deviceName string) error {
	token, err := accessToken(siteID)
	if err != nil {
		return err
	}

	req := url.Values{}
	req.Set("accessToken", token)
	req.Set("deviceSerial", deviceSerial)
	req.Set("deviceName", deviceName)

	reqBytes := []byte(req.Encode())

	resp, err := http.Post("https://open.ys7.com/api/lapp/device/name/update", "application/x-www-form-urlencoded", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error ezviz update device name:", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error ezviz update device name:", err)
		return err
	}

	var ret baseRet

	json.Unmarshal(body, &ret)

	if ret.Code != "200" {
		return errors.New(ret.Msg)
	}

	return nil
}
