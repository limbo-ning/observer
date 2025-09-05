package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"obsessiontech/common/encrypt"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/data/recent"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/mission"
	"obsessiontech/environment/peripheral"
	"obsessiontech/environment/push"
	"obsessiontech/environment/user"
	"obsessiontech/environment/wechat"

	wechatUtil "obsessiontech/wechat/util"
)

/*
#cgo LDFLAGS: -ldl
*/
import "C"

//export getRecentDataValue
func getRecentDataValue(siteID *C.char, dataType *C.char, stationID int, monitorID int) float64 {

	goSiteID := C.GoString(siteID)
	goDataType := C.GoString(dataType)

	recentDatas, err := recent.GetRecentData(goSiteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, goDataType, stationID)
	if err != nil {
		log.Println("error get recent data: ", goSiteID, stationID, err)
		return 0
	}

	stationData := recentDatas[stationID]
	if stationData == nil {
		return 0
	}

	d := stationData[monitorID]
	if d == nil {
		return 0
	}

	if rtd, ok := d.(data.IRealTime); ok {
		return rtd.GetRtd()
	} else if interval, ok := d.(data.IInterval); ok {
		return interval.GetAvg()
	}

	return 0
}

//export getMonitorName
func getMonitorName(siteID *C.char, monitorID int) *C.char {

	cSiteID := C.GoString(siteID)

	m := monitor.GetMonitor(cSiteID, monitorID)
	if m == nil {
		log.Println("get monitor name nil")
		return C.CString("-")
	}

	// if m.Unit != "" {
	// 	return C.CString(fmt.Sprintf("%s-%s", m.Name, m.Unit))
	// }

	return C.CString(m.Name)
}

//export getMonitorFlagLimit
func getMonitorFlagLimit(siteID *C.char, stationID int, monitorID int, flag *C.char) *C.char {

	mfl := monitor.GetFlagLimit(C.GoString(siteID), stationID, monitorID, C.GoString(flag))

	if mfl == nil {
		return C.CString("-")
	}

	return C.CString(mfl.Region)
}

//export getMonitorFlagName
func getMonitorFlagName(siteID *C.char, flag *C.char) *C.char {

	goFlag := C.GoString(flag)

	f, err := monitor.GetFlag(C.GoString(siteID), goFlag)
	if err != nil {
		return C.CString(goFlag)
	}
	return C.CString(f.Name)
}

const (
	STATION_STATUS = "station_status"
	DATA_DAILY     = "data_" + data.DAILY
	DATA_HOURLY    = "data_" + data.HOURLY
	DATA_MINUTELY  = "data_" + data.MINUTELY
	DATA_REAL_TIME = "data_" + data.REAL_TIME
)

func init() {
	push.RegisterSubsciption(STATION_STATUS, func(sub *push.Subscription) push.IPush {
		p := new(StationSubscription)
		p.Subscription = *sub
		return p
	})
	push.RegisterSubsciption(DATA_DAILY, func(sub *push.Subscription) push.IPush {
		p := new(MonitorSubscription)
		p.Subscription = *sub
		return p
	})
	push.RegisterSubsciption(DATA_HOURLY, func(sub *push.Subscription) push.IPush {
		p := new(MonitorSubscription)
		p.Subscription = *sub
		return p
	})
	push.RegisterSubsciption(DATA_MINUTELY, func(sub *push.Subscription) push.IPush {
		p := new(MonitorSubscription)
		p.Subscription = *sub
		return p
	})
	push.RegisterSubsciption(DATA_REAL_TIME, func(sub *push.Subscription) push.IPush {
		p := new(MonitorSubscription)
		p.Subscription = *sub
		return p
	})
}

type StationSubscription struct {
	push.Subscription
	Entity  *entity.Entity
	Station *entity.Station
	Time    time.Time
	// Duration string

	IsCease bool
}

func (s *StationSubscription) GetSubscriptionType() string {
	return s.Type
}
func (s *StationSubscription) GetPushType() string {
	return s.Push
}
func (s *StationSubscription) ShouldPush(siteID string) error {
	if _, _, err := s.PushInterval.GetInterval(time.Now(), true); err != nil {
		return err
	}
	return nil
}

type MonitorSubscription struct {
	push.Subscription
	Entity   *entity.Entity
	Station  *entity.Station
	Time     time.Time
	DataList []data.IData

	IsCease bool
}

