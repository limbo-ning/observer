package wechat

import (
	"obsessiontech/common/config"
)

type WechatConfig struct {
	WechatAppID        string
	WechatAppSecret    string
	WechatKey          string
	WechatPayMchID     string
	WechatPayNotifyURL string
}

var Config WechatConfig

func init() {
	config.GetConfig("config.yaml", &Config)
}
