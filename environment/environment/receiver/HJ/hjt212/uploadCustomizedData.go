package hjt212

import (
	"fmt"
	"log"
	"obsessiontech/environment/environment/dataprocess"
	sLog "obsessiontech/environment/environment/receiver/log"
	"obsessiontech/environment/environment/receiver/upload"
	"strings"
)

func init() {
	RegisterExecutor("_upload_data", func() Executor { return new(UploadCustomizedData) })
}

type UploadCustomizedData struct {
	DataType    string            `json:"dataType"`
	CodeMapping map[string]string `json:"codeMapping"`
}

func (uploader *UploadCustomizedData) GetMN() string {
	log.Println("UploadDailyData should not be triggered from platform")
	return ""
}

func (uploader *UploadCustomizedData) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("error process upload custom data: ", err)
			close(SERVER_ERROR)
			return
		}
	}()

	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	uploadData.dataType = uploader.DataType

	translated := make([]map[string]string, 0)
	for _, dataGroup := range uploadData.CP {
		group := make(map[string]string)
		for prev, value := range dataGroup {
			if parts := strings.Split(prev, "-"); len(parts) == 2 {
				if to, exists := uploader.CodeMapping[parts[0]]; exists {
					group[fmt.Sprintf("%s-%s", to, parts[1])] = value
				} else {
					group[prev] = value
				}
			} else {
				group[prev] = value
			}

		}
		translated = append(translated, group)
	}

	uploadData.CP = translated

	datas, err := parseData(siteID, uploadData)
	if err != nil {
		sLog.Log(uploadData.MN, "数据错误: %s", err.Error())
		log.Println("process custom data error:", err)
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
			return
		}
	}

	process(uploadData)
	close(nil)
}
