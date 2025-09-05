package ipcmessage

import (
	"encoding/json"
	"errors"

	"obsessiontech/environment/environment/data"
)

const (
	ack = iota
	listenReq

	moduleReload
	stationReload
	monitorReload
	monitorCodeReload
	flagLimitReload

	stationStatusReq
	stationStatusRes

	controlReq
	controlRes

	stationOffline

	rtd
	minutely
	hourly
	daily
)

type IMessage interface {
	GetIPCMessageType() int
}

type Ack int

func (m *Ack) GetIPCMessageType() int { return ack }

type ListenReq bool

func (m *ListenReq) GetIPCMessageType() int { return listenReq }

type ModuleReloadReq struct{}

func (m *ModuleReloadReq) GetIPCMessageType() int { return moduleReload }

type StationReloadReq int

func (m *StationReloadReq) GetIPCMessageType() int { return stationReload }

type MonitorReloadReq struct{}

func (m *MonitorReloadReq) GetIPCMessageType() int { return monitorReload }

type MonitorCodeReloadReq struct{}

func (m *MonitorCodeReloadReq) GetIPCMessageType() int { return monitorCodeReload }

type FlagLimitReloadReq struct{}

func (m *FlagLimitReloadReq) GetIPCMessageType() int { return flagLimitReload }

type StationStatusReq []int

func (m *StationStatusReq) GetIPCMessageType() int { return stationStatusReq }

type StationStatusRes map[int]bool

func (m *StationStatusRes) GetIPCMessageType() int { return stationStatusRes }

type ControlReq struct{}

func (m *ControlReq) GetIPCMessageType() int { return controlReq }

type ControlRes struct{}

func (m *ControlRes) GetIPCMessageType() int { return controlRes }

type StationOffline int

func (m *StationOffline) GetIPCMessageType() int { return stationOffline }

type RealTime data.RealTimeData

func (m *RealTime) GetIPCMessageType() int { return rtd }

type Minutely data.MinutelyData

func (m *Minutely) GetIPCMessageType() int { return minutely }

type Hourly data.HourlyData

func (m *Hourly) GetIPCMessageType() int { return hourly }

type Daily data.DailyData

func (m *Daily) GetIPCMessageType() int { return daily }

var E_unmarshal_failure = errors.New("json unmarshal failed")
var E_message_type_unknown = errors.New("message type unknown")

func ParseMessage(input []byte) (IMessage, error) {
	var datagram struct {
		MessageType int             `json:"type"`
		Message     json.RawMessage `json:"message"`
	}

	if err := json.Unmarshal(input, &datagram); err != nil {
		return nil, E_unmarshal_failure
	}

	var message IMessage

	switch datagram.MessageType {
	case ack:
		message = new(Ack)
	case listenReq:
		message = new(ListenReq)
	case moduleReload:
		message = new(ModuleReloadReq)
	case stationReload:
		message = new(StationReloadReq)
	case monitorReload:
		message = new(MonitorReloadReq)
	case monitorCodeReload:
		message = new(MonitorCodeReloadReq)
	case flagLimitReload:
		message = new(FlagLimitReloadReq)
	case stationStatusReq:
		message = new(StationStatusReq)
	case stationStatusRes:
		message = new(StationStatusRes)
	case controlReq:
		message = new(ControlReq)
	case controlRes:
		message = new(ControlRes)
	case stationOffline:
		message = new(StationOffline)
	case rtd:
		message = new(RealTime)
	case minutely:
		message = new(Minutely)
	case hourly:
		message = new(Hourly)
	case daily:
		message = new(Daily)
	}

	if err := json.Unmarshal(datagram.Message, &message); err != nil {
		return nil, E_unmarshal_failure
	}

	return message, nil
}

func WrapMessage(message IMessage) ([]byte, error) {

	datagram := map[string]interface{}{
		"type":    message.GetIPCMessageType(),
		"message": message,
	}

	return json.Marshal(datagram)
}
