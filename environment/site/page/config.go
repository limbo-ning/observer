package page

import "obsessiontech/common/config"

var Config struct {
	PageExportPath         string
	PageTemplateFolderPath string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
