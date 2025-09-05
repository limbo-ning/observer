package logging

import (
	"errors"

	"obsessiontech/environment/site"
)

var registry = make(map[string][]*Registrant)

type Registrant struct {
	Source  string              `json:"source"`
	Name    string              `json:"name"`
	Actions []*ActionRegistrant `json:"actions"`
}

type ActionRegistrant struct {
	Action string `json:"action"`
	Name   string `json:"name"`
}

func Register(moduleID string, registrant ...*Registrant) {
	if _, exists := registry[moduleID]; exists {
		panic(errors.New("duplicate module register at logging registry"))
	}
	registry[moduleID] = registrant
}

func ParseRegistrant(souce, name string, actionPair ...[2]string) *Registrant {
	r := &Registrant{
		Source:  souce,
		Name:    name,
		Actions: make([]*ActionRegistrant, 0),
	}

	for _, pair := range actionPair {
		r.Actions = append(r.Actions, &ActionRegistrant{
			Action: pair[0],
			Name:   pair[1],
		})
	}

	return r
}

type Logger struct {
	ModuleID    string        `json:"moduleID,omitempty"`
	Name        string        `json:"name,omitempty"`
	Registrants []*Registrant `json:"registrants"`
}

func GetLoggerRegistrants(siteID string) ([]*Logger, error) {

	siteModules, err := site.GetSiteModuleList(siteID)
	if err != nil {
		return nil, err
	}

	result := make([]*Logger, 0)

	for _, sm := range siteModules {
		if r, exists := registry[sm.ModuleID]; exists {
			result = append(result, &Logger{
				ModuleID:    sm.ModuleID,
				Name:        sm.Name,
				Registrants: r,
			})
		}
	}

	return result, nil
}

func GetLogger(siteID, moduleID, source string) (*Registrant, error) {
	loggingModule, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	logger, exists := loggingModule.Loggers[moduleID]
	if !exists {
		return nil, e_logger_not_support
	}

	_, exists = logger[source]
	if !exists {
		return nil, e_logger_not_support
	}

	registrants, exists := registry[moduleID]
	if !exists {
		return nil, e_logger_not_support
	}

	for _, r := range registrants {
		if r.Source == source {
			return r, nil
		}
	}

	return nil, e_logger_not_support
}
