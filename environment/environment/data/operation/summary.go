package operation

import (
	"database/sql"
	"fmt"
	"log"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
	"time"
)

func init() {
	dataprocess.Register("summary", func() dataprocess.IDataProcessor { return new(summaryProcessor) })
}

type summaryProcessor struct {
	dataprocess.BaseDataProcessor
	TargetMonitorID     int     `json:"targetMonitorID"`
	TargetMonitorCodeID int     `json:"targetMonitorCodeID"`
	Stops               []*stop `json:"stops,omitempty"`
}

type stop struct {
	TargetCondition data.Criterias         `json:"targetConditions"`
	SourceCriteria  data.Criterias         `json:"sourceCriterias"`
	Setters         map[string]interface{} `json:"setters"`
}

func (p *summaryProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) (bool, error) {
	log.Println("run summary: ", entry.GetMonitorID(), entry.GetStationID(), entry.GetDataTime())

	_, unuploaded, lock := uploader.GetUploadCache()
	lock.Lock()
	defer lock.Unlock()
	// lock here to prevent data corrupt

	target, err := p.getTargetData(siteID, txn, entry, uploader)
	if err != nil {
		log.Println("error summary get target data: ", err)
		return false, err
	}

	var updateStop *stop
	for _, stop := range p.Stops {
		if target != nil {
			if len(stop.TargetCondition.FilterData([]data.IData{target}, false)) == 0 {
				continue
			}
		}
		if len(stop.SourceCriteria.FilterData([]data.IData{entry}, false)) == 0 {
			continue
		}
		updateStop = stop
	}

	if target == nil {
		switch entry.GetDataType() {
		case data.REAL_TIME:
			target = new(data.RealTimeData)
		case data.MINUTELY:
			target = new(data.MinutelyData)
		case data.HOURLY:
			target = new(data.HourlyData)
		case data.DAILY:
			target = new(data.DailyData)
		}
		target.SetStationID(entry.GetStationID())
		target.SetMonitorID(p.TargetMonitorID)
		target.SetMonitorCodeID(p.TargetMonitorCodeID)
		target.SetDataTime(entry.GetDataTime())
	}

	if p.TargetMonitorCodeID > 0 {
		mc := monitor.GetMonitorCodeByID(siteID, p.TargetMonitorCodeID)
		if mc != nil {
			target.SetMonitorID(mc.MonitorID)
			target.SetCode(mc.Code)
		}
	}

	if target.GetCode() == "" {
		target.SetCode(fmt.Sprintf("%s%d", monitor.CODE_DEFAULT, p.TargetMonitorID))
	}

	if updateStop != nil {
		if err := p.applySetters(target, updateStop.Setters); err != nil {
			log.Println("error apply setters: ", err)
			return false, err
		}
	}

	stations, exists := unuploaded[target.GetDataType()]
	if !exists {
		stations = make(map[int]map[int]map[time.Time]data.IData)
		unuploaded[target.GetDataType()] = stations
	}
	monitors, exists := stations[target.GetStationID()]
	if !exists {
		monitors = make(map[int]map[time.Time]data.IData)
		stations[target.GetStationID()] = monitors
	}
	times, exists := monitors[target.GetMonitorID()]
	if !exists {
		times = make(map[time.Time]data.IData)
		monitors[target.GetMonitorID()] = times
	}

	times[time.Time(target.GetDataTime())] = target

	return false, nil
}

func (p *summaryProcessor) getTargetData(siteID string, txn *sql.Tx, source data.IData, uploader *dataprocess.Uploader) (data.IData, error) {

	target := p.getTargetDataFromUnuploaded(siteID, source, uploader)
	if target != nil {
		return target, nil
	}

	list, err := data.GetData(siteID, source.GetDataType(), []int{source.GetStationID()}, []int{p.TargetMonitorID}, []int{p.TargetMonitorCodeID}, nil, time.Time(source.GetDataTime()), time.Time(source.GetDataTime()), nil, data.ORIGIN_DATA)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

func (p *summaryProcessor) getTargetDataFromUnuploaded(siteID string, source data.IData, uploader *dataprocess.Uploader) data.IData {

	_, unuploaded, _ := uploader.GetUploadCache()
	stations, exists := unuploaded[source.GetDataType()]
	if !exists {
		return nil
	}
	monitors, exists := stations[source.GetStationID()]
	if !exists {
		return nil
	}
	times, exists := monitors[p.TargetMonitorID]
	if !exists {
		return nil
	}
	for t, v := range times {
		if t.Equal(time.Time(source.GetDataTime())) {
			return v
		}
	}
	return nil
}

func (p *summaryProcessor) applySetters(target data.IData, setters map[string]interface{}) error {

	for field, v := range setters {
		switch field {
		case data.RTD:
			rtd, ok := target.(data.IRealTime)
			if !ok {
				continue
			}
			flt, ok := v.(float64)
			if ok {
				rtd.SetRtd(flt)
				continue
			}
			integer, ok := v.(int64)
			if ok {
				rtd.SetRtd(float64(integer))
				continue
			}
		case data.MIN:
			interval, ok := target.(data.IInterval)
			if !ok {
				continue
			}
			flt, ok := v.(float64)
			if ok {
				interval.SetMin(flt)
				continue
			}
			integer, ok := v.(int64)
			if ok {
				interval.SetMin(float64(integer))
				continue
			}
		case data.MAX:
			interval, ok := target.(data.IInterval)
			if !ok {
				continue
			}
			flt, ok := v.(float64)
			if ok {
				interval.SetMax(flt)
				continue
			}
			integer, ok := v.(int64)
			if ok {
				interval.SetMax(float64(integer))
				continue
			}
		case data.COU:
			interval, ok := target.(data.IInterval)
			if !ok {
				continue
			}
			flt, ok := v.(float64)
			if ok {
				interval.SetCou(flt)
				continue
			}
			integer, ok := v.(int64)
			if ok {
				interval.SetCou(float64(integer))
				continue
			}
		case data.AVG:
			interval, ok := target.(data.IInterval)
			if !ok {
				continue
			}
			flt, ok := v.(float64)
			if ok {
				interval.SetAvg(flt)
				continue
			}
			integer, ok := v.(int64)
			if ok {
				interval.SetAvg(float64(integer))
				continue
			}
		case data.FLAG:
			str, ok := v.(string)
			if ok {
				target.SetFlag(str)
				continue
			}
		default:
			continue
		}

		return fmt.Errorf("invalid value for setter: %s", field)
	}

	return nil
}
