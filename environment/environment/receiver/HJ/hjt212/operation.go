package hjt212

import (
	"encoding/json"
	"log"
)

func init() {
	RegisterExecutor("1011", func() Executor { return new(Operation) })
	RegisterExecutor("1012", func() Executor { return new(Operation) })
	RegisterExecutor("1013", func() Executor { return new(Operation) })
	RegisterExecutor("1014", func() Executor { return new(Operation) })
	RegisterExecutor("1061", func() Executor { return new(Operation) })
	RegisterExecutor("1063", func() Executor { return new(Operation) })

	RegisterExecutor("3011", func() Executor { return new(Operation) })
	RegisterExecutor("3012", func() Executor { return new(Operation) })
	RegisterExecutor("3013", func() Executor { return new(Operation) })
	RegisterExecutor("3014", func() Executor { return new(Operation) })
	RegisterExecutor("3015", func() Executor { return new(Operation) })
	RegisterExecutor("3016", func() Executor { return new(Operation) })
	RegisterExecutor("3017", func() Executor { return new(Operation) })
	RegisterExecutor("3018", func() Executor { return new(Operation) })
	RegisterExecutor("3019", func() Executor { return new(Operation) })
	RegisterExecutor("3020", func() Executor { return new(Operation) })
	RegisterExecutor("3021", func() Executor { return new(Operation) })
}

type Operation struct{}

func (uploader *Operation) GetMN() string {
	log.Println("Operation should not be triggered from platform")
	return ""
}

func (uploader *Operation) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {
	uploadData, err := input()
	if err != nil {
		close(err)
		return
	}

	cp, _ := json.Marshal(uploadData.CP)
	log.Printf("控制命令 MN[%s]: CN[%s] CP[%s]", uploadData.MN, uploadData.CN, string(cp))

	if NeedRespond(uploadData) {
		if err := output(respondUploadData(uploadData)); err != nil {
			close(err)
			return
		}
	}

	process(uploadData)
	close(nil)
}
