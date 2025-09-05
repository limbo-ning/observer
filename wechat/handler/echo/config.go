package echo

import (
	"log"
	"obsessiontech/common/config"
)

type EchoConfig struct {
	Echo []*Setting
}

type Setting struct {
	OpenAccount    string
	Key            string
	PoolSize       int32
	Enable         bool
	EnableInterval []int
}

var echoConfig EchoConfig

func init() {
	config.GetConfig("config.yaml", &echoConfig)
	log.Println(echoConfig)
}
