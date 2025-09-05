package hjt212

import (
	"log"

	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	sLog "obsessiontech/environment/environment/receiver/log"
	"obsessiontech/environment/environment/receiver/upload"
)

func init() {
	RegisterExecutor("2051", func() Executor { return new(UploadMinutelyData) })
}

type UploadMinutelyData struct{}

func (uploader *UploadMinutelyData) GetMN() string {
	log.Println("UploadMinutelyData should not be triggered from platform")
	return ""
}

func (uploader *UploadMinutelyData) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload minutely data: ", err)
			close(SERVER_ERROR)
			return
		}
	}()

	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	uploadData.dataType = data.MINUTELY
	datas, err := parseData(siteID, uploadData)
	if err != nil {
		sLog.Log(uploadData.MN, "数据错误: %s", err.Error())
		log.Println("process minutely data error:", err)
		close(err)
		return
	}

	uper := new(dataprocess.Uploader)
	if err := uper.UploadBatchData(siteID, upload.ReceiverUpload, datas...); err != nil {
		sLog.Log(uploadData.MN, "上传错误: %s", err.Error())
		close(err)
		return
	}

	if err := uper.UploadUnuploaded(siteID, upload.ReceiverUpload); err != nil {
		sLog.Log(uploadData.MN, "上传错误: %s", err.Error())
		close(err)
		return
	}

	if NeedRespond(uploadData) {
		if err := output(respondUploadData(uploadData)); err != nil {
			close(err)
		}
	}

	process(uploadData)
	close(nil)
}