func (s *MonitorSubscription) GetSubscriptionType() string {
	return s.Type
}
func (s *MonitorSubscription) GetPushType() string {
	return s.Push
}
func (s *MonitorSubscription) ShouldPush(siteID string) error {
	if _, _, err := s.PushInterval.GetInterval(time.Now(), true); err != nil {
		return err
	}
	return nil
}

func getSMSName(name string) string {
	if utf8.RuneCountInString(name) < 20 {
		return name
	}

	name = strings.ReplaceAll(name, "有限责任公司", "")
	name = strings.ReplaceAll(name, "有限公司", "")
	name = strings.ReplaceAll(name, "公司", "")

	if utf8.RuneCountInString(name) < 20 {
		return name
	}

	return string([]rune(name)[:17]) + "..."
}

//Implement AliSmsPush interface
func (s *StationSubscription) GetMobile(siteID string) (string, error) {
	if s.SubscriberType == "user" {
		u, err := user.GetUser(siteID, "id", s.SubscriberID)
		if err != nil {
			log.Println("error push subscription: ", err, s)
			return "", err
		}
		if u.Mobile == "" {
			return "", errors.New("用户未设置手机号")
		}
		return u.Mobile, nil
	} else if s.SubscriberType == "entity" || s.SubscriberType == "station" {
		mobile, exists := s.Ext["mobile"]
		if !exists {
			return "", errors.New("未设置推送号码")
		}
		mobileS, ok := mobile.(string)
		if !ok {
			return "", errors.New("不正常的推送号码")
		}
		return mobileS, nil
	} else if s.SubscriberType == "role" {
		return "", push.E_subscriber_should_not_push_but_valid
	}

	return "", push.E_invalid_subsriber
}

func (s *StationSubscription) GetTemplateCode(siteID string) (string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.SMSTemplateID == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Cease.SMSTemplateID, nil
	} else {
		if pSetting.Trigger.SMSTemplateID == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Trigger.SMSTemplateID, nil
	}
}

func (s *StationSubscription) GetSignature(siteID string) (string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		log.Println("error push subscription: ", err, s)
		return "", err
	}
	if m.SMSSignature == "" {
		return "", errors.New("未设置短信签名")
	}
	return m.SMSSignature, nil
}

func (s *StationSubscription) GetSMSParam(siteID string) (map[string]string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return nil, errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getSMSParam(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true, s.Entity, s.Station, s.Time, nil)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getSMSParam(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false, s.Entity, s.Station, s.Time, nil)

	}
}

func (s *MonitorSubscription) GetMobile(siteID string) (string, error) {
	if s.SubscriberType == "user" {
		u, err := user.GetUser(siteID, "id", s.SubscriberID)
		if err != nil {
			log.Println("error push subscription: ", err, s)
			return "", err
		}
		if u.Mobile == "" {
			return "", errors.New("用户未设置手机号")
		}
		return u.Mobile, nil
	} else if s.SubscriberType == "entity" || s.SubscriberType == "station" {
		mobile, exists := s.Ext["mobile"]
		if !exists {
			return "", errors.New("未设置推送号码")
		}
		mobileS, ok := mobile.(string)
		if !ok {
			return "", errors.New("不正常的推送号码")
		}
		return mobileS, nil
	} else if s.SubscriberType == "role" {
		return "", push.E_subscriber_should_not_push_but_valid
	}

	return "", push.E_invalid_subsriber
}

func (s *MonitorSubscription) GetTemplateCode(siteID string) (string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.SMSTemplateID == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Cease.SMSTemplateID, nil
	} else {
		if pSetting.Trigger.SMSTemplateID == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Trigger.SMSTemplateID, nil
	}
}

func (s *MonitorSubscription) GetSignature(siteID string) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		log.Println("error push subscription: ", err, s)
		return "", err
	}
	if m.SMSSignature == "" {
		return "", errors.New("未设置短信签名")
	}
	return m.SMSSignature, nil
}

func (s *MonitorSubscription) GetSMSParam(siteID string) (map[string]string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return nil, errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getSMSParam(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true, s.Entity, s.Station, s.Time, s.DataList)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getSMSParam(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false, s.Entity, s.Station, s.Time, s.DataList)
	}
}

