package subscription

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_SUBSCRIPTION = "environment_subscription"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"
)

type SubscriptionModule struct {
	SMSSignature string `json:"smsSignature"`

	WxSettings   []*WxSetting            `json:"wxSettings"`
	PushSettings map[string]*PushSetting `json:"pushSettings"`
}

type WxSetting struct {
	WxOpenAppID            string `json:"wxOpenAppID"`
	WxMiniappAppID         string `json:"wxMiniappAppID"`
	WxMiniappStationPage   string `json:"wxMiniappStationPage"`
	WxMiniappOverproofPage string `json:"wxMiniappOverproofPage"`
}

type PushSetting struct {
	Trigger *PushDetail `json:"trigger"`
	Cease   *PushDetail `json:"cease"`
}

type PushDetail struct {
	FlagThresholdCount map[string]int    `json:"flagThresholdCount,omitempty"`
	CooldownMin        time.Duration     `json:"cooldownMin,omitempty"`
	LDLibPath          string            `json:"ldLibPath"`
	SMSTemplateID      string            `json:"smsTemplateID,omitempty"`
	WxOpenTemplateIDs  map[string]string `json:"wxOpenTemplateIDs,omitempty"`
}

func GetModule(siteID string) (*SubscriptionModule, error) {
	var m *SubscriptionModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_SUBSCRIPTION, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal subscription module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal subscription module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *SubscriptionModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_SUBSCRIPTION, true)
		if err != nil {
			panic(err)
		}

		paramByte, _ := json.Marshal(&m)
		json.Unmarshal(paramByte, &sm.Param)

		if err := sm.Save(siteID, txn); err != nil {
			panic(err)
		}
	})
}
