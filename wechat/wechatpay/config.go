package wechatpay

import (
	"obsessiontech/common/config"
)

type PayConfig struct {
	WechatAppID             string
	WechatAppSecret         string
	WechatMiniAppID         string
	WechatMiniAppSecret     string
	WechatPayKey            string
	WechatPayMchID          string
	WechatPayNotifyURL      string
	WechatPayRootCA         string
	WechatPayMchCertPath    string
	WechatPayMchCertKeyPath string
}

var Config PayConfig

func init() {
	config.GetConfig("config.yaml", &Config)
}
