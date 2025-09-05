package hjt212

import (
	"encoding/json"
	"log"
)

func init() {
	RegisterExecutor("2021", func() Executor { return new(UploadDeviceStatus) })
}

type UploadDeviceStatus struct{}

func (uploader *UploadDeviceStatus) GetMN() string {
	log.Println("UploadDeviceStatus should not be triggered from platform")
	return ""
}

func (uploader *UploadDeviceStatus) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload device status: ", err)
			close(SERVER_ERROR)
			return
		}
	}()

	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	parseDeviceStatus(uploadData)

	if NeedRespond(uploadData) {
		if err := output(respondUploadData(uploadData)); err != nil {
			close(err)
			return
		}
	}

	process(uploadData)
	close(nil)
}

func parseDeviceStatus(uploadData *Instruction) {
	output, _ := json.Marshal(uploadData.CP)
	log.Printf("upload device status MN[%s]: %s", uploadData.MN, string(output))
}
