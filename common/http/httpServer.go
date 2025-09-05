package http

import (
	"fmt"
	"runtime"

	"github.com/gin-gonic/gin"

	"obsessiontech/common/config"
)

type ServerConfig struct {
	Port       string
	ServerHost string
	ServerName string
	Version    string
}

var serverConfig ServerConfig

func init() {
	config.GetConfig("config.yaml", &serverConfig)
}

func GetPrefix() string {
	return fmt.Sprintf("%s/%s", serverConfig.ServerName, serverConfig.Version)
}

func GetHost() string {
	return serverConfig.ServerHost
}

func GetPort() string {
	return serverConfig.Port
}

func GetEngine() *gin.Engine {
	runtime.GOMAXPROCS(runtime.NumCPU())
	gin.SetMode(gin.ReleaseMode)
	return gin.Default()
}

func GetObEngine() *gin.Engine {
	runtime.GOMAXPROCS(runtime.NumCPU())
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Recovery())
	return e
}
