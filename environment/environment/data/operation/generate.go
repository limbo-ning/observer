package operation

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/stats"
)

func init() {
	dataprocess.Register("generate", func() dataprocess.IDataProcessor { return new(generatorProcessor) })
}

type generatorProcessor struct {
	dataprocess.BaseDataProcessor
	DataType       string        `json:"dataType"`
	Interval       time.Duration `json:"interval"`
	TracebackCount int           `json:"tracebackCount"`
	CoverExist     bool          `json:"coverExist"`
}

func (p *generatorProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload) (bool, error) {
	if p.Interval <= 0 {
		return false, nil
	}

	var interval time.Duration
	var dataTime time.Time
	switch p.DataType {
	case data.MINUTELY:
		if entry.GetDataType() != data.REAL_TIME {
			return false, nil
		}
		interval = time.Minute * p.Interval
		dataTime = time.Time(entry.GetDataTime())
		dataTime = dataTime.Add(-1 * time.Minute * time.Duration(dataTime.Minute()%int(p.Interval))).Truncate(time.Minute)
	case data.HOURLY:
		if entry.GetDataType() != data.MINUTELY {
			return false, nil
		}
		interval = time.Hour * p.Interval
		dataTime = time.Time(entry.GetDataTime())
		dataTime = dataTime.Add(-1 * time.Hour * time.Duration(dataTime.Hour()%int(p.Interval))).Truncate(time.Hour)
	case data.DAILY:
		if entry.GetDataType() != data.HOURLY {
			return false, nil
		}
		interval = time.Hour * 24 * p.Interval
		dataTime = time.Time(entry.GetDataTime())
		dataTime = dataTime.Add(-1 * time.Hour * time.Duration(dataTime.Hour()%int(24*p.Interval))).Truncate(time.Hour)
	}

	go func() {
		if err := generate(siteID, uploader, upload, p.DataType, entry.GetStationID(), entry.GetMonitorID(), entry.GetCode(), interval, dataTime, p.CoverExist, p.TracebackCount); err != nil {
			log.Println("error generate: ", err)
		}
	}()

	return false, nil
}

func generate(siteID string, uploader *dataprocess.Uploader, upload dataprocess.IDataUpload, dataType string, stationID, monitorID int, code string, interval time.Duration, dataTime time.Time, replaceExists bool, tracebackCount int) error {

	dataTimes, err := getTargetData(siteID, dataType, stationID, monitorID, interval, dataTime, uploader, replaceExists, tracebackCount)
	if err != nil {
		return err
	}

	if len(dataTimes) == 0 {
		return nil
	}

	generated := make([]data.IData, 0)

	for _, dataTime := range dataTimes {
		d, err := generateTargetData(siteID, dataType, monitorID, stationID, code, dataTime, dataTime.Add(interval))
		if err != nil {
			continue
		}
		generated = append(generated, d)
	}
	if err := uploader.UploadBatchData(siteID, upload, generated...); err != nil {
		return err
	}

	return nil
}

func getTargetData(siteID, dataType string, stationID, monitorID int, interval time.Duration, dataTime time.Time, uploader *dataprocess.Uploader, replaceExists bool, tracebackCount int) ([]time.Time, error) {

	beginTime := time.Time(dataTime)

	dataTimeToCheck := make(map[string]time.Time)

	if tracebackCount < 0 {
		dataTimeToCheck[util.FormatDateTime(beginTime)] = beginTime
	} else {
		for i := 0; i <= tracebackCount; i++ {
			t := beginTime.Add(-1 * interval)
			dataTimeToCheck[util.FormatDateTime(t)] = t
			beginTime = t
		}
	}

	if !replaceExists {
		exists, err := getExistsDataTime(siteID, dataType, monitorID, stationID, beginTime, dataTime)
		if err != nil {
			return nil, err
		}

		for _, t := range exists {
			delete(dataTimeToCheck, util.FormatDateTime(t))
		}
	}

	uploaderCache, _, uploaderLock := uploader.GetUploadCache()
	uploaderLock.RLock()

	dataTimes := make([]time.Time, 0)
	for _, dataTime := range dataTimeToCheck {
		if datas, exists := uploaderCache[dataType]; exists {
			if stations, exists := datas[stationID]; exists {
				if monitors, exists := stations[monitorID]; exists {
					for dt := range monitors {
						if dt.Equal(dataTime) {
							continue
						}
					}
				}
			}
		}
		dataTimes = append(dataTimes, dataTime)
	}

	uploaderLock.RUnlock()

	return dataTimes, nil

}

