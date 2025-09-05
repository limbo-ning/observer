package resource

import "obsessiontech/common/config"

var Config struct {
	ResourceFolderPath string
	PermitFileType     []struct {
		Name   string
		Subfix []struct {
			Name string
		}
	}
}

func init() {
	config.GetConfig("config.yaml", &Config)
}
