package instruction

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

type Instruction struct {
	version  string
	dataTime *time.Time
	dataType string
	data     map[string]string
	QN       string
	PNUM     int
	PNO      int
	ST       string
	CN       string
	PW       string
	MN       string
	Flag     string
	CP       []map[string]string
}

var e_invalid_instruction = errors.New("错误的指令")

func decomposeCP(reader *strings.Reader) ([]map[string]string, error) {
	cp := make([]map[string]string, 0)

	count := 0
	var key, value string
	readingKey := true
	cpGroup := make(map[string]string)
	for {
		ch, _, err := reader.ReadRune()
		if err != nil {
			log.Println("error decomposing CP: ", count, err)
			return nil, e_invalid_instruction
		}
		switch ch {
		case '&':
			count++
			if count == 4 {
				cpGroup[key] = value
				cp = append(cp, cpGroup)
				return cp, nil
			}
		case ',':
			readingKey = true
			cpGroup[key] = value
			key = ""
			value = ""
		case ';':
			readingKey = true
			cpGroup[key] = value
			key = ""
			value = ""
			cp = append(cp, cpGroup)
			cpGroup = make(map[string]string)
		case '=':
			readingKey = false
		default:
			if readingKey {
				key += string(ch)
			} else {
				value += string(ch)
			}
		}
	}
}

func DecomposeInstruction(bodyString string) (*Instruction, error) {

	var result Instruction

	reader := strings.NewReader(bodyString)

	var key, value string
	readingKey := true
	for {
		ch, _, err := reader.ReadRune()
		if err == io.EOF || ch == ';' {
			readingKey = true
			switch key {
			case "QN":
				result.QN = value
			case "PNUM":
				result.PNUM, _ = strconv.Atoi(value)
			case "PNO":
				result.PNO, _ = strconv.Atoi(value)
			case "ST":
				result.ST = value
			case "CN":
				result.CN = value
			case "PW":
				result.PW = value
			case "MN":
				result.MN = value
			case "Flag":
				result.Flag = value
			case "CP":
			default:
				log.Println("Unknown field", key, value)
			}
			key = ""
			value = ""
		} else if ch == '=' {
			readingKey = false
			if key == "CP" {
				result.CP, err = decomposeCP(reader)
				if err != nil {
					return nil, err
				}
			}
		} else {
			if readingKey {
				key += string(ch)
			} else {
				value += string(ch)
			}
		}

		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("error decomposing instruction: ", err)
			return nil, e_invalid_instruction
		}
	}

	return &result, nil
}

func composeCPGroup(dataType string, dataTime *time.Time, datas map[string]string) []map[string]string {
	result := make([]map[string]string, 0)

	if dataTime != nil {
		result = append(result, map[string]string{
			"DataTime": dataTime.Format("20060102150405"),
		})
	}

	monitorGroup := make(map[string]map[string]string)

	for k, v := range datas {
		if parts := strings.Split(k, "-"); len(parts) == 2 {
			if _, exists := monitorGroup[parts[0]]; !exists {
				monitorGroup[parts[0]] = make(map[string]string, 0)
			}
			monitorGroup[parts[0]][k] = v
		}
	}

	for _, sets := range monitorGroup {
		result = append(result, sets)
	}

	return result
}

func composeCP(cp []map[string]string) string {

	pairs := make([]string, 0)

	for _, cpGroup := range cp {
		groupPairs := make([]string, 0)
		for k, v := range cpGroup {
			if v == "" {
				continue
			}
			groupPairs = append(groupPairs, fmt.Sprintf("%s=%v", k, v))
		}
		pairs = append(pairs, strings.Join(groupPairs, ","))
	}

	return fmt.Sprintf("&&%s&&", strings.Join(pairs, ";"))
}

func ComposeInstruction(input *Instruction) string {

	values := make([]string, 0)

	if input.QN != "" {
		values = append(values, fmt.Sprintf("QN=%s", input.QN))
	}
	if input.ST != "" {
		values = append(values, fmt.Sprintf("ST=%s", input.ST))
	}
	if input.CN != "" {
		values = append(values, fmt.Sprintf("CN=%s", input.CN))
	}
	if input.PW != "" {
		values = append(values, fmt.Sprintf("PW=%s", input.PW))
	}
	if input.MN != "" {
		values = append(values, fmt.Sprintf("MN=%s", input.MN))
	}
	if input.Flag != "" {
		values = append(values, fmt.Sprintf("Flag=%s", input.Flag))
	}
	if input.PNO > 0 {
		values = append(values, fmt.Sprintf("PNO=%d", input.PNO))
	}
	if input.PNUM > 0 {
		values = append(values, fmt.Sprintf("PNUM=%d", input.PNUM))
	}

	cp := composeCP(input.CP)
	values = append(values, fmt.Sprintf("CP=%s", cp))

	return strings.Join(values, ";")
}

func PackDatagram(bodyField string) string {
	crc := calculateCrc([]byte(bodyField))
	crcField := strings.Replace(strings.ToUpper(fmt.Sprintf("%4x", crc)), " ", "0", -1)

	length := len(bodyField)
	lengthField := fmt.Sprintf("%04d", length)

	return fmt.Sprintf("##%s%s%s\r\n", lengthField, bodyField, crcField)
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

var e_invalid_header = errors.New("报文起始不正确")
var e_invalid_length = errors.New("报文长度不正确")
var e_incomplete = errors.New("报文未完结")
var e_invalid_crc = errors.New("CRC校验不正确")

func validateCrc(crcField, bodyField string) error {
	crc := calculateCrc([]byte(bodyField))
	crcFieldNum, _ := strconv.ParseUint(crcField, 16, 16)

	if uint16(crcFieldNum) != crc {
		return e_invalid_crc
	}
	return nil
}