func getExistsDataTime(siteID, dataType string, monitorID, stationID int, beginTime, endTime time.Time) ([]time.Time, error) {

	result := make([]time.Time, 0)
	tables := data.FetchTableNames(siteID, dataType, beginTime, endTime)

	for _, table := range tables {
		rows, err := datasource.GetConn().Query(fmt.Sprintf(`
			SELECT
				DISTINCT %s
			FROM
				%s
			WHERE
				%s >= ? AND %s <= ? AND %s = ? AND %s = ?
		`, data.DATA_TIME, table, data.DATA_TIME, data.DATA_TIME, data.STATION_ID, data.MONITOR_ID), beginTime, endTime, stationID, monitorID)

		if err != nil {
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var t time.Time
			rows.Scan(&t)
			result = append(result, t)
		}
	}

	return result, nil
}

func generateTargetData(siteID, dataType string, monitorID, stationID int, code string, beginTime, endTime time.Time) (data.IData, error) {

	log.Printf("generate %s data: stationID[%d] monitorID[%d] code[%s] begin[%v] end[%v]", dataType, stationID, monitorID, code, beginTime, endTime)

	monitorModule, err := monitor.GetModule(siteID)
	if err != nil {
		return nil, err
	}

	effectFlags := make(map[string]byte)

	for _, f := range monitorModule.Flags {
		if monitor.CheckFlag(monitor.FLAG_EFFECTIVE, f.Bits) {
			effectFlags[f.Flag] = 1
		}
	}

	var fetchDataType string
	var table string
	var column string
	var avg, min, max, cou, total, effectiveTotal float64
	min = math.MaxFloat64
	var count, effectiveCount, accuracy int

	switch dataType {
	case data.MINUTELY:
		fetchDataType = data.REAL_TIME
		table = data.TableName(siteID, data.REAL_TIME)
		column = data.RTD
	case data.HOURLY:
		fetchDataType = data.MINUTELY
		table = data.TableName(siteID, data.MINUTELY)
		column = fmt.Sprintf("%s,%s,%s,%s", data.AVG, data.MIN, data.MAX, data.COU)
	case data.DAILY:
		fetchDataType = data.HOURLY
		table = data.TableName(siteID, data.HOURLY)
		column = fmt.Sprintf("%s,%s,%s,%s", data.AVG, data.MIN, data.MAX, data.COU)
	default:
		return nil, errors.New("unsupported data type to generate")
	}

	column += "," + data.FLAG

	ineffectFlagCount := make(map[string]int)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s
		WHERE
			%s = ? AND %s = ? AND %s >= ? AND %s < ?
	`, column, table, data.STATION_ID, data.MONITOR_ID, data.DATA_TIME, data.DATA_TIME)

	values := []interface{}{stationID, monitorID, beginTime, endTime}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var toCount float64
		var flag string

		count++

		switch dataType {
		case data.MINUTELY:
			var rtd float64
			rows.Scan(&rtd, &flag)

			total += rtd
			if _, exists := effectFlags[flag]; exists {
				effectiveTotal += rtd
				if rtd < min {
					min = rtd
				}
				if rtd > max {
					max = rtd
				}
				toCount = rtd
			} else {
				ineffectFlagCount[flag] += 1
			}

		case data.HOURLY:
			fallthrough
		case data.DAILY:
			var perAvg, perMin, perMax, perCou float64
			if err := rows.Scan(&perAvg, &perMin, &perMax, &perCou, &flag); err != nil {
				log.Println("error scan: ", column, table, err)
				return nil, err
			}
			total += perAvg
			if _, exists := effectFlags[flag]; exists {
				effectiveTotal += perAvg
				if perMin < min {
					min = perMin
				}
				if perMax > max {
					max = perMax
				}
				toCount = perAvg
				cou += perCou

			} else {
				ineffectFlagCount[flag] += 1
			}
		}

		if _, exists := effectFlags[flag]; exists {
			effectiveCount++

			toCountStr := fmt.Sprintf("%v", toCount)
			if strings.Contains(toCountStr, ".") {
				parts := strings.Split(toCountStr, ".")
				if len(parts) == 2 && len(parts[1]) > accuracy {
					accuracy = len(parts[1])
				}
			}
		}
	}

	if count == 0 {
		log.Printf("无生成数据 stationID[%d] dataType[%s] monitorID[%d] dataTime[%v-%v]", stationID, dataType, monitorID, beginTime, endTime)
		return nil, errors.New("无数据生成")
	}

	accuracy++

	var result data.IData
	switch dataType {
	case data.MINUTELY:
		result = &data.MinutelyData{}
	case data.HOURLY:
		result = &data.HourlyData{}
	case data.DAILY:
		result = &data.DailyData{}
	}

	result.SetDataTime(util.Time(beginTime))
	result.SetStationID(stationID)
	result.SetMonitorID(monitorID)
	result.SetCode(code)

	if threshold, exists := monitorModule.EffectiveIntervalThreshold[dataType]; exists {

		slots := stats.CountSlots(fetchDataType, &beginTime, &endTime)

		if float64(effectiveCount)/float64(slots) < threshold {

			var highestIneffectFlag string
			var highestIneffrectCount int

			for f, count := range ineffectFlagCount {
				if count > highestIneffrectCount {
					highestIneffectFlag = f
					highestIneffrectCount = count
				}
			}
			log.Printf("时段有效数据不足 stationID[%d] dataType[%s] monitorID[%d] dataTime[%v-%v] effectCount[%d / %d] flag[%s]", stationID, fetchDataType, monitorID, beginTime, endTime, effectiveCount, slots, highestIneffectFlag)

			if err := monitor.ChangeFlag(siteID, result, highestIneffectFlag, -1); err != nil {
				return nil, err
			}
			return result, nil
		}

		if effectiveCount > 0 {
			raw := effectiveTotal / float64(effectiveCount)
			avg = math.Round(raw*math.Pow10(accuracy)) / math.Pow10(accuracy)

			result.(data.IInterval).SetAvg(avg)
			result.(data.IInterval).SetMax(max)
			result.(data.IInterval).SetMin(min)
			result.(data.IInterval).SetCou(cou)
		}
	} else {
		raw := total / float64(count)
		avg = math.Round(raw*math.Pow10(accuracy)) / math.Pow10(accuracy)

		result.(data.IInterval).SetAvg(avg)
		result.(data.IInterval).SetMax(max)
		result.(data.IInterval).SetMin(min)
		result.(data.IInterval).SetCou(cou)
	}

	m := monitor.GetMonitor(siteID, monitorID)
	if m != nil {
		if monitor.CheckFlag(monitor.MONITOR_SWITCH, m.Type) {
			interval, ok := result.(data.IInterval)
			if ok {
				interval.SetAvg(math.Ceil(interval.GetAvg()))
				interval.SetMax(math.Ceil(interval.GetMax()))
				interval.SetMin(math.Ceil(interval.GetMin()))
				interval.SetCou(math.Ceil(interval.GetCou()))
			}
		}
	}

	return result, nil
}