//Implement WxOpenTemplatePush interface
func (s *StationSubscription) GetPushParam(siteID string) ([]push.WxOpenTemplatePushParam, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	result := make([]push.WxOpenTemplatePushParam, 0)

	for _, setting := range m.WxSettings {
		param := push.WxOpenTemplatePushParam{}

		param.AccessToken, err = s.getAccessToken(siteID, setting)
		if err != nil {
			log.Println("error get access token: ", err)
			continue
		}

		param.OpenID, err = s.getOpenID(siteID, setting)
		if err != nil {
			log.Println("error get open id: ", err)
			continue
		}

		param.TemplateID, err = s.getTemplateID(siteID, setting)
		if err != nil {
			log.Println("error get template id: ", err)
			continue
		}
		param.First, err = s.getFirst(siteID, setting)
		if err != nil {
			log.Println("error get first: ", err)
			continue
		}
		param.Keywords, err = s.getKeywords(siteID, setting)
		if err != nil {
			log.Println("error get keywords: ", err)
			continue
		}
		param.Remark, err = s.getRemark(siteID, setting)
		if err != nil {
			log.Println("error get remark: ", err)
			continue
		}
		param.MiniappAppID, err = s.getMiniappAppID(siteID, setting)
		if err != nil {
			log.Println("error get miniapp id: ", err)
			continue
		}
		param.MiniappPage, err = s.getMiniappPage(siteID, setting)
		if err != nil {
			log.Println("error get miniapp page: ", err)
			continue
		}
		param.URL = ""

		result = append(result, param)
	}

	return result, nil
}

func (s *StationSubscription) getAccessToken(siteID string, setting *WxSetting) (string, error) {
	if setting.WxOpenAppID == "" {
		log.Println("error push subscription: openAppID not set")
		return "", errors.New("未配置appID")
	}
	accessToken, err := wechat.GetAgentAccessToken(setting.WxOpenAppID)
	if err != nil {
		log.Println("error get access token: ", err)
		return "", err
	}
	return accessToken, nil
}
func (s *StationSubscription) getOpenID(siteID string, setting *WxSetting) (string, error) {
	if wxUserInfo, exists := s.Ext["wxUserInfo"]; exists {
		wxUserInfoS, ok := wxUserInfo.(string)
		if !ok {
			return "", errors.New("不正常的微信用户信息")
		}
		infoData, err := encrypt.Base64Decrypt(wxUserInfoS)
		if err != nil {
			log.Println("error get openid from session: ", err)
			return "", err
		}
		userInfo := new(wechatUtil.UserInfo)
		if err := json.Unmarshal(infoData, userInfo); err != nil {
			log.Println("error get openid from session key: ", err)
			return "", err
		}
		return userInfo.Openid, nil
	}
	if s.SubscriberType == "user" {
		u, err := user.GetUser(siteID, "id", s.SubscriberID)
		if err != nil {
			log.Println("error push subscription: ", err, s)
			return "", err
		}
		if info, exists := u.WechatInfo[setting.WxOpenAppID]; exists {
			openid, exists := info.(map[string]interface{})["openid"]
			if !exists {
				openid, exists = info.(map[string]interface{})["openId"]
			}
			if exists {
				return openid.(string), nil
			}
		}
		log.Println("error get push openID: user has no wechatID: ", u.UserID)
	}
	return "", errors.New("未与微信账号绑定")
}
func (s *StationSubscription) getTemplateID(siteID string, setting *WxSetting) (string, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.WxOpenTemplateIDs[setting.WxOpenAppID] == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Cease.WxOpenTemplateIDs[setting.WxOpenAppID], nil
	} else {
		if pSetting.Trigger.WxOpenTemplateIDs[setting.WxOpenAppID] == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Trigger.WxOpenTemplateIDs[setting.WxOpenAppID], nil
	}
}
func (s *StationSubscription) getFirst(siteID string, setting *WxSetting) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateFirst(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateFirst(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false)

	}
	// return "您关注的监测点失联", nil
}
func (s *StationSubscription) getRemark(siteID string, setting *WxSetting) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateRemark(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateRemark(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false)

	}
}
func (s *StationSubscription) getURL(siteID string, setting *WxSetting) (string, error) {
	return "", nil
}
func (s *StationSubscription) getMiniappAppID(siteID string, setting *WxSetting) (string, error) {
	return setting.WxMiniappAppID, nil
}
func (s *StationSubscription) getMiniappPage(siteID string, setting *WxSetting) (string, error) {
	return setting.WxMiniappStationPage, nil
}
func (s *StationSubscription) getKeywords(siteID string, setting *WxSetting) ([]string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return nil, errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getWxOpenTemplateKeywords(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true, s.Entity, s.Station, s.Time, nil)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getWxOpenTemplateKeywords(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false, s.Entity, s.Station, s.Time, nil)
	}
}

