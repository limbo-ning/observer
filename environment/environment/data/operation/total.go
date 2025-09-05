package operation

import (
	"database/sql"
	"log"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"time"
)

func init() {
	dataprocess.Register("total", func() dataprocess.IDataProcessor { return new(totalProcessor) })
}

type totalProcessor struct {
	dataprocess.BaseDataProcessor
	Adders   []*adder `json:"adders"`
	DataType []string `json:"dataTypes"`
}

type adder struct {
	MonitorID  int     `json:"monitorID"`
	Multiplier float64 `json:"multiplier"`
}

func (p *totalProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) (bool, error) {
	log.Println("run total: ", entry.GetDataType(), entry.GetMonitorID(), entry.GetStationID(), entry.GetDataTime())

	dtChecked := false
	for _, dt := range p.DataType {
		if dt == entry.GetDataType() {
			dtChecked = true
			continue
		}
	}

	if !dtChecked {
		return false, nil
	}

	_, _, lock := uploader.GetUploadCache()
	lock.Lock()
	defer lock.Unlock()

	sources, err := p.getSourceData(siteID, txn, entry, uploader)
	if err != nil {
		log.Println("error total get target data: ", err)
		return false, err
	}

	var total float64

	for _, a := range p.Adders {
		log.Println("check source: ", a.MonitorID)
		d := sources[a.MonitorID]
		if d == nil {
			log.Println("no source found: ", a.MonitorID)
			continue
		}

		var part float64
		if rtd, ok := d.(data.IRealTime); ok {
			part = rtd.GetRtd()
		} else if interval, ok := d.(data.IInterval); ok {
			part = interval.GetAvg()
		} else {
			log.Println("unknown source dataType: ", d.GetDataType())
			continue
		}

		accuracy := util.GetAccuracy(part) + util.GetAccuracy(a.Multiplier)
		total += util.ApplyAccuracy(part*a.Multiplier, accuracy)
	}

	log.Println("total calculated:", total)

	if rtd, ok := entry.(data.IRealTime); ok {
		rtd.SetRtd(total)
	} else if interval, ok := entry.(data.IInterval); ok {
		interval.SetAvg(total)
	}

	return false, nil
}

func (p *totalProcessor) getSourceData(siteID string, txn *sql.Tx, source data.IData, uploader *dataprocess.Uploader) (map[int]data.IData, error) {

	sources := p.getSourcesFromUploader(siteID, source, uploader)
	if sources == nil {
		sources = make(map[int]data.IData)
	}

	mids := make([]int, 0)
	for _, a := range p.Adders {
		if _, exists := sources[a.MonitorID]; !exists {
			mids = append(mids, a.MonitorID)
		}
	}

	if len(mids) == 0 {
		log.Println("all sources loaded from uploader: ", len(sources), sources)
		return sources, nil
	}

	list, err := data.GetData(siteID, source.GetDataType(), []int{source.GetStationID()}, mids, nil, nil, time.Time(source.GetDataTime()), time.Time(source.GetDataTime()), nil)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return sources, nil
	}

	for _, d := range list {
		sources[d.GetMonitorID()] = d
	}

	return sources, nil
}

func (p *totalProcessor) getSourcesFromUploader(siteID string, source data.IData, uploader *dataprocess.Uploader) map[int]data.IData {

	uploaded, unuploaded, _ := uploader.GetUploadCache()

	result := make(map[int]data.IData)

	fromUploaded := p.getSourcesFromPool(source, uploaded)
	log.Println("total sources from uploaded: ", len(fromUploaded))
	for k, v := range fromUploaded {
		result[k] = v
	}

	fromUnuploaded := p.getSourcesFromPool(source, unuploaded)
	log.Println("total sources from unuploaded: ", len(unuploaded))
	for k, v := range fromUnuploaded {
		result[k] = v
	}

	return result
}

func (p *totalProcessor) getSourcesFromPool(source data.IData, pool map[string]map[int]map[int]map[time.Time]data.IData) map[int]data.IData {

	result := make(map[int]data.IData)

	stations, exists := pool[source.GetDataType()]
	if !exists {
		return nil
	}
	monitors, exists := stations[source.GetStationID()]
	if !exists {
		return nil
	}

	for _, a := range p.Adders {
		times, exists := monitors[a.MonitorID]
		if !exists {
			continue
		}
		for t, v := range times {
			if t.Equal(time.Time(source.GetDataTime())) {
				result[v.GetMonitorID()] = v
			}
		}
	}
	return result
}
