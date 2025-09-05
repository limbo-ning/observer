package hjt212

import (
	"log"

	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	sLog "obsessiontech/environment/environment/receiver/log"
	"obsessiontech/environment/environment/receiver/upload"
)

func init() {
	RegisterExecutor("2011", func() Executor { return new(UploadRealTimeData) })
	RegisterExecutor("2082", func() Executor { return new(UploadRealTimeData) })
	RegisterExecutor("5902", func() Executor { return new(UploadRealTimeData) })
}

type UploadRealTimeData struct{}

func (uploader *UploadRealTimeData) GetMN() string {
	log.Println("uploadRealTimeData should not be triggered from platform")
	return ""
}

func (uploader *UploadRealTimeData) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload real time data: ", err)
			sLog.Log(uploader.GetMN(), "发生错误: %+v", err)
			close(SERVER_ERROR)
			return
		}
	}()

	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	uper := new(dataprocess.Uploader)

	worker := func() error {
		uploadData.dataType = data.REAL_TIME
		datas, err := parseData(siteID, uploadData)
		if err != nil {
			sLog.Log(uploadData.MN, "数据错误: %s", err.Error())
			log.Println("process real time data error:", err)
			return err
		}

		if err := uper.UploadBatchData(siteID, upload.ReceiverUpload, datas...); err != nil {
			sLog.Log(uploadData.MN, "上传错误: %s", err.Error())
			return err
		}

		process(uploadData)

		return nil
	}

	if uploadData.PNUM == 0 {
		if err := worker(); err != nil {
			close(err)
			return
		}
	} else {
		first := uploadData

		sLog.Log(uploadData.MN, "处理分包: %s(%d/%d)", uploadData.QN, uploadData.PNO, uploadData.PNUM)

		for uploadData.PNO < uploadData.PNUM {
			if err := worker(); err != nil {
				close(err)
				return
			}
			uploadData, err = input()
			if err != nil {
				close(err)
				return
			}

			uploadData.dataTime = first.dataTime
			sLog.Log(uploadData.MN, "处理分包: %s(%d/%d)", uploadData.QN, uploadData.PNO, uploadData.PNUM)
		}

		if err := worker(); err != nil {
			close(err)
			return
		}
	}

	if err := uper.UploadUnuploaded(siteID, upload.ReceiverUpload); err != nil {
		sLog.Log(uploadData.MN, "上传错误: %s", err.Error())
		close(err)
		return
	}

	if NeedRespond(uploadData) {
		if err := output(respondUploadData(uploadData)); err != nil {
			close(err)
			return
		}
	}

	close(nil)
}
