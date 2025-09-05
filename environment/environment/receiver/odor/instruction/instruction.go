package instruction

import (
	"bufio"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type Instruction struct {
	MN       string
	DataType string
	DateTime time.Time
	Data     map[string]string
}

var e_no_stationid = errors.New("没有stationid")
var e_invalid_datetime = errors.New("报文时间错误")

func Parse(datagram string) (*Instruction, error) {

	if methodIndex := strings.Index(datagram, "POST"); methodIndex > 0 {
		datagram = datagram[methodIndex:]
	}

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(datagram)))
	if err != nil {
		log.Println("error read request: ", err)
		return nil, err
	}

	if err := req.ParseForm(); err != nil {
		log.Println("error pase form: ", err)
		return nil, err
	}

	values := req.PostForm

	result := new(Instruction)
	result.Data = make(map[string]string)

	var dateStr, timeStr string

	for k, v := range values {
		switch strings.ToLower(k) {
		case "stationid":
			result.MN = v[0]
		case "date":
			dateStr = v[0]
		case "time":
			timeStr = v[0]
		default:
			result.Data[k] = v[0]
		}
	}

	if result.MN == "" {
		return nil, e_no_stationid
	}

	if dateStr == "" || timeStr == "" {
		return nil, e_invalid_datetime
	}

	result.DateTime, err = time.ParseInLocation("200601021504", dateStr+timeStr, time.Local)
	if err != nil {
		return nil, e_invalid_datetime
	}

	return result, nil
}
