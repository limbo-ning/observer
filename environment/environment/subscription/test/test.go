package main

/*
#cgo LDFLAGS: -ldl
*/
import "C"
import (
	"log"
	"obsessiontech/common/util"
	"time"
)

//export getRecentDataValue
func getRecentDataValue(siteID *C.char, dataType *C.char, stationID int, monitorID int) float64 {

	return 0
}

//export getMonitorName
func getMonitorName(siteID *C.char, monitorID int) *C.char {

	return C.CString("-")
}

//export getMonitorFlagLimit
func getMonitorFlagLimit(siteID *C.char, stationID int, monitorID int, flag *C.char) *C.char {

	return C.CString("-")
}

//export getMonitorFlagName
func getMonitorFlagName(siteID *C.char, flag *C.char) *C.char {

	goFlag := C.GoString(flag)
	return C.CString(goFlag)
}

const (
	STATION_STATUS = "station_status"
	DATA_DAILY     = "data_daily"
	DATA_HOURLY    = "data_hourly"
	DATA_MINUTELY  = "data_minutely"
	DATA_REAL_TIME = "data_realtime"
)

const REAL_TIME = "realTime"
const MINUTELY = "minutely"
const HOURLY = "hourly"
const DAILY = "daily"

type IData interface {
	GetID() int
	SetID(int)
	GetDataType() string
	GetMonitorID() int
	SetMonitorID(int)
	GetStationID() int
	SetStationID(int)
	GetDataTime() util.Time
	SetDataTime(util.Time)
	GetFlag() string
	SetFlag(string)
	GetFlagBit() int
	SetFlagBit(int)
	GetOriginData() map[string]interface{}
	SetOriginData(originData map[string]interface{})

	LockOriginData()
	UnLockOriginData()

	RLockOriginData()
	RUnlockOriginData()

	SetCode(string)
	GetCode() string
}

type hourlydata struct {
	Value     float64
	ID        int
	MonitorID int
	StationID int
	DataTime  util.Time
	Flag      string
	FlagBit   int
}

func (h *hourlydata) GetID() int                            { return h.ID }
func (h *hourlydata) SetID(i int)                           { h.ID = i }
func (h *hourlydata) GetDataType() string                   { return "hourly" }
func (h *hourlydata) GetMonitorID() int                     { return h.MonitorID }
func (h *hourlydata) SetMonitorID(i int)                    { h.MonitorID = i }
func (h *hourlydata) GetStationID() int                     { return h.StationID }
func (h *hourlydata) SetStationID(i int)                    { h.StationID = i }
func (h *hourlydata) GetDataTime() util.Time                { return h.DataTime }
func (h *hourlydata) SetDataTime(t util.Time)               { h.DataTime = t }
func (h *hourlydata) GetFlag() string                       { return h.Flag }
func (h *hourlydata) SetFlag(f string)                      { h.Flag = f }
func (h *hourlydata) GetFlagBit() int                       { return 0 }
func (h *hourlydata) SetFlagBit(i int)                      {}
func (h *hourlydata) GetOriginData() map[string]interface{} { return make(map[string]interface{}) }
func (h *hourlydata) SetOriginData(map[string]interface{})  {}
func (h *hourlydata) LockOriginData()                       {}
func (h *hourlydata) UnLockOriginData()                     {}

func (h *hourlydata) RLockOriginData()   {}
func (h *hourlydata) RUnlockOriginData() {}

func (h *hourlydata) SetCode(string)  {}
func (h *hourlydata) GetCode() string { return "" }

func (h *hourlydata) GetAvg() float64 { return h.Value }
func (h *hourlydata) SetAvg(float64)  {}
func (h *hourlydata) GetMin() float64 { return h.Value }
func (h *hourlydata) SetMin(float64)  {}
func (h *hourlydata) GetMax() float64 { return h.Value }
func (h *hourlydata) SetMax(float64)  {}
func (h *hourlydata) GetCou() float64 { return h.Value }
func (h *hourlydata) SetCou(float64)  {}

type IRealTime interface {
	GetRtd() float64
	SetRtd(rtd float64)
}

type IInterval interface {
	GetAvg() float64
	SetAvg(float64)
	GetMin() float64
	SetMin(float64)
	GetMax() float64
	SetMax(float64)
	GetCou() float64
	SetCou(float64)
}

type Entity struct {
	ID        int                    `json:"ID"`
	Name      string                 `json:"name"`
	Address   string                 `json:"address"`
	Longitude float64                `json:"longitude"`
	Latitude  float64                `json:"latitude"`
	GeoType   string                 `json:"geoType"`
	Ext       map[string]interface{} `json:"ext"`
}

type Station struct {
	ID          int                    `json:"ID"`
	EntityID    int                    `json:"entityID"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	OnlineTime  util.Time              `json:"onlineTime"`
	Status      string                 `json:"status"`
	MN          string                 `json:"mn"`
	Protocol    string                 `json:"protocol"`
	Redirect    string                 `json:"redirect"`
	Ext         map[string]interface{} `json:"ext"`
}

func main() {
	// if result, err := getSMSParam("", "/Users/limbo/GIT/ob_server/c/push/environment/keqin/keqin_station_push.so", "station_status", false, &Entity{Name: "abc"}, &Station{Name: "ABC"}, time.Now().Add(time.Hour*-1), nil); err != nil {
	// 	log.Println(err)
	// } else {
	// 	log.Println(result)
	// }
	dataList := make([]IData, 0)
	dataList = append(dataList, &hourlydata{
		StationID: 10,
		MonitorID: 5,
		Flag:      "O",
		Value:     20,
		DataTime:  util.Time(time.Now()),
	})
	if result, err := getSMSParam("", "/Users/limbo/GIT/ob_server/c/push/environment/keqin/keqin_data_push.so", "data_hourly", false, &Entity{Name: "abc"}, &Station{Name: "ABC"}, time.Time(dataList[0].GetDataTime()), dataList); err != nil {
		log.Println(err)
	} else {
		log.Println(result)
	}
	// if err := hello("/Users/limbo/GIT/ob_server/c/push/environment/keqin/keqin_station_push.so"); err != nil {
	// 	log.Println(err)
	// }
}
