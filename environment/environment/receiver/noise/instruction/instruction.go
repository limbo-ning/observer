package instruction

import (
	"errors"
	"log"
	"obsessiontech/common/util"
	"strconv"
	"strings"
	"time"
)

type Instruction struct {
	MN       string
	DataType string
	DateTime *time.Time
	Data     map[string]string
}

var E_incomplete = errors.New("报文不完整")
var e_wrong_datagram = errors.New("报文内容错误")
var e_invalid_crc = errors.New("报文CRC校验不正确")

func Parse(datagram string) (*Instruction, error) {

	if !strings.HasPrefix(datagram, "##") {
		return nil, errors.New("报文起始不正确")
	}

	datagram = strings.TrimPrefix(datagram, "##")

	parts := strings.Split(datagram, "&")
	if len(parts) != 3 {
		log.Println("ERROR datagram missing parts: ", datagram)
		return nil, e_wrong_datagram
	}

	bodyLength, err := getBodyLength(parts[0])
	if err != nil {
		return nil, err
	}

	if len(parts[1])+2 != bodyLength {
		return nil, E_incomplete
	}

	if err := validateCrc(strings.TrimRight(parts[2], "\r\n"), "&"+parts[1]+"&"); err != nil {
		return nil, err
	}

	return parseInstruction(parts[1])
}

func getBodyLength(bodyField string) (int, error) {
	return strconv.Atoi(bodyField)
}

func parseInstruction(body string) (*Instruction, error) {

	result := new(Instruction)

	result.Data = make(map[string]string)

	parts := strings.Split(body, ",")
	for _, p := range parts {
		k, v, err := parseKV(p)
		if err != nil {
			return nil, err
		}

		switch k {
		case "MN":
			result.MN = v
		case "QN":
			t, err := util.ParseDateTimeWithFormat(v[:14], "YYYYMMDDhhmmss")
			if err != nil {
				return nil, err
			}

			result.DateTime = &t
		default:
			result.Data[k] = v
		}
	}

	return result, nil
}

func parseKV(field string) (k, v string, err error) {
	parts := strings.Split(field, "=")
	if len(parts) != 2 {
		log.Println("ERROR datagram not in k=v form:", field)
		return "", "", e_wrong_datagram
	}
	return parts[0], parts[1], nil
}

func validateCrc(crcField, bodyField string) error {
	crc := calculateCrc([]byte(bodyField))
	crcFieldNum, err := strconv.ParseUint(crcField, 16, 16)

	if err != nil {
		return err
	}

	log.Printf("calculate crc [%s] of [%s]: [%d] crcFieldNumber[%d]", crcField, bodyField, crc, crcFieldNum)

	if uint16(crcFieldNum) != crc {
		return e_invalid_crc
	}
	return nil
}

func calculateCrc(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, byteData := range data {
		crc = crc>>8 ^ uint16(byteData)
		for i := 0; i < 8; i++ {
			flag := crc & 0x01
			crc = crc >> 1

			if flag == 1 {
				crc ^= 0xA001
			}
		}
	}

	return crc
}
