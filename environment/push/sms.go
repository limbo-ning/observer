package push

import (
	"errors"
	"log"

	"obsessiontech/ali/notify"
)

const (
	PUSH_ALI_SMS = "ali_sms"
)

func init() {
	Register(PUSH_ALI_SMS, new(AliSmsPusher))
}

var e_invalid_sms_push_config = errors.New("未实现短信推送接口")

type IAliSmsPush interface {
	GetTemplateCode(string) (string, error)
	GetMobile(string) (string, error)
	GetSignature(string) (string, error)
	GetSMSParam(string) (map[string]string, error)
}

type AliSmsPusher struct{}

func (p *AliSmsPusher) Validate(siteID string, ipush IPush) error {

	i, ok := ipush.(IAliSmsPush)
	if !ok {
		log.Println("error validate not implement aliSmsPushInterface")
		return e_invalid_sms_push_config
	}

	if _, err := i.GetMobile(siteID); err != nil {
		return err
	}
	if _, err := i.GetTemplateCode(siteID); err != nil {
		return err
	}
	if _, err := i.GetSignature(siteID); err != nil {
		return err
	}

	return nil
}
func (p *AliSmsPusher) Push(siteID string, ipush IPush) error {
	i, ok := ipush.(IAliSmsPush)
	if !ok {
		log.Println("error push not implement aliSmsPushInterface")
		return e_invalid_sms_push_config
	}

	mobile, err := i.GetMobile(siteID)
	if err != nil {
		return err
	}
	templateCode, err := i.GetTemplateCode(siteID)
	if err != nil {
		return err
	}
	signature, err := i.GetSignature(siteID)
	if err != nil {
		return err
	}
	param, err := i.GetSMSParam(siteID)
	if err != nil {
		return err
	}

	return notify.SendSMS(mobile, signature, templateCode, param)
}
