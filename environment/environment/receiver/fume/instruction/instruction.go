package instruction

import (
	"errors"
	"log"
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

	if strings.HasPrefix(datagram, "##") {
		if !strings.HasSuffix(datagram, "\r\n") {
			return nil, E_incomplete
		}
		datagram = strings.TrimRight(datagram[2:], "\r\n")
	} else {
		log.Println("ERROR datagram head is not ##: ", datagram)
		return nil, e_wrong_datagram
	}

	var i Instruction

	main := strings.Split(datagram, "&&")
	if len(main) != 3 {
		log.Println("ERROR datagram missing parts: ", datagram)
		return nil, e_wrong_datagram
	}

	mn, dataTime, err := parseDateTimeAndMN(strings.Split(main[0], ";"))
	if err != nil {
		return nil, err
	}

	i.MN = mn
	i.DateTime = dataTime

	data := strings.Split(main[1], ";")
	i.Data = make(map[string]string)

	for _, d := range data {
		k, v, err := parseKV(d)
		if err != nil {
			return nil, err
		}

		i.Data[k] = v
	}

	crcField := main[2]
	err = validateCrc(crcField, datagram[:len(datagram)-len(crcField)])
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func parseDateTimeAndMN(fields []string) (mn string, dateTime *time.Time, e error) {
	if len(fields) != 2 {
		log.Println("ERROR datagram missing mn and datetime")
		return "", nil, e_wrong_datagram
	}

	mnK, mnV, err := parseKV(fields[0])
	if err != nil {
		e = err
		return
	}
	if mnK != "MN" {
		log.Println("ERROR datagram cannot find mn in place. find instead: ", mnK)
		e = e_wrong_datagram
		return
	}

	mn = mnV

	dateTimeK, dateTimeV, err := parseKV(fields[1])
	if err != nil {
		e = err
		return
	}
	if dateTimeK != "DataTime" {
		log.Println("ERROR datagram cannot find dateTime in place. find instead: ", dateTimeK)
		e = e_wrong_datagram
		return
	}

	t, err := time.ParseInLocation("20060102150405", dateTimeV, time.Local)
	if err != nil {
		log.Println("ERROR datagram cannot parse date time: ", dateTimeV)
		e = e_wrong_datagram
		return
	}

	dateTime = &t

	return
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
	crcFieldNum, _ := strconv.ParseUint(crcField, 16, 16)

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
