package engine

import (
	"log"
	"regexp"

	"obsessiontech/common/config"
)

var Config struct {
	SiteID   string
	MNRegExp string
}

func init() {
	config.GetConfig("config.yaml", &Config)
	log.Println("mn reg exp: ", Config.MNRegExp)
	mnRegexp = regexp.MustCompile(Config.MNRegExp)
}
