package config

import (
	"errors"
	"io/ioutil"
	"log"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

var E_config_file_access_failed = errors.New("config file access failed")
var E_config_file_invalid = errors.New("config file invalid")

var lock sync.RWMutex
var cache = make(map[string][]byte)

func GetConfig(filePath string, configStruct interface{}) error {

	lock.RLock()
	if content, exists := cache[filePath]; exists {
		if err := yaml.Unmarshal(content, configStruct); err == nil {
			lock.RUnlock()
			return nil
		}
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()

	if content, exists := cache[filePath]; exists {
		if err := yaml.Unmarshal(content, configStruct); err == nil {
			return nil
		}
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("%s file acess failed: %v", filePath, err)
		return E_config_file_access_failed
	}
	if err = yaml.Unmarshal(content, configStruct); err != nil {
		log.Printf("parsing %s failed:%v", filePath, err)
		return E_config_file_invalid
	}

	cache[filePath] = content

	return nil
}
