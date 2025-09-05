package event

import (
	"database/sql"
	"errors"
)

var registry = make(map[string]func() IEvent)

func Register(eventType string, fac func() IEvent) {
	if _, exists := registry[eventType]; exists {
		panic("duplicate event: " + eventType)
	}
	registry[eventType] = fac
}

func GetEvent(eventType string) (IEvent, error) {
	fac, exists := registry[eventType]
	if !exists {
		return nil, errors.New("不支持的事件:" + eventType)
	}

	return fac(), nil
}

type IEvent interface {
	ValidateEvent(siteID string, event *Event) error
	ExecuteEvent(siteID string, txn *sql.Tx, event *Event) error
}
