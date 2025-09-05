package hjt212

import (
	"log"
	"time"
)

func init() {
	RegisterExecutor("1013", func() Executor { return new(TimeSync) })
}

type TimeSync struct{}

func (e *TimeSync) GetMN() string {
	log.Println("acknowledge should not be triggered from platform")
	return ""
}

func (e *TimeSync) Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error)) {
	request, err := input()
	if err != nil {
		close(err)
		return
	}

	log.Printf("[%s] 请求时间校准", request.MN)

	if err := output(respondUploadData(request)); err != nil {
		close(err)
		return
	}

	log.Printf("[%s] 发送校准时间戳", request.MN)
	if err := output(e.parseTimeSyncInstruction(request)); err != nil {
		close(err)
		return
	}

	process(request)
	close(nil)
}

func (e *TimeSync) parseTimeSyncInstruction(request *Instruction) *Instruction {
	i := new(Instruction)
	i.ST = request.ST
	i.MN = request.MN
	i.CN = "1012"
	i.PW = request.PW
	i.Flag = "5"
	i.QN = GenerateQN()

	cp := make(map[string]string)
	cp["SystemTime"] = time.Now().Format("20060102150405")
	i.CP = []map[string]string{cp}

	return i
}