func (s *MonitorSubscription) GetPushParam(siteID string) ([]push.WxOpenTemplatePushParam, error) {

	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	result := make([]push.WxOpenTemplatePushParam, 0)

	for _, setting := range m.WxSettings {
		param := push.WxOpenTemplatePushParam{}

		param.AccessToken, err = s.getAccessToken(siteID, setting)
		if err != nil {
			log.Println("error get access token: ", err)
			continue
		}

		param.OpenID, err = s.getOpenID(siteID, setting)
		if err != nil {
			log.Println("error get open id: ", err)
			continue
		}

		param.TemplateID, err = s.getTemplateID(siteID, setting)
		if err != nil {
			log.Println("error get template id: ", err)
			continue
		}
		param.First, err = s.getFirst(siteID, setting)
		if err != nil {
			log.Println("error get first: ", err)
			continue
		}
		param.Keywords, err = s.getKeywords(siteID, setting)
		if err != nil {
			log.Println("error get keywords: ", err)
			continue
		}
		param.Remark, err = s.getRemark(siteID, setting)
		if err != nil {
			log.Println("error get remark: ", err)
			continue
		}
		param.MiniappAppID, err = s.getMiniappAppID(siteID, setting)
		if err != nil {
			log.Println("error get miniapp id: ", err)
			continue
		}
		param.MiniappPage, err = s.getMiniappPage(siteID, setting)
		if err != nil {
			log.Println("error get miniapp page: ", err)
			continue
		}
		param.URL = ""

		result = append(result, param)
	}

	return result, nil
}
func (s *MonitorSubscription) getAccessToken(siteID string, setting *WxSetting) (string, error) {
	if setting.WxOpenAppID == "" {
		log.Println("error push subscription: openAppID not set")
		return "", errors.New("未配置appID")
	}
	accessToken, err := wechat.GetAgentAccessToken(setting.WxOpenAppID)
	if err != nil {
		log.Println("error get catering notice template setting: get access token: ", err)
		return "", err
	}
	return accessToken, nil
}
func (s *MonitorSubscription) getOpenID(siteID string, setting *WxSetting) (string, error) {
	if wxUserInfo, exists := s.Ext["wxUserInfo"]; exists {
		wxUserInfoS, ok := wxUserInfo.(string)
		if !ok {
			return "", errors.New("不正常的微信用户信息")
		}
		infoData, err := encrypt.Base64Decrypt(wxUserInfoS)
		if err != nil {
			log.Println("error get openid from session: ", err)
			return "", err
		}
		userInfo := new(wechatUtil.UserInfo)
		if err := json.Unmarshal(infoData, userInfo); err != nil {
			log.Println("error get openid from session key: ", err)
			return "", err
		}
		return userInfo.Openid, nil
	}
	if s.SubscriberType == "user" {
		u, err := user.GetUser(siteID, "id", s.SubscriberID)
		if err != nil {
			log.Println("error push subscription: ", err, s)
			return "", err
		}
		if info, exists := u.WechatInfo[setting.WxOpenAppID]; exists {
			openid, exists := info.(map[string]interface{})["openid"]
			if !exists {
				openid, exists = info.(map[string]interface{})["openId"]
			}
			if exists {
				return openid.(string), nil
			}
		}
		log.Println("error get push openID: user has no wechatID: ", u.UserID)
	}
	return "", errors.New("未与微信账号绑定")
}
func (s *MonitorSubscription) getTemplateID(siteID string, setting *WxSetting) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.WxOpenTemplateIDs[setting.WxOpenAppID] == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Cease.WxOpenTemplateIDs[setting.WxOpenAppID], nil
	} else {
		if pSetting.Trigger.WxOpenTemplateIDs[setting.WxOpenAppID] == "" {
			return "", errors.New("未设置该推送")
		}
		return pSetting.Trigger.WxOpenTemplateIDs[setting.WxOpenAppID], nil
	}
}
func (s *MonitorSubscription) getFirst(siteID string, setting *WxSetting) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateFirst(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateFirst(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false)

	}
}
func (s *MonitorSubscription) getRemark(siteID string, setting *WxSetting) (string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return "", err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return "", errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateRemark(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return "", errors.New("未设置该推送")
		}
		return getWxOpenTemplateRemark(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false)

	}
}
func (s *MonitorSubscription) getURL(siteID string, setting *WxSetting) (string, error) {
	return "", nil
}
func (s *MonitorSubscription) getMiniappAppID(siteID string, setting *WxSetting) (string, error) {
	return setting.WxMiniappAppID, nil
}
func (s *MonitorSubscription) getMiniappPage(siteID string, setting *WxSetting) (string, error) {
	return setting.WxMiniappStationPage, nil
}
func (s *MonitorSubscription) getKeywords(siteID string, setting *WxSetting) ([]string, error) {
	m, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	pSetting := m.PushSettings[s.GetSubscriptionType()]
	if pSetting == nil {
		return nil, errors.New("未设置该推送")
	}

	if s.IsCease {
		if pSetting.Cease.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getWxOpenTemplateKeywords(siteID, pSetting.Cease.LDLibPath, s.GetSubscriptionType(), true, s.Entity, s.Station, s.Time, s.DataList)
	} else {
		if pSetting.Trigger.LDLibPath == "" {
			return nil, errors.New("未设置该推送")
		}
		return getWxOpenTemplateKeywords(siteID, pSetting.Trigger.LDLibPath, s.GetSubscriptionType(), false, s.Entity, s.Station, s.Time, s.DataList)
	}
}

