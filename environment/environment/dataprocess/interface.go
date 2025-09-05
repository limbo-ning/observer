package dataprocess

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/environment/data"
)

var registry = make(map[string]func() IDataProcessor)

func Register(rule string, fac func() IDataProcessor) {
	if _, exists := registry[rule]; exists {
		log.Panic("duplicate registry: " + rule)
	}
	registry[rule] = fac
}

type IDataUpload interface {
	UploadBatchData(siteID string, uploader *Uploader, dataset ...data.IData) error
}

type Uploader struct {
	UploadedCache   map[string]map[int]map[int]map[time.Time]data.IData
	UnUploadedCache map[string]map[int]map[int]map[time.Time]data.IData
	UploadCacheLock sync.RWMutex
}

func (u *Uploader) GetUploadCache() (map[string]map[int]map[int]map[time.Time]data.IData, map[string]map[int]map[int]map[time.Time]data.IData, *sync.RWMutex) {
	if u.UploadedCache == nil {
		u.UploadedCache = make(map[string]map[int]map[int]map[time.Time]data.IData)
	}
	if u.UnUploadedCache == nil {
		u.UnUploadedCache = make(map[string]map[int]map[int]map[time.Time]data.IData)
	}
	return u.UploadedCache, u.UnUploadedCache, &u.UploadCacheLock
}

func (u *Uploader) UploadBatchData(siteID string, upload IDataUpload, dataset ...data.IData) error {

	uploaded, _, uploadCacheLock := u.GetUploadCache()
	uploadCacheLock.Lock()

	for _, d := range dataset {
		stations, exists := uploaded[d.GetDataType()]
		if !exists {
			stations = make(map[int]map[int]map[time.Time]data.IData)
			uploaded[d.GetDataType()] = stations
		}
		monitors, exists := stations[d.GetStationID()]
		if !exists {
			monitors = make(map[int]map[time.Time]data.IData)
			stations[d.GetStationID()] = monitors
		}
		datas, exists := monitors[d.GetMonitorID()]
		if !exists {
			datas = make(map[time.Time]data.IData)
			monitors[d.GetMonitorID()] = datas
		}
		datas[time.Time(d.GetDataTime())] = d
	}
	uploadCacheLock.Unlock()

	if err := upload.UploadBatchData(siteID, u, dataset...); err != nil {
		return err
	}

	return nil
}

func (u *Uploader) UploadUnuploaded(siteID string, uploader IDataUpload) error {

	for {
		_, unuploaded, uploadCacheLock := u.GetUploadCache()

		uploadCacheLock.Lock()

		datas := make([]data.IData, 0)
		dts := make([]string, 0)
		for dt, stations := range unuploaded {
			for _, monitors := range stations {
				for _, times := range monitors {
					for _, d := range times {
						datas = append(datas, d)
					}
				}
			}
			dts = append(dts, dt)
		}
		log.Println("unuploaded count: ", len(datas))

		for _, dt := range dts {
			delete(unuploaded, dt)
		}

		uploadCacheLock.Unlock()

		if len(datas) == 0 {
			return nil
		}

		if err := u.UploadBatchData(siteID, uploader, datas...); err != nil {
			return err
		}
	}
}

type IDataProcessor interface {
	GetRule() string
	ProcessData(siteID string, txn *sql.Tx, d data.IData, uploader *Uploader, upload IDataUpload) (interrupt bool, err error)
}

type BaseDataProcessor struct {
	Rule string `json:"rule"`
}

func (r *BaseDataProcessor) GetRule() string { return r.Rule }

type DataProcessors []IDataProcessor

func (processors *DataProcessors) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	list := make([]IDataProcessor, 0)

	for _, r := range raw {
		var a BaseDataProcessor
		json.Unmarshal(r, &a)

		fac := registry[a.Rule]
		if fac == nil {
			return fmt.Errorf("dataProcess rule not exists:%s", a.Rule)
		}
		instance := fac()
		if err := json.Unmarshal([]byte(r), instance); err != nil {
			log.Println("error unmarsahl actions: ", a.Rule, instance, err)
			return err
		}

		list = append(list, instance)
	}

	*processors = list

	return nil
}

func (processors *DataProcessors) Process(siteID string, uploader *Uploader, upload IDataUpload, datas ...data.IData) error {

	return datasource.Txn(func(txn *sql.Tx) {
		for _, d := range datas {
			for _, p := range *processors {
				interrupt, err := p.ProcessData(siteID, txn, d, uploader, upload)
				if err != nil {
					panic(err)
				}
				if interrupt {
					break
				}
			}

			if err := data.UpdateWithTxn(siteID, txn, d); err != nil {
				panic(err)
			}
		}
	})
}
