package logging

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

const (
	MODULE_LOGGING = "logging"

	ACTION_ADMIN_VIEW        = "admin_view"
	ACTION_ADMIN_VIEW_MODULE = "admin_view_module"
	ACTION_ADMIN_EDIT_MODULE = "admin_edit_module"
)

type LoggingModule struct {
	Loggers map[string]map[string][]string `json:"loggers"`
}

var e_logger_not_support = errors.New("不支持的记录类型")

func (m *LoggingModule) ShouldLog(moduleID, source, action string) bool {
	if sources, exists := m.Loggers[moduleID]; exists {
		if actions, exists := sources[source]; exists {
			for _, a := range actions {
				if action == a {
					return true
				}
			}
		}
	}

	return false
}

func GetModule(siteID string) (*LoggingModule, error) {
	var m *LoggingModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_LOGGING, false)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal logging module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal logging module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *LoggingModule) Save(siteID string) error {

	for moduleID, loggers := range m.Loggers {
		registrants, exists := registry[moduleID]
		if !exists {
			return e_logger_not_support
		}

		sources := make(map[string]*Registrant)
		for _, s := range registrants {
			sources[s.Source] = s
		}

		for source, actions := range loggers {
			sourceRegistrant, exists := sources[source]
			if !exists {
				return e_logger_not_support
			}

			allowedActions := make(map[string]byte)

			for _, a := range sourceRegistrant.Actions {
				allowedActions[a.Action] = 1
			}

			for _, action := range actions {
				if _, exists := allowedActions[action]; !exists {
					return e_logger_not_support
				}
			}
		}

	}

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_LOGGING, true)
		if err != nil {
			panic(err)
		}

		paramByte, _ := json.Marshal(&m)
		json.Unmarshal(paramByte, &sm.Param)

		if err := sm.Save(siteID, txn); err != nil {
			panic(err)
		}
	})
}
