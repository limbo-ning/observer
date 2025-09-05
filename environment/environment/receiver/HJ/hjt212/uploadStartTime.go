package hjt212

import (
	"encoding/json"
	"log"
)

func init() {
	RegisterExecutor("2081", func() Executor { return new(UploadStartTime) })
}

type UploadStartTime struct{}

func (uploader *UploadStartTime) GetMN() string {
	log.Println("upload start time should not be triggered from platform")
	return ""
}

func (uploader *UploadStartTime) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload start time: ", err)
			close(SERVER_ERROR)
			return
		}
	}()

	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	parseStartTime(uploadData)

	if NeedRespond(uploadData) {
		if err := output(respondUploadData(uploadData)); err != nil {
			close(err)
			return
		}
	}

	process(uploadData)
	close(nil)
}

func parseStartTime(uploadData *Instruction) {
	output, _ := json.Marshal(uploadData.CP)
	log.Printf("upload start time MN[%s]: %s", uploadData.MN, string(output))
}