//speaker interface
func (s *MonitorSubscription) GetDeviceIDs(siteID string) ([]int, error) {

	if s.SubscriberType == "entity" {

		stations, err := entity.GetStations(siteID, nil, []int{s.SubscriberID}, entity.ACTIVE, "", "")
		if err != nil {
			return nil, err
		}

		stationIDs := make([]string, 0)
		for _, s := range stations {
			stationIDs = append(stationIDs, fmt.Sprintf("%d", s.ID))
		}

		deviceList, _, err := peripheral.GetDevices(siteID, authority.ActionAuthSet{{Action: peripheral.ACTION_ADMIN_VIEW}}, "", peripheral.DEVICE_SPEAKER, "", 0, -1, "", "environment_entity#station", stationIDs...)
		if err != nil {
			return nil, err
		}

		result := make([]int, 0)
		for _, d := range deviceList {
			result = append(result, d.ID)
		}

		return result, nil

	} else if s.SubscriberType == "station" {
		deviceList, _, err := peripheral.GetDevices(siteID, authority.ActionAuthSet{{Action: peripheral.ACTION_ADMIN_VIEW}}, "", peripheral.DEVICE_SPEAKER, "", 0, -1, "", "environment_entity#station", fmt.Sprintf("%s", s.SubscriberID))
		if err != nil {
			return nil, err
		}

		result := make([]int, 0)
		for _, d := range deviceList {
			result = append(result, d.ID)
		}

		return result, nil
	}

	return nil, errors.New("不支持的订阅者")
}

func (s *MonitorSubscription) GetResourceURI(siteID string) (string, error) {

	resourceURI, exists := s.Ext["resourceURI"]
	if !exists {
		return "", nil
	}

	uri, ok := resourceURI.(string)
	if !ok {
		return "", errors.New("音频资源错误")
	}

	return uri, nil
}

func (s *MonitorSubscription) GetResourceURL(siteID string) (string, error) {
	resourceURL, exists := s.Ext["resourceURL"]
	if !exists {
		return "", nil
	}

	url, ok := resourceURL.(string)
	if !ok {
		return "", errors.New("音频链接错误")
	}

	return url, nil
}

func (s *MonitorSubscription) GetRepeat(siteID string) (int, error) {
	repeat, exists := s.Ext["repeat"]
	if !exists {
		return 1, nil
	}

	repeatInt, ok := repeat.(int)
	if !ok {
		return 0, errors.New("播放次数错误")
	}

	return repeatInt, nil
}

