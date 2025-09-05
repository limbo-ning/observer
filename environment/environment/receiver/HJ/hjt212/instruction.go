package hjt212

import (
	"errors"
	"fmt"
	"io"
	"log"
	"obsessiontech/environment/environment/data"
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
		} else {
			if _, exists := monitorGroup[k]; !exists {
				monitorGroup[k] = make(map[string]string, 0)
			}
			switch dataType {
			case data.REAL_TIME:
				monitorGroup[k][k+"-Rtd"] = v
			default:
				monitorGroup[k][k+"-Avg"] = v
				monitorGroup[k][k+"-Max"] = v
				monitorGroup[k][k+"-Min"] = v
				monitorGroup[k][k+"-Cou"] = "0"
			}
			monitorGroup[k][k+"-Flag"] = "N"
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
