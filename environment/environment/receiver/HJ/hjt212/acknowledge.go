package hjt212

import (
	"encoding/json"
	"log"
)

func init() {
	RegisterExecutor("9011", func() Executor { return new(Acknowledge) })
	RegisterExecutor("9012", func() Executor { return new(Acknowledge) })
	RegisterExecutor("9013", func() Executor { return new(Acknowledge) })
	RegisterExecutor("9014", func() Executor { return new(Acknowledge) })
}

type Acknowledge struct{}

func (uploader *Acknowledge) GetMN() string {
	log.Println("acknowledge should not be triggered from platform")
	return ""
}

func (uploader *Acknowledge) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {
	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	cp, _ := json.Marshal(uploadData.CP)
	log.Printf("应答 MN[%s]: CN[%s] CP[%s]", uploadData.MN, uploadData.CN, string(cp))

	process(uploadData)
	close(nil)
}