//mission interface
func (s *StationSubscription) GetMission(siteID string) (*mission.Mission, error) {

	if s.SubscriberType == "user" {
		return nil, errors.New("不支持用户订阅")
	}

	missionType, exists := s.Ext["type"]
	if exists {
		return nil, errors.New("未配置任务类型")
	}

	missionTypeS, ok := missionType.(string)
	if !ok {
		return nil, errors.New("任务类型不正确")
	}

	sm := new(mission.Mission)
	sm.Type = missionTypeS

	sm.Name = fmt.Sprintf("%s-%s 失联", s.Entity.Name, s.Station.Name)
	// sm.Description = s.Duration

	relateID := make(map[string]string)
	relateID["entityID"] = fmt.Sprintf("%d", s.Entity.ID)
	relateID["stationID"] = fmt.Sprintf("%d", s.Station.ID)

	sm.RelateID = relateID

	if err := sm.Validate(siteID); err != nil {
		return nil, err
	}

	return sm, nil
}

func (s *StationSubscription) GetMissionEmpowers(siteID string) (map[string]map[string][]string, error) {

	var result map[string]map[string][]string

	empowers, exists := s.Ext["empowers"]
	if !exists {
		return result, nil
	}

	if err := util.Clone(empowers, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *MonitorSubscription) GetMission(siteID string) (*mission.Mission, error) {
	if s.SubscriberType == "user" {
		return nil, errors.New("不支持用户订阅")
	}

	missionType, exists := s.Ext["type"]
	if exists {
		return nil, errors.New("未配置任务类型")
	}

	missionTypeS, ok := missionType.(string)
	if !ok {
		return nil, errors.New("任务类型不正确")
	}

	sm := new(mission.Mission)
	sm.Type = missionTypeS

	stationName := fmt.Sprintf("%s-%s", s.Entity.Name, s.Station.Name)

	switch s.Type {
	case DATA_DAILY:
		sm.Name = fmt.Sprintf("%s 日均数据异常", stationName)
	case DATA_HOURLY:
		sm.Name = fmt.Sprintf("%s 时均数据异常", stationName)
	case DATA_MINUTELY:
		sm.Name = fmt.Sprintf("%s 分均数据异常", stationName)
	}

	detail := util.FormatDateTime(s.Time)

	for _, d := range s.DataList {
		var value float64

		if rtd, ok := d.(data.IRealTime); ok {
			value = rtd.GetRtd()
		} else if interval, ok := d.(data.IInterval); ok {
			value = interval.GetAvg()
		} else {
			log.Println("error unknown data interface to push")
			continue
		}

		m := monitor.GetMonitor(siteID, d.GetMonitorID())
		if m == nil {
			log.Println("error monitor not found to push: ", d.GetStationID(), d.GetMonitorID())
			continue
		}
		l := monitor.GetFlagLimit(siteID, d.GetStationID(), d.GetMonitorID(), d.GetFlag())
		if l == nil {
			log.Println("error monitor flag limit not found to push: ", d.GetStationID(), d.GetMonitorID(), d.GetFlag())
			continue
		}

		flag, err := monitor.GetFlag(siteID, d.GetFlag())
		if err != nil {
			continue
		}

		if flag == nil {
			continue
		}

		if monitor.CheckFlag(monitor.FLAG_DATA_INVARIANCE, flag.Bits) {
			detail += fmt.Sprintf("%s %s %G(%s小时)\n", m.Name, flag.Name, value, l.Region)
		} else {
			if l.Region != "" {
				detail += fmt.Sprintf("%s %s %G(%s)\n", m.Name, flag.Name, value, l.Region)
			} else {
				detail += fmt.Sprintf("%s %s %G %s\n", m.Name, flag.Name, value, flag.Description)
			}
		}
	}
	sm.Description = detail

	relateID := make(map[string]string)
	relateID["entityID"] = fmt.Sprintf("%d", s.Entity.ID)
	relateID["stationID"] = fmt.Sprintf("%d", s.Station.ID)

	sm.RelateID = relateID

	return sm, nil
}
func (s *MonitorSubscription) GetMissionEmpowers(siteID string) (map[string]map[string][]string, error) {

	var result map[string]map[string][]string

	empowers, exists := s.Ext["empowers"]
	if !exists {
		return result, nil
	}

	if err := util.Clone(empowers, &result); err != nil {
		return nil, err
	}

	return result, nil

}
