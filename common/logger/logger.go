package logger

import (
	"log"
	"obsessiontech/common/config"
	"os"
)

type logConfig struct {
	Logfile string
}

var LogFile *os.File

func init() {
	SetLog()
}

func SetLog() {
	if LogFile == nil {
		var Config logConfig
		config.GetConfig("config.yaml", &Config)

		file, err := os.OpenFile(Config.Logfile, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			if os.IsNotExist(err) {
				file, err = os.Create(Config.Logfile)
				log.Fatalln("fail to create log file!", Config.Logfile)
			}
			log.Fatalln("fail to open log file!", Config.Logfile)
		}

		LogFile = file
	}
	log.SetOutput(LogFile)
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
}
