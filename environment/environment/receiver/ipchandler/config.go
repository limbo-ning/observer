package ipchandler

import "obsessiontech/common/config"

var Config struct {
	SiteID string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
