package api

import (
	"fmt"

	"obsessiontech/common/config"
)

type InternalApiConfig struct {
	InternalHost string
}

var apiConfig InternalApiConfig

func init() {
	config.GetConfig("config.yaml", &apiConfig)
	if apiConfig.InternalHost == "" {
		apiConfig.InternalHost = "internal"
	}
}

func GetInternalURL(server, version, uri string) string {
	return fmt.Sprintf("http://%s/%s/%s/internal/%s", apiConfig.InternalHost, server, version, uri)
}
