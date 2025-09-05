package hjt212

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
)

func ValidateDatagram(data string) (result string, err error) {

	defer func() {
		if e := recover(); e != nil {
			log.Println("validate datagram panic: ", e)
			debug.PrintStack()
			result = ""
			err = errors.New("校验失败:" + data)
		}
	}()

	if len(data) < 10 {
		return "", e_incomplete
	}

	head := data[0:2]
	err = validateHead(head)
	if err != nil {
		return "", err
	}

	lengthField := data[2:6]
	bodyLen, err := validateLength(lengthField)
	if err != nil {
		return "", err
	}

	if len(data) < 10+bodyLen {
		return "", e_incomplete
	}

	bodyField := data[6 : 6+bodyLen]
	result = bodyField

	if !Config.IgnoreCRC {
		crcField := data[6+bodyLen : 6+bodyLen+4]
		err = validateCrc(crcField, bodyField)
		if err != nil {
			return "", err
		}
	}

	end := data[6+bodyLen+4:]
	validateEnd(end)

	return result, err
}

var e_invalid_header = errors.New("报文起始不正确")
var e_invalid_length = errors.New("报文长度不正确")
var e_incomplete = errors.New("报文未完结")
var e_invalid_crc = errors.New("CRC校验不正确")

func validateHead(head string) error {
	if head != "##" {
		return e_invalid_header
	}
	return nil
}

func validateLength(length string) (int, error) {
	len, err := strconv.Atoi(length)

	if err != nil {
		return -1, e_invalid_length
	}

	return len, nil
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

func validateCrc(crcField, bodyField string) error {
	crc := calculateCrc([]byte(bodyField))
	crcFieldNum, _ := strconv.ParseUint(crcField, 16, 16)

	if uint16(crcFieldNum) != crc {
		return e_invalid_crc
	}
	return nil
}

func validateEnd(end string) {

}

func PackDatagram(bodyField string) string {
	crc := calculateCrc([]byte(bodyField))
	crcField := strings.Replace(strings.ToUpper(fmt.Sprintf("%4x", crc)), " ", "0", -1)

	length := len(bodyField)
	lengthField := fmt.Sprintf("%04d", length)

	return fmt.Sprintf("##%s%s%s\r\n", lengthField, bodyField, crcField)
}
