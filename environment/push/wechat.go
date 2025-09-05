package push

import (
	"errors"
	"log"
	"strings"

	"obsessiontech/wechat/template"
)

const (
	PUSH_WXMINIAPP_SUBSCRIPTION = "wxminiapp_subscription"
	PUSH_WXOPEN_TEMPLATE        = "wxopen_template"
)

func init() {
	Register(PUSH_WXMINIAPP_SUBSCRIPTION, new(WxMiniAppSubscriptionPusher))
	Register(PUSH_WXOPEN_TEMPLATE, new(WxOpenTemplatePusher))
}

type IWxMiniAppSubscriptionPush interface {
	GetAccessToken(string) (string, error)
	GetTemplateID(string) (string, error)
	GetOpenID(string) (string, error)
	GetPage(string) (string, error)
	GetMiniAppSubscriptionParam(string) (map[string]interface{}, error)
	Delete(string) error
}

type WxMiniAppSubscriptionPusher struct{}

func (p *WxMiniAppSubscriptionPusher) Validate(siteID string, ipush IPush) error {

	i, ok := ipush.(IWxMiniAppSubscriptionPush)
	if !ok {
		log.Println("error validate not implement WxMiniAppSubscriptionPushInterface")
		return e_invalid_sms_push_config
	}

	if _, err := i.GetAccessToken(siteID); err != nil {
		return err
	}
	if _, err := i.GetTemplateID(siteID); err != nil {
		return err
	}
	if _, err := i.GetOpenID(siteID); err != nil {
		return err
	}

	return nil
}

func (p *WxMiniAppSubscriptionPusher) Push(siteID string, ipush IPush) error {
	i, ok := ipush.(IWxMiniAppSubscriptionPush)
	if !ok {
		log.Println("error push not implement WxMiniAppSubscriptionPushInterface")
		return e_invalid_sms_push_config
	}

	//微信小程序订阅只有一次
	defer func() {
		i.Delete(siteID)
	}()

	accessToken, err := i.GetAccessToken(siteID)
	if err != nil {
		return err
	}
	templateID, err := i.GetTemplateID(siteID)
	if err != nil {
		return err
	}
	openID, err := i.GetOpenID(siteID)
	if err != nil {
		return err
	}
	page, err := i.GetPage(siteID)
	if err != nil {
		return err
	}
	param, err := i.GetMiniAppSubscriptionParam(siteID)
	if err != nil {
		return err
	}

	for _, id := range strings.Split(openID, ",") {
		if err := template.PlatformPushMiniAppSubscription(accessToken, templateID, id, page, param); err != nil {
			log.Println("error push miniapp template: ", err)
		}
	}

	return nil
}

var e_invalid_wxopen_template_push_config = errors.New("微信公众号模版消息推送参数不正确")

type IWxOpenTemplatePush interface {
	GetPushParam(string) ([]WxOpenTemplatePushParam, error)
	// GetAccessToken(string) (string, error)
	// GetTemplateID(string) (string, error)
	// GetOpenID(string) (string, error)
	// GetURL(string) (string, error)
	// GetMiniappAppID(string) (string, error)
	// GetMiniappPage(string) (string, error)
	// GetFirst(string) (string, error)
	// GetRemark(string) (string, error)
	// GetKeywords(string) ([]string, error)
	Delete(string) error
}
type WxOpenTemplatePushParam struct {
	AccessToken  string
	TemplateID   string
	OpenID       string
	URL          string
	MiniappAppID string
	MiniappPage  string
	First        string
	Remark       string
	Keywords     []string
}

type WxOpenTemplatePusher struct{}

func (p *WxOpenTemplatePusher) Validate(siteID string, ipush IPush) error {

	i, ok := ipush.(IWxOpenTemplatePush)
	if !ok {
		log.Println("error validate not implement WxOpenTemplatePushInterface")
		return e_invalid_sms_push_config
	}

	if _, err := i.GetPushParam(siteID); err != nil {
		return err
	}

	// if _, err := i.GetAccessToken(siteID); err != nil {
	// 	return err
	// }
	// if _, err := i.GetTemplateID(siteID); err != nil {
	// 	return err
	// }
	// if _, err := i.GetOpenID(siteID); err != nil {
	// 	return err
	// }

	return nil
}

func (p *WxOpenTemplatePusher) Push(siteID string, ipush IPush) error {
	i, ok := ipush.(IWxOpenTemplatePush)
	if !ok {
		log.Println("error push not implement WxOpenTemplatePushInterface")
		return e_invalid_sms_push_config
	}

	params, err := i.GetPushParam(siteID)
	if err != nil {
		return err
	}

	for _, param := range params {
		accessToken := param.AccessToken
		templateID := param.TemplateID
		openID := param.OpenID
		first := param.First
		remark := param.Remark
		url := param.URL
		miniappAppID := param.MiniappAppID
		miniappPage := param.MiniappPage
		keywords := param.Keywords

		for _, id := range strings.Split(openID, ",") {
			if err := template.PlatformPushOpenTemplate(accessToken, templateID, id, first, remark, url, miniappAppID, miniappPage, keywords...); err != nil {
				log.Println("error push open template: ", err)
			}
		}
	}

	return nil
}
