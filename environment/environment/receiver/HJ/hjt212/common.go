package hjt212

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	sLog "obsessiontech/environment/environment/receiver/log"
	"obsessiontech/environment/environment/receiver/upload"
)

var Config struct {
	IgnoreCRC bool
}

func init() {
	config.GetConfig("config.yaml", &Config)
}

var SERVER_ERROR = errors.New("服务器错误")

func ParseTime(timeStr string) (time.Time, error) {
	return time.ParseInLocation("20060102150405", timeStr, time.Local)
}
func GenerateQN() string {
	return strings.Replace(time.Now().Format("20060102150405.000"), ".", "", 1)
}

func NeedRespond(instruction *Instruction) bool {

	if instruction.version == "2005" {
		return true
	}

	flag, _ := strconv.Atoi(instruction.Flag)

	if flag%2 == 1 {
		return true
	}
	return false
}

var e_station_not_found = errors.New("未知监测点")
var e_monitor_inconsistent = errors.New("同数据段监测物不一致")
var e_data_field_invalid = errors.New("数据字段与数据类型不匹配")
var e_need_data_time = errors.New("缺少数据时间")

func respondUploadData(uploadData *Instruction) *Instruction {

	var resCode string

	if strings.HasPrefix(uploadData.CN, "20") {
		resCode = "9014"
	} else {
		resCode = "9013"
	}

	switch uploadData.version {
	case "2005":
		cp := make([]map[string]string, 0)
		cp = append(cp, map[string]string{
			"QN": uploadData.QN,
		}, map[string]string{
			"CN": uploadData.CN,
		})
		return &Instruction{
			QN:   "",
			ST:   "91",
			CN:   resCode,
			PW:   "",
			MN:   "",
			Flag: "",
			CP:   cp,
		}
	case "2017":
		return &Instruction{
			QN:   uploadData.QN,
			ST:   "91",
			CN:   resCode,
			PW:   uploadData.PW,
			MN:   uploadData.MN,
			Flag: "4",
			CP:   make([]map[string]string, 0),
		}
	default:
		return &Instruction{
			QN:   uploadData.QN,
			ST:   "91",
			CN:   resCode,
			PW:   uploadData.PW,
			MN:   uploadData.MN,
			Flag: "4",
			CP:   make([]map[string]string, 0),
		}
	}
}

func parseData(siteID string, uploadData *Instruction) ([]data.IData, error) {

	uploadData.data = make(map[string]string)
	result := make([]data.IData, 0)
	station := entity.GetCacheStationByMN(siteID, uploadData.MN)
	if station == nil {
		return nil, fmt.Errorf("监测点MN无匹配[%s]", uploadData.MN)
	}

	var dataInstance func() data.IData
	switch uploadData.dataType {
	case data.REAL_TIME:
		dataInstance = func() data.IData { return &data.RealTimeData{} }
	case data.MINUTELY:
		dataInstance = func() data.IData { return &data.MinutelyData{} }
	case data.HOURLY:
		dataInstance = func() data.IData { return &data.HourlyData{} }
	case data.DAILY:
		dataInstance = func() data.IData { return &data.DailyData{} }
	}

	for _, dataGroup := range uploadData.CP {
		if dataTimeStr, exists := dataGroup["DataTime"]; exists {
			t, err := ParseTime(dataTimeStr)
			if err != nil {
				return nil, err
			}
			uploadData.dataTime = &t
		} else {

			ds := make(map[string]data.IData)

			for k, v := range dataGroup {
				if k == "" {
					continue
				}

				uploadData.data[k] = v

				if parts := strings.Split(k, "-"); len(parts) == 2 {

					var d data.IData
					code := parts[0]

					field := strings.ToLower(parts[1])

					if strings.HasPrefix(field, "zs") {
						d = ds["zs"]
						if d == nil {
							d = dataInstance()
							ds["zs"] = d
						}

						code += "#zs"
						field = strings.TrimLeft(field, "zs")
					} else {
						d = ds[""]
						if d == nil {
							d = dataInstance()
							ds[""] = d
						}
					}

					if d.GetCode() != "" && d.GetCode() != code {
						return nil, fmt.Errorf("同组数据因子不一致: [%s][%s]", code, d.GetCode())
					}
					d.SetCode(code)

					if strings.ToLower(parts[1]) == "flag" {
						d.SetFlag(v)
						continue
					}

					monitorID, value, monitorCodeID, err := upload.ParseMonitorValue(siteID, station.ID, code, v)
					if err != nil {
						return nil, err
					}

					switch field {
					case "rtd":
						if realTime, ok := d.(data.IRealTime); ok {
							realTime.SetRtd(value)
						} else {
							return nil, e_data_field_invalid
						}
					case "avg":
						if interval, ok := d.(data.IInterval); ok {
							interval.SetAvg(value)
						} else {
							return nil, e_data_field_invalid
						}
					case "min":
						if interval, ok := d.(data.IInterval); ok {
							interval.SetMin(value)
						} else {
							return nil, e_data_field_invalid
						}
					case "max":
						if interval, ok := d.(data.IInterval); ok {
							interval.SetMax(value)
						} else {
							return nil, e_data_field_invalid
						}
					case "cou":
						if interval, ok := d.(data.IInterval); ok {
							interval.SetCou(value)
						} else {
							return nil, e_data_field_invalid
						}
					case "sampletime":
						t, err := ParseTime(v)
						if err != nil {
							return nil, err
						}
						d.SetDataTime(util.Time(t))
					default:
						sLog.Log(uploadData.MN, "未知数据段【%s】", parts[1])
						continue
					}

					if d.GetMonitorID() == 0 {
						d.SetMonitorID(monitorID)
					} else if d.GetMonitorID() != monitorID {
						return nil, e_monitor_inconsistent
					}

					d.SetMonitorCodeID(monitorCodeID)
				} else {
					d := ds[""]
					if d == nil {
						d = dataInstance()
						ds[""] = d
					}

					code := k

					if d.GetCode() != "" && d.GetCode() != code {
						return nil, fmt.Errorf("同组数据因子不一致: [%s][%s]", code, d.GetCode())
					}
					d.SetCode(code)

					monitorID, value, monitorCodeID, err := upload.ParseMonitorValue(siteID, station.ID, code, v)
					if err != nil {
						return nil, err
					}

					if realTime, ok := d.(data.IRealTime); ok {
						realTime.SetRtd(value)
					} else if interval, ok := d.(data.IInterval); ok {
						interval.SetAvg(value)
						interval.SetMin(value)
						interval.SetMax(value)
						interval.SetCou(value)
					}

					if d.GetMonitorID() == 0 {
						d.SetMonitorID(monitorID)
					} else if d.GetMonitorID() != monitorID {
						return nil, e_monitor_inconsistent
					}

					d.SetMonitorCodeID(monitorCodeID)
				}
			}

			for _, d := range ds {
				if d.GetMonitorID() > 0 {
					result = append(result, d)
				}
			}
		}
	}

	for _, data := range result {
		data.SetStationID(station.ID)
		if time.Time(data.GetDataTime()).IsZero() {
			if uploadData.dataTime == nil {
				return nil, e_need_data_time
			}
			data.SetDataTime(util.Time(*uploadData.dataTime))
		}
	}

	sLog.Log(uploadData.MN, "解析到[%d]组数据", len(result))

	return result, nil
}
