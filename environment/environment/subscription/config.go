package subscription

import "obsessiontech/common/config"

var Config struct {
	// EnvironmentSMSDataPushTemplateCode    string
	// EnvironmentSMSStationPushTemplateCode string
	// EnvironmentWXDataPushTemplateID       string
	// EnvironmentWXStationPushTemplateCode  string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
