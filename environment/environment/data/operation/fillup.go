package operation

import (
	"database/sql"
	"log"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
)

func init() {
	dataprocess.Register("fillup", func() dataprocess.IDataProcessor { return new(fillupProcessor) })
}

type fillupProcessor struct {
	dataprocess.BaseDataProcessor
	DataType    string        `json:"dataType"`
	Interval    time.Duration `json:"interval"`
	EndDuration time.Duration `json:"endDuration"`
	CoverExist  bool          `json:"coverExist"`
}

func (p *fillupProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) (bool, error) {
	if p.Interval <= 0 {
		return false, nil
	}

	flag, err := monitor.GetFlag(siteID, entry.GetFlag())
	if err != nil {
		return false, err
	}

	if monitor.CheckFlag(monitor.FLAG_EFFECTIVE, flag.Bits) {
		return false, nil
	}

	var interval time.Duration
	var endDuration time.Duration
	switch p.DataType {
	case data.REAL_TIME:
		if entry.GetDataType() != data.MINUTELY {
			return false, nil
		}
		interval = time.Minute * p.Interval
		endDuration = time.Minute * p.Interval
	case data.MINUTELY:
		if entry.GetDataType() != data.HOURLY {
			return false, nil
		}
		interval = time.Minute * p.Interval
		endDuration = time.Minute * p.Interval
	case data.HOURLY:
		if entry.GetDataType() != data.DAILY {
			return false, nil
		}
		interval = time.Hour * p.Interval
		endDuration = time.Hour * p.EndDuration
	}

	go func() {
		if err := p.fillup(siteID, uploader, upload, p.DataType, entry.GetStationID(), entry.GetMonitorID(), interval, time.Time(entry.GetDataTime()), time.Time(entry.GetDataTime()).Add(endDuration), entry.GetFlag()); err != nil {
			log.Println("error fillup: ", err)
		}
	}()

	return false, nil
}

func (p *fillupProcessor) fillup(siteID string, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload, dataType string, stationID, monitorID int, interval time.Duration, beginTime, endTime time.Time, flag string) error {

	dataTimes, err := p.getTargetData(siteID, dataType, stationID, monitorID, interval, beginTime, endTime, uploader, upload)
	if err != nil {
		return err
	}

	if len(dataTimes) == 0 {
		return nil
	}

	fillupd := make([]data.IData, 0)

	for _, dataTime := range dataTimes {
		beginTime := dataTime
		endTime := dataTime.Add(interval)

		log.Printf("fillup %s data: stationID[%d] monitorID[%d] begin[%v] end[%v] flag[%v]", dataType, stationID, monitorID, beginTime, endTime, flag)

		var d data.IData
		switch dataType {
		case data.REAL_TIME:
			d = &data.RealTimeData{}
		case data.MINUTELY:
			d = &data.MinutelyData{}
		case data.HOURLY:
			d = &data.HourlyData{}
		case data.DAILY:
			d = &data.DailyData{}
		}

		d.SetDataTime(util.Time(beginTime))
		d.SetStationID(stationID)
		d.SetMonitorID(monitorID)
		d.SetFlag(flag)

		fillupd = append(fillupd, d)
	}
	if err := uploader.UploadBatchData(siteID, upload, fillupd...); err != nil {
		return err
	}

	return nil
}

func (p *fillupProcessor) getTargetData(siteID, dataType string, stationID, monitorID int, interval time.Duration, beginTime, endTime time.Time, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) ([]time.Time, error) {

	checkTime := time.Time(beginTime)

	dataTimeToCheck := make(map[string]time.Time)

	for {
		if !checkTime.Before(endTime) {
			break
		}

		dataTimeToCheck[util.FormatDateTime(checkTime)] = checkTime
		checkTime = checkTime.Add(interval)
	}

	if !p.CoverExist {
		exists, err := getExistsDataTime(siteID, dataType, monitorID, stationID, beginTime, endTime)
		if err != nil {
			return nil, err
		}

		for _, t := range exists {
			delete(dataTimeToCheck, util.FormatDateTime(t))
		}
	}

	uploaderCache, _, uploaderLock := uploader.GetUploadCache()
	uploaderLock.RLock()

	dataTimes := make([]time.Time, 0)
	for _, dataTime := range dataTimeToCheck {
		if datas, exists := uploaderCache[dataType]; exists {
			if stations, exists := datas[stationID]; exists {
				if monitors, exists := stations[monitorID]; exists {
					for dt := range monitors {
						if dt.Equal(dataTime) {
							continue
						}
					}
				}
			}
		}
		dataTimes = append(dataTimes, dataTime)
	}

	uploaderLock.RUnlock()

	return dataTimes, nil

}
