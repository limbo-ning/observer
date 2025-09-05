package noise

import (
	"errors"
	"log"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	sLog "obsessiontech/environment/environment/receiver/log"
	"obsessiontech/environment/environment/receiver/noise/instruction"
	"obsessiontech/environment/environment/receiver/upload"
)

var e_mn_not_found = errors.New("MN号未激活")
var e_monitor_not_found = errors.New("因子未注册")

func (p *Noise) uploadData(i *instruction.Instruction) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload noise data: ", err)
			return
		}
	}()

	dataList, err := p.parseUploadData(i)
	if err != nil {
		log.Println("process upload fume noise error:", err)
		return err
	}
	uper := new(dataprocess.Uploader)
	if err := uper.UploadBatchData(p.SiteID, upload.ReceiverUpload, dataList...); err != nil {
		sLog.Log(p.MN, "上传错误: %s", err.Error())
		return err
	}

	if err := uper.UploadUnuploaded(p.SiteID, upload.ReceiverUpload); err != nil {
		sLog.Log(p.MN, "上传错误: %s", err.Error())
		return err
	}

	return nil
}

func (p *Noise) parseUploadData(i *instruction.Instruction) ([]data.IData, error) {
	i.DataType = data.REAL_TIME
	result := make([]data.IData, 0)

	station := p.GetStation()

	for k, v := range i.Data {
		monitorID, value, monitorCodeID, err := upload.ParseMonitorValue(p.SiteID, station.ID, k, v)
		if err != nil {
			log.Println("error parse value: ", k, v, err)
			continue
		}

		rtd := new(data.RealTimeData)
		rtd.DataTime = util.Time(*i.DateTime)
		rtd.StationID = station.ID
		rtd.MonitorID = monitorID
		rtd.Rtd = value

		rtd.SetMonitorCodeID(monitorCodeID)
		rtd.SetCode(k)

		result = append(result, rtd)
	}

	return result, nil
}
