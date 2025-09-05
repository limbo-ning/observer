package externalsource

import (
	"encoding/json"
	"log"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/data/operation"
	"obsessiontech/environment/environment/data/recent"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
)

type uploader struct {
	UploaderUID int
}

func (u *uploader) UploadBatchData(siteID string, uploader *dataprocess.Uploader, dataset ...data.IData) error {
	if len(dataset) == 0 {
		return nil
	}

	for _, d := range dataset {

		if err := data.Add(siteID, d); err != nil {
			if err != data.E_data_exists {
				dataToPrint, _ := json.Marshal(d)
				log.Println("error save data: ", string(dataToPrint), err)
				return err
			}

			d, err = operation.Modify(siteID, d, nil, u.UploaderUID)
			if err != nil {
				return err
			}
		}

		monitorCode := monitor.GetMonitorCodeByCode(siteID, d.GetStationID(), d.GetCode())
		if monitorCode != nil {
			if err := monitorCode.Processors.Process(siteID, uploader, u, d); err != nil {
				return err
			}
		}

		go recent.UpdateRecentData(siteID, d)
	}

	return nil
}
