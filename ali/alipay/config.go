package alipay

import "obsessiontech/common/config"

var Config struct {
	AlipayAppID              string
	AlipayNotifyURL          string
	AlipayWapPayReturnURL    string
	AlipayPrivateKey         string
	AlipayPlatformAppID      string
	AlipayPlatformPrivateKey string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
