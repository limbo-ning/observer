package operation

import (
	"encoding/json"
	"log"

	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
)

type Upload struct {
	UploaderUID int
	Processors  dataprocess.DataProcessors
}

func (u *Upload) UploadBatchData(siteID string, uploader *dataprocess.Uploader, dataset ...data.IData) error {
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

			d, err = Modify(siteID, d, nil, u.UploaderUID)
			if err != nil {
				return err
			}
		}

		if len(u.Processors) > 0 {
			log.Println("upload processors: ", len(u.Processors))
			if err := u.Processors.Process(siteID, uploader, u, d); err != nil {
				return err
			}
		} else {
			monitorCode := monitor.GetMonitorCodeByCode(siteID, d.GetStationID(), d.GetCode())
			if monitorCode != nil {
				if err := monitorCode.Processors.Process(siteID, uploader, u, d); err != nil {
					return err
				}
			} else {
				log.Println("warn: no monitor code found: ", siteID, d.GetStationID(), d.GetCode())
			}
		}
	}

	return nil
}
