package engine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

var e_auth_method_invalid = errors.New("不支持的验证方式")
var e_register_unavailable = errors.New("不开放注册")
var e_user_exists = errors.New("用户已存在")

var registration = make(map[string]func() IAuth)
var nameResitry = make(map[string]string)

func register(method, moduleName string, factoryFunc func() IAuth) {
	if _, exists := registration[method]; exists {
		panic("auth engine duplicated: " + method)
	}
	registration[method] = factoryFunc
	nameResitry[method] = moduleName
}

func GetAuthMethod(siteID string) map[string]map[string]any {
	result := make(map[string]map[string]any)

	for method, fac := range registration {
		_, siteModule, err := site.GetSiteModule(siteID, nameResitry[method], true)
		if err != nil {
			log.Println("error get auth method site module: ", method, err)
			continue
		}

		instance := fac()

		paramBytes, _ := json.Marshal(siteModule.Param)
		if err := json.Unmarshal(paramBytes, instance); err != nil {
			log.Println("error unmarshal auth:", err)
			continue
		}

		registerAvailable, exists := siteModule.Param["registerAvailable"]
		if exists {
			result[method] = instance.Tip()
			result[method]["registerAvailable"] = registerAvailable
		}

	}

	return result
}

func GetAuth(siteID, method string) (IAuth, error) {
	if factory, exists := registration[method]; exists {
		instance := factory()

		_, siteModule, err := site.GetSiteModule(siteID, nameResitry[method], true)
		if err != nil {
			log.Println("error get site module: ", err)
			return instance, err
		}
		paramBytes, _ := json.Marshal(siteModule.Param)
		if err := json.Unmarshal(paramBytes, instance); err != nil {
			log.Println("error unmarshal auth:", err)
			return nil, err
		}
		return instance, nil
	}

	return nil, e_auth_method_invalid
}

func SaveAuth(siteID, method string, paramBytes []byte) error {

	fac, exists := registration[method]
	if !exists {
		return e_auth_method_invalid
	}

	instance := fac()
	if err := json.Unmarshal(paramBytes, &instance); err != nil {
		return err
	}

	if err := instance.Validate(); err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, nameResitry[method], true)
		if err != nil {
			panic(err)
		}

		json.Unmarshal(paramBytes, &sm.Param)
		if err := sm.Save(siteID, txn); err != nil {
			panic(err)
		}
	})
}
