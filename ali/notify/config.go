package notify

import "obsessiontech/common/config"

var Config struct {
	AliAccessKey    string
	AliAccessSecret string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
