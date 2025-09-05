package data

import (
	"fmt"
	"log"
	"time"

	"obsessiontech/common/util"
)

const REAL_TIME = "realTime"
const MINUTELY = "minutely"
const HOURLY = "hourly"
const DAILY = "daily"

type RealTimeData struct {
	data
	Rtd float64 `json:"rtd"`
}

func (d *RealTimeData) GetDataType() string { return REAL_TIME }
func (d *RealTimeData) GetRtd() float64     { return d.Rtd }
func (d *RealTimeData) SetRtd(rtd float64)  { d.Rtd = rtd }

type interval struct {
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Cou float64 `json:"cou"`
}

type MinutelyData struct {
	data
	interval
}

func (d *MinutelyData) GetDataType() string { return MINUTELY }

type HourlyData struct {
	data
	interval
	review
}

func (d *HourlyData) GetDataType() string { return HOURLY }

type DailyData struct {
	data
	interval
	review
}

func (d *DailyData) GetDataType() string { return DAILY }

func TableName(siteID, dataType string) string {
	switch dataType {
	case REAL_TIME:
		return siteID + "_realtimedata"
	case MINUTELY:
		return siteID + "_minutelydata"
	case HOURLY:
		return siteID + "_hourlydata"
	case DAILY:
		return siteID + "_dailydata"
	}

	return ""
}

type DataTable struct {
	Name      string    `json:"name"`
	Table     string    `json:"table"`
	BeginTime util.Time `json:"beginTime"`
	EndTime   util.Time `json:"endTime"`
	Status    string    `json:"status"`
}

func FetchableTables(siteID, dataType string) ([]*DataTable, error) {
	result := make([]*DataTable, 0)

	dataModule, err := GetModule(siteID)
	if err != nil {
		log.Println("error rotate: get data module", err)
		return nil, err
	}

	current := new(DataTable)
	result = append(result, current)
	current.EndTime = util.Time(util.GetEndOfDate(time.Now()))
	current.Name = "- " + time.Now().Format("20060102")
	current.Table = TableName(siteID, dataType)[len(siteID+"_"):]
	current.Status = active
	for _, r := range dataModule.Rotations {
		if r.DataType == dataType {
			beginTime := r.getActiveTime()
			current.BeginTime = util.Time(beginTime)
			current.Name = fmt.Sprintf("%s - %s", beginTime.Format("20060102"), time.Now().Format("20060102"))
			break
		}
	}

	_, archives := getArchiveTables(siteID, dataType)

	for _, archive := range archives {

		log.Println("archive: ", archive.TableName, archive.Status)

		toShow := new(DataTable)
		result = append(result, toShow)
		toShow.BeginTime = util.Time(archive.BeginTime)
		toShow.EndTime = util.Time(archive.EndTime)
		toShow.Status = archive.Status
		toShow.Name = fmt.Sprintf("%s - %s", archive.BeginTime.Format("20060102"), archive.EndTime.Format("20060102"))
		toShow.Table = archive.TableName[len(siteID+"_"):]
	}

	return result, nil
}

func FetchTableNames(siteID, dataType string, beginTime, endTime time.Time) []string {

	result := make([]string, 0)
	result = append(result, TableName(siteID, dataType))

	var isWithin bool
	_, archives := getArchiveTables(siteID, dataType)

	for _, archive := range archives {
		if archive.Status == active {
			isWithin, _ = checkTable(util.GetDate(beginTime), util.GetDate(endTime), archive.BeginTime, archive.EndTime)
			if isWithin {
				result = append(result, archive.TableName)
				TriggerArchiveRollback(siteID, dataType, archive.TableName)
			}
		}
	}

	return result
}

func checkTable(beginTime, endTime, tableBeginTime, tableEndTime time.Time) (isWithin, isBeyond bool) {

	beginTime = util.GetDate(beginTime)
	endTime = util.GetDate(endTime)

	isBefore := beginTime.Before(tableBeginTime)
	isAfter := endTime.After(tableEndTime)

	isBeyond = beginTime.Before(tableBeginTime)

	if isAfter && isBefore {
		isWithin = true
	} else if isAfter {
		isWithin = beginTime.Before(tableEndTime) || beginTime.Equal(tableEndTime)
	} else if isBefore {
		isWithin = endTime.After(tableBeginTime) || endTime.Equal(tableBeginTime)
	} else {
		isWithin = true
	}

	return
}
