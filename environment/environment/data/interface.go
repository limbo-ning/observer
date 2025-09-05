package data

import (
	"obsessiontech/common/util"
	"sync"
)

type IData interface {
	GetID() int
	SetID(int)
	GetDataType() string
	GetMonitorID() int
	SetMonitorID(int)
	GetMonitorCodeID() int
	SetMonitorCodeID(int)
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

type data struct {
	ID            int                    `json:"ID"`
	DataTime      util.Time              `json:"dataTime"`
	MonitorID     int                    `json:"monitorID"`
	MonitorCodeID int                    `json:"monitorCodeID"`
	StationID     int                    `json:"stationID"`
	Flag          string                 `json:"flag"`
	FlagBit       int                    `json:"flagBit"`
	OriginData    map[string]interface{} `json:"originData,omitempty"`

	OriginDataLock sync.RWMutex `json:"-"`
}

func (d *data) GetID() int   { return d.ID }
func (d *data) SetID(id int) { d.ID = id }

func (d *data) GetMonitorID() int                     { return d.MonitorID }
func (d *data) GetMonitorCodeID() int                 { return d.MonitorCodeID }
func (d *data) GetStationID() int                     { return d.StationID }
func (d *data) GetDataTime() util.Time                { return d.DataTime }
func (d *data) GetFlag() string                       { return d.Flag }
func (d *data) GetFlagBit() int                       { return d.FlagBit }
func (d *data) GetOriginData() map[string]interface{} { return d.OriginData }

func (d *data) SetMonitorID(monitorID int)                      { d.MonitorID = monitorID }
func (d *data) SetMonitorCodeID(monitorCodeID int)              { d.MonitorCodeID = monitorCodeID }
func (d *data) SetStationID(stationID int)                      { d.StationID = stationID }
func (d *data) SetDataTime(dataTime util.Time)                  { d.DataTime = dataTime }
func (d *data) SetFlag(flag string)                             { d.Flag = flag }
func (d *data) SetFlagBit(flagBit int)                          { d.FlagBit = flagBit }
func (d *data) SetOriginData(originData map[string]interface{}) { d.OriginData = originData }

func (d *data) LockOriginData()    { d.OriginDataLock.Lock() }
func (d *data) UnLockOriginData()  { d.OriginDataLock.Unlock() }
func (d *data) RLockOriginData()   { d.OriginDataLock.RLock() }
func (d *data) RUnlockOriginData() { d.OriginDataLock.RUnlock() }

func (d *data) SetCode(code string) {

	d.LockOriginData()
	defer d.UnLockOriginData()

	if d.OriginData == nil {
		d.OriginData = make(map[string]interface{})
	}
	d.OriginData["code"] = code
}

func (d *data) GetCode() string {

	d.RLockOriginData()
	defer d.RUnlockOriginData()

	if d.OriginData != nil {
		code := d.OriginData["code"]
		if code, ok := code.(string); ok {
			return code
		}
	}

	return ""
}

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

func (d *interval) GetAvg() float64 { return d.Avg }
func (d *interval) GetMin() float64 { return d.Min }
func (d *interval) GetMax() float64 { return d.Max }
func (d *interval) GetCou() float64 { return d.Cou }

func (d *interval) SetAvg(avg float64) { d.Avg = avg }
func (d *interval) SetMin(min float64) { d.Min = min }
func (d *interval) SetMax(max float64) { d.Max = max }
func (d *interval) SetCou(cou float64) { d.Cou = cou }

type IReview interface {
	GetReviewed() bool
	SetReviewed(bool)
}

type review struct {
	Reviewed bool `json:"reviewed,omitempty"`
}

func (d *review) GetReviewed() bool       { return d.Reviewed }
func (d *review) SetReview(reviewed bool) { d.Reviewed = reviewed }
