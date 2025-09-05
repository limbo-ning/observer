package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/receiver/ipchandler"
)

var e_save_batch = errors.New("批量数据保存失败")

type upload struct{}

func (u *upload) UploadBatchData(siteID string, uploader *dataprocess.Uploader, dataset ...data.IData) error {
	if len(dataset) == 0 {
		return nil
	}
	hasErr := false

	for _, d := range dataset {
		if err := data.AddUpdate(siteID, d); err != nil {
			dataToPrint, _ := json.Marshal(d)
			log.Println("error save data: ", string(dataToPrint), err)
			hasErr = true
			continue
		}

		monitorCode := monitor.GetMonitorCodeByCode(siteID, d.GetStationID(), d.GetCode())
		if monitorCode != nil {
			if err := monitorCode.Processors.Process(siteID, uploader, u, d); err != nil {
				dataToPrint, _ := json.Marshal(d)
				log.Println("error process data: ", string(dataToPrint), err)
				hasErr = true
				continue
			}
		} else {
			log.Println("no monitor code: ", siteID, d.GetStationID(), d.GetMonitorID(), d.GetCode())
		}
	}

	if hasErr {
		return e_save_batch
	}

	for _, d := range dataset {
		ipchandler.ReportData(d)
	}

	return nil
}

var ReceiverUpload = new(upload)

func ParseMonitorValue(siteID string, stationID int, code, value string) (int, float64, int, error) {

	monitorCode := monitor.GetMonitorCodeByCode(siteID, stationID, code)

	if monitorCode == nil {
		return 0, 0, 0, fmt.Errorf("因子未知[%s]", code)
	}

	flt, err := strconv.ParseFloat(strings.Trim(value, "\r\n "), 64)
	if err != nil {
		log.Printf("error parse float: value[%s] err[%v]", value, err)
		return 0, 0, 0, fmt.Errorf("数值无效[%s]", value)
	}
	return monitorCode.MonitorID, flt, monitorCode.ID, nil

}
