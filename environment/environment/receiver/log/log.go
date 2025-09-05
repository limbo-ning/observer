package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/util"
)

var Config struct {
	LogDir string
}

var lock sync.RWMutex
var logs = make(map[string]*os.File)

func init() {
	config.GetConfig("config.yaml", &Config)
	if len(Config.LogDir) > 0 && !strings.HasSuffix(Config.LogDir, "/") {
		Config.LogDir += "/"
	}
}

func get(mn string) (*os.File, error) {

	lock.RLock()

	logFile, exists := logs[mn]
	if exists {
		defer lock.RUnlock()
		return logFile, nil
	}

	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()

	logFile, exists = logs[mn]
	if exists {
		return logFile, nil
	}

	logFileName := fmt.Sprintf("%s%s.log", Config.LogDir, mn)

	info, err := os.Stat(logFileName)
	if err == nil {
		log.Println("log file got: ", info.Name(), info.Size())
		logFile, err = os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	} else if os.IsNotExist(err) {
		logFile, err = os.Create(logFileName)
	}

	if err != nil {
		return nil, err
	}

	logs[mn] = logFile

	return logFile, nil
}

func Log(mn, msg string, args ...interface{}) {

	logFile, err := get(mn)
	if err != nil {
		log.Println("error get log file: ", err)
		return
	}

	prefix := fmt.Sprintf("%s【%s】", util.FormatDateTime(time.Now()), mn)

	if _, err := io.WriteString(logFile, prefix+fmt.Sprintf(msg, args...)+"\n"); err != nil {
		log.Println("error write log: ", err)
	}
}
