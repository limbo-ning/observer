package util

import (
	"obsessiontech/common/config"
	"time"
)

var Config struct {
	WechatAppID         string
	WechatAppSecret     string
	WechatMiniAppID     string
	WechatMiniAppSecret string

	WechatPlatformAppID            string
	WechatPlatformAppSecret        string
	WechatPlatformEncryptKey       string
	WechatPlatformEncryptToken     string
	WechatPlatformVerifyTicketPath string

	IsWechatPlatformHost     bool
	WechatPlatformHostType   string
	WechatPlatformHost       string
	WechatPlatformTimeoutSec time.Duration
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
