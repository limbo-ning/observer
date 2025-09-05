package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"regexp"
	"strconv"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"

	"github.com/tealeg/xlsx/v3"
)

var e_invalid_exceluploader = errors.New("excel上传设置无效")

var numericRegexp *regexp.Regexp

func init() {
	numericRegexp = regexp.MustCompile("\\d*(\\.\\d*)?")
}

type ExcelUploader struct {
	Sheet         []int                `json:"sheet"`
	DataType      string               `json:"dataType"`
	EntityConfig  *FieldConfig         `json:"entityConfig,omitempty"`
	StationConfig *FieldConfig         `json:"stationConfig,omitempty"`
	TimeConfig    *DataTimeConfig      `json:"timeConfig"`
	DataConfig    []*MonitorDataConfig `json:"dataConfig"`
}

const (
	layout_column_entry = "entry"
	layout_row_entry    = "row_entry"
	layout_input        = "input"
	layout_id           = "id"
	layout_offset       = "offset"
)

type FieldConfig struct {
	Layout string `json:"layout"`
	Index  int    `json:"index"`
	ID     int    `json:"id"`
	Value  string `json:"value"`
}

func (f *FieldConfig) getFieldValue(row *xlsx.Row, options ...bool) (interface{}, error) {

	var cell *xlsx.Cell

	switch f.Layout {
	case layout_column_entry:
		cell = row.GetCell(f.Index)
	case layout_offset:
		cell = row.GetCell(f.Index + f.ID)
	case layout_input:
		return f.Value, nil
	case layout_id:
		return nil, nil
	default:
		return nil, e_invalid_exceluploader
	}

	if cell != nil {

		log.Println("cell: ", cell.Type(), cell.Value)

		if cell.IsTime() {
			t, err := cell.GetTime(cell.Row.Sheet.File.Date1904)
			if err != nil {
				return nil, err
			}

			Y, M, D := t.Date()
			h, m, s := t.Clock()

			localTime := time.Date(Y, M, D, h, m, s, 0, time.Local)

			return localTime, nil
		}

		switch cell.Type() {
		case xlsx.CellTypeNumeric:

			if len(options) > 0 && options[0] {
				t, err := cell.GetTime(cell.Row.Sheet.File.Date1904)
				if err != nil {
					return nil, err
				}

				Y, M, D := t.Date()
				h, m, s := t.Clock()

				localTime := time.Date(Y, M, D, h, m, s, 0, time.Local)

				return localTime, nil
			}

			return strconv.ParseFloat(cell.Value, 64)
		case xlsx.CellTypeBool:
			return strconv.ParseBool(cell.Value)
		case xlsx.CellTypeDate:
			t, err := cell.GetTime(cell.Row.Sheet.File.Date1904)
			if err != nil {
				return nil, err
			}

			Y, M, D := t.Date()
			h, m, s := t.Clock()

			localTime := time.Date(Y, M, D, h, m, s, 0, time.Local)

			return localTime, nil
		}

		return cell.FormattedValue()
	}

	return nil, e_invalid_exceluploader
}

func (f *FieldConfig) getFieldFloatValue(row *xlsx.Row) (float64, error) {
	value, err := f.getFieldValue(row)
	if err != nil {
		return 0, err
	}

	if str, ok := value.(string); ok {
		fltPart := numericRegexp.FindString(str)
		if fltPart == "" {
			return 0, fmt.Errorf("无效的数值【%s】", str)
		}

		flt, err := strconv.ParseFloat(fltPart, 64)
		if err != nil {
			return 0, err
		}
		return flt, nil
	} else if flt, ok := value.(float64); ok {
		return flt, nil
	} else {
		return 0, fmt.Errorf("无效的数值【%v】", value)
	}
}

type DataTimeConfig struct {
	DateConfig     *DateTimeFieldConfig `json:"dateConfig,omitempty"`
	TimeConfig     *DateTimeFieldConfig `json:"timeConfig,omitempty"`
	DateTimeConfig *DateTimeFieldConfig `json:"dateTimeConfig,omitempty"`
}

type DateTimeFieldConfig struct {
	FieldConfig
	Format string `json:"format"`
}

type MonitorDataConfig struct {
	MonitorConfig  *FieldConfig `json:"monitorConfig"`
	RtdConfig      *FieldConfig `json:"rtdConfig,omitempty"`
	AvgConfig      *FieldConfig `json:"avgConfig,omitempty"`
	MinConfig      *FieldConfig `json:"minConfig,omitempty"`
	MaxConfig      *FieldConfig `json:"maxConfig,omitempty"`
	CouConfig      *FieldConfig `json:"couConfig,omitempty"`
	FlagConfig     *FieldConfig `json:"flagConfig,omitempty"`
	monitorEntries map[int]string
}

func (m *MonitorDataConfig) parseMonitors(sheet *xlsx.Sheet) error {

	m.monitorEntries = make(map[int]string)

	switch m.MonitorConfig.Layout {
	case layout_row_entry:
		row, err := sheet.Row(m.MonitorConfig.Index)
		if err != nil {
			return err
		}
		if err := row.ForEachCell(func(cell *xlsx.Cell) error {

			code, err := cell.FormattedValue()
			if err != nil {
				return err
			}

			x, _ := cell.GetCoordinates()
			m.monitorEntries[x] = code

			return nil
		}, xlsx.SkipEmptyCells); err != nil {
			return err
		}
	}

	return nil
}

func (m *MonitorDataConfig) parseData(siteID string, row *xlsx.Row, stationID int, dataTime *util.Time, fac func() data.IData) ([]data.IData, error) {

	codes, err := monitor.GetMonitorCodes(siteID, -1, "", 0, stationID)
	if err != nil {
		return nil, err
	}

	monitorIDSet := make(map[int]bool)
	codeMaps := make(map[string]int)
	monitorIDs := make([]int, 0)
	for _, c := range codes {
		if c.StationID == stationID {
			if !monitorIDSet[c.MonitorID] {
				monitorIDSet[c.MonitorID] = true
				monitorIDs = append(monitorIDs, c.MonitorID)
			}
			codeMaps[c.Code] = c.MonitorID
		}
	}

	for _, c := range codes {
		if c.StationID == 0 {
			if monitorIDSet[c.MonitorID] {
				codeMaps[c.Code] = c.MonitorID
			}
		}
	}

	monitorList, err := monitor.GetMonitors(siteID, nil, nil, monitorIDs...)
	if err != nil {
		return nil, err
	}

	nameMaps := make(map[string]int)
	for _, c := range monitorList {
		nameMaps[c.Name] = c.ID
	}

	result := make([]data.IData, 0)
	switch m.MonitorConfig.Layout {
	case layout_row_entry:
		for cellIndex, code := range m.monitorEntries {

			monitorCode := ""

			mid, exists := codeMaps[code]
			monitorCode = code
			if !exists {
				mid, exists = nameMaps[code]
				monitorCode = fmt.Sprintf("DEFAULT%d", mid)
			}
			if !exists {
				log.Printf("error upload monitor code not exists: column[%d] code[%s] stationID[%d]", cellIndex, code, stationID)
				continue
			}
			d := fac()

			if d == nil {
				return nil, errors.New("invalid data type")
			}

			d.SetStationID(stationID)
			d.SetDataTime(*dataTime)
			d.SetMonitorID(mid)
			d.SetCode(monitorCode)

			if rtd, ok := d.(data.IRealTime); ok {
				if m.RtdConfig != nil && m.RtdConfig.Layout == layout_offset {
					fieldConfig := new(FieldConfig)
					fieldConfig.Layout = m.RtdConfig.Layout
					fieldConfig.Index = m.RtdConfig.Index
					fieldConfig.ID = cellIndex
					flt, err := fieldConfig.getFieldFloatValue(row)
					if err != nil {
						log.Println("error parse flt value: ", err)
						continue
					}
					rtd.SetRtd(flt)
				}
			} else if interval, ok := d.(data.IInterval); ok {
				if m.AvgConfig != nil && m.AvgConfig.Layout == layout_offset {
					fieldConfig := new(FieldConfig)
					fieldConfig.Layout = m.AvgConfig.Layout
					fieldConfig.Index = m.AvgConfig.Index
					fieldConfig.ID = cellIndex
					flt, err := fieldConfig.getFieldFloatValue(row)
					if err != nil {
						log.Println("error parse flt value: ", err)
						continue
					}
					interval.SetAvg(flt)
				}
				if m.MinConfig != nil && m.MinConfig.Layout == layout_offset {
					fieldConfig := new(FieldConfig)
					fieldConfig.Layout = m.MinConfig.Layout
					fieldConfig.Index = m.MinConfig.Index
					fieldConfig.ID = cellIndex
					flt, err := fieldConfig.getFieldFloatValue(row)
					if err != nil {
						log.Println("error parse flt value: ", err)
						continue
					}
					interval.SetMin(flt)
				}
				if m.MaxConfig != nil && m.MaxConfig.Layout == layout_offset {
					fieldConfig := new(FieldConfig)
					fieldConfig.Layout = m.MaxConfig.Layout
					fieldConfig.Index = m.MaxConfig.Index
					fieldConfig.ID = cellIndex
					flt, err := fieldConfig.getFieldFloatValue(row)
					if err != nil {
						log.Println("error parse flt value: ", err)
						continue
					}
					interval.SetMax(flt)
				}
				if m.CouConfig != nil && m.CouConfig.Layout == layout_offset {
					fieldConfig := new(FieldConfig)
					fieldConfig.Layout = m.CouConfig.Layout
					fieldConfig.Index = m.CouConfig.Index
					fieldConfig.ID = cellIndex
					flt, err := fieldConfig.getFieldFloatValue(row)
					if err != nil {
						log.Println("error parse flt value: ", err)
						continue
					}
					interval.SetCou(flt)
				}
			}

			if m.FlagConfig != nil && m.FlagConfig.Layout == layout_offset {
				fieldConfig := new(FieldConfig)
				fieldConfig.Layout = m.CouConfig.Layout
				fieldConfig.Index = m.CouConfig.Index
				fieldConfig.ID = cellIndex
				value, err := m.FlagConfig.getFieldValue(row)
				if err != nil {
					log.Println("error parse flag value: ", err)
					continue
				}
				if str, ok := value.(string); ok {
					d.SetFlag(str)
				} else {
					log.Println("error parse flag value: ", value)
					continue
				}
			}

			result = append(result, d)
		}
	case layout_column_entry:
		codeValue, err := m.MonitorConfig.getFieldValue(row)
		if err != nil {
			log.Println("error get monitor code: ", err)
			return result, nil
		}

		code, ok := codeValue.(string)
		if !ok {
			log.Println("error get monitor code: code not string")
			return result, nil
		}

		mid, exists := codeMaps[code]
		if !exists {
			mid, exists = nameMaps[code]
		}
		if !exists {
			log.Println("error upload monitor code not exists: ", code, stationID)
			return result, nil
		}
		d := fac()

		if d == nil {
			return nil, errors.New("invalid data type")
		}

		d.SetStationID(stationID)
		d.SetDataTime(*dataTime)
		d.SetMonitorID(mid)

		if rtd, ok := d.(data.IRealTime); ok {
			if m.RtdConfig != nil {
				flt, err := m.RtdConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				rtd.SetRtd(flt)
			}
		} else if interval, ok := d.(data.IInterval); ok {
			if m.AvgConfig != nil {
				flt, err := m.AvgConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetAvg(flt)
			}
			if m.MinConfig != nil {
				flt, err := m.MinConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetMin(flt)
			}
			if m.MaxConfig != nil {
				flt, err := m.MaxConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetMax(flt)
			}
			if m.CouConfig != nil {
				flt, err := m.CouConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetCou(flt)
			}
		}

		if m.FlagConfig != nil {
			value, err := m.FlagConfig.getFieldValue(row)
			if err != nil {
				log.Println("error parse flag value: ", err)
				return result, nil
			}
			if str, ok := value.(string); ok {
				d.SetFlag(str)
			} else {
				log.Println("error parse flag value: ", value)
				return result, nil
			}
		}
		result = append(result, d)
	case layout_id:
		d := fac()
		if d == nil {
			return nil, errors.New("invalid data type")
		}

		d.SetStationID(stationID)
		d.SetDataTime(*dataTime)
		d.SetMonitorID(m.MonitorConfig.ID)

		if rtd, ok := d.(data.IRealTime); ok {
			if m.RtdConfig != nil {
				flt, err := m.RtdConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				rtd.SetRtd(flt)
			}
		} else if interval, ok := d.(data.IInterval); ok {
			if m.AvgConfig != nil {
				flt, err := m.AvgConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetAvg(flt)
			}
			if m.MinConfig != nil {
				flt, err := m.MinConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetMin(flt)
			}
			if m.MaxConfig != nil {
				flt, err := m.MaxConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetMax(flt)
			}
			if m.CouConfig != nil {
				flt, err := m.CouConfig.getFieldFloatValue(row)
				if err != nil {
					log.Println("error parse flt value: ", err)
					return result, nil
				}
				interval.SetCou(flt)
			}
		}

		if m.FlagConfig != nil {
			value, err := m.FlagConfig.getFieldValue(row)
			if err != nil {
				log.Println("error parse flag value: ", err)
				return result, nil
			}
			if str, ok := value.(string); ok {
				d.SetFlag(str)
			} else {
				log.Println("error parse flag value: ", value)
				return result, nil
			}
		}
		result = append(result, d)
	}

	return result, nil
}

func (u *ExcelUploader) Validate() error {
	return nil
}

func UploadExcel(siteID string, actionAuth authority.ActionAuthSet, files map[string][]*multipart.FileHeader, params map[string][]string) ([]*data.TimeData, error) {
	uploaderStrs, exists := params["uploader"]
	if !exists {
		return nil, errors.New("需要uploader")
	}
	uploaderStr := uploaderStrs[0]

	uploader := new(ExcelUploader)
	if err := json.Unmarshal([]byte(uploaderStr), &uploader); err != nil {
		return nil, err
	}

	excelFiles, exists := files["excel"]
	if !exists {
		return nil, errors.New("需要上传excel文件")
	}
	excelFile := excelFiles[0]

	f, err := excelFile.Open()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return uploadExcel(siteID, actionAuth, uploader, excelFile.Filename, data)
}

func uploadExcel(siteID string, actionAuth authority.ActionAuthSet, uploader *ExcelUploader, excelFileName string, excelFileData []byte) ([]*data.TimeData, error) {

	if err := uploader.Validate(); err != nil {
		return nil, err
	}

	excel, err := xlsx.OpenBinary(excelFileData)
	if err != nil {
		log.Println("error open excel file: ", excelFileName, err)
		return nil, err
	}

	filtered := make([]entity.IEntityStationAuth, 0)

	if uploader.Sheet != nil {
		for _, sheetIndex := range uploader.Sheet {
			sheet := excel.Sheets[sheetIndex]

			list, err := processSheet(siteID, uploader, sheet)
			if err != nil {
				return nil, err
			}
			filtered = append(filtered, list...)
		}
	} else {
		log.Println("error upload no sheet")
		return nil, e_invalid_exceluploader
	}

	result := make([]*data.TimeData, 0)

	filtered, err = entity.FilterEntityStationAuthInterface(siteID, filtered, actionAuth, entity.ACTION_ENTITY_EDIT)
	if err != nil {
		return nil, err
	}

	dataTimeMapping := make(map[string]*data.TimeData)

	for _, ele := range filtered {

		d := ele.(data.IData)

		dataTimeStr := util.FormatDateTime(time.Time(d.GetDataTime()))

		timeData, exists := dataTimeMapping[dataTimeStr]
		if !exists {
			timeData = new(data.TimeData)
			timeData.DataTime = d.GetDataTime()
			timeData.Data = make(map[int][]data.IData)
			dataTimeMapping[dataTimeStr] = timeData
		}

		if _, exists := timeData.Data[d.GetStationID()]; !exists {
			timeData.Data[d.GetStationID()] = make([]data.IData, 0)
		}

		timeData.Data[d.GetStationID()] = append(timeData.Data[d.GetStationID()], d)
	}

	for _, d := range dataTimeMapping {
		result = append(result, d)
	}

	return result, nil
}

func processSheet(siteID string, uploader *ExcelUploader, sheet *xlsx.Sheet) ([]entity.IEntityStationAuth, error) {

	for _, monitorConfig := range uploader.DataConfig {
		if err := monitorConfig.parseMonitors(sheet); err != nil {
			return nil, err
		}
	}

	uploadedDataList := make([]entity.IEntityStationAuth, 0)

	if err := sheet.ForEachRow(func(row *xlsx.Row) error {

		stationID, err := getStationID(siteID, uploader.EntityConfig, uploader.StationConfig, row)
		if err != nil {
			log.Println("error upload get stationID: ", err)
			return nil
		}

		dataTime, err := getDataTime(siteID, uploader.TimeConfig, row)
		if err != nil {
			log.Println("error upload get dataTime: ", err)
			return nil
		}

		log.Println("upload process row data time: ", dataTime)

		for _, monitorConfig := range uploader.DataConfig {

			dataList, err := monitorConfig.parseData(siteID, row, stationID, dataTime, func() data.IData {
				switch uploader.DataType {
				case data.REAL_TIME:
					return &data.RealTimeData{}
				case data.MINUTELY:
					return &data.MinutelyData{}
				case data.HOURLY:
					return &data.HourlyData{}
				case data.DAILY:
					return &data.DailyData{}
				default:
					return nil
				}
			})
			if err != nil {
				log.Println("error upload parse data: ", err)
				return err
			}
			for _, d := range dataList {
				uploadedDataList = append(uploadedDataList, d)
			}
		}

		return nil
	}, xlsx.SkipEmptyRows); err != nil {
		return nil, err
	}

	return uploadedDataList, nil
}

func getStationID(siteID string, entityConfig *FieldConfig, stationConfig *FieldConfig, row *xlsx.Row) (int, error) {

	var entityID int

	if entityConfig != nil {
		switch entityConfig.Layout {
		case layout_id:
			entityID = entityConfig.ID
		default:
			value, err := entityConfig.getFieldValue(row)
			if err != nil {
				return 0, err
			}
			entityName, ok := value.(string)
			if !ok {
				return 0, errors.New("单元格内容不是文字类型")
			}

			entities, err := entity.GetEntityList(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, "", "", nil, nil, entityName)
			if err != nil {
				return 0, err
			}
			for _, e := range entities {
				if e.Name == entityName {
					entityID = e.ID
				}
			}

			if entityID <= 0 {
				return 0, fmt.Errorf("找不到企业[%s]", entityName)
			}
		}
	}

	if stationConfig != nil {
		switch stationConfig.Layout {
		case layout_id:
			return stationConfig.ID, nil
		}

		value, err := stationConfig.getFieldValue(row)
		if err != nil {
			return 0, err
		}
		stationName, ok := value.(string)
		if !ok {
			return 0, errors.New("单元格内容不是文字类型")
		}

		stations, err := entity.GetStations(siteID, nil, []int{entityID}, "", "", stationName)
		if err != nil {
			return 0, err
		}
		for _, s := range stations {
			if s.Name == stationName {
				return s.ID, nil
			}
		}

		return 0, fmt.Errorf("找不到站点[%s]", stationName)
	}

	if entityID > 0 {
		stations, err := entity.GetStations(siteID, nil, []int{entityID}, "", "", "")
		if err != nil {
			return 0, err
		}
		if len(stations) != 1 {
			return 0, fmt.Errorf("无法确定企业的监测点[%d]", entityID)
		}
		return stations[0].ID, nil
	}

	return 0, e_invalid_exceluploader
}

func getDataTime(siteID string, dataTimeConfig *DataTimeConfig, row *xlsx.Row) (*util.Time, error) {

	var result util.Time

	if dataTimeConfig.DateTimeConfig != nil {

		value, err := dataTimeConfig.DateTimeConfig.getFieldValue(row, true)
		if err != nil {
			return nil, err
		}
		if t, ok := value.(time.Time); ok {
			result = util.Time(t)
		} else if str, ok := value.(string); ok {
			t, err := util.ParseDateTimeWithFormat(str, dataTimeConfig.DateTimeConfig.Format)
			if err != nil {
				return nil, err
			}

			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year()-t.Year(), 0, 0)
			}

			result = util.Time(t)
		}

		log.Println("datetime config: ", value, result)

	} else {
		resultT := time.Date(0, time.January, 1, 0, 0, 0, 0, time.Local)

		if dataTimeConfig.DateConfig != nil {
			value, err := dataTimeConfig.DateConfig.getFieldValue(row, true)
			if err != nil {
				return nil, err
			}

			if t, ok := value.(time.Time); ok {
				log.Println("date config is time: ", value, t)
				resultT = resultT.AddDate(t.Year()-resultT.Year(), 0, t.YearDay()-resultT.YearDay())
			} else if str, ok := value.(string); ok {
				resultT, err = util.ParseDateWithFormat(str, dataTimeConfig.DateConfig.Format)
				if err != nil {
					return nil, err
				}
				if resultT.Year() == 0 {
					resultT = t.AddDate(time.Now().Year()-resultT.Year(), 0, 0)
				}
			}

			log.Println("date config: ", value, resultT)
		}

		if dataTimeConfig.TimeConfig != nil {
			value, err := dataTimeConfig.TimeConfig.getFieldValue(row, true)
			if err != nil {
				return nil, err
			}
			if t, ok := value.(time.Time); ok {
				log.Println("time config is time: ", value, t)
				tDate := util.GetDate(t)
				resultT = resultT.Add(t.Sub(tDate))
			} else if str, ok := value.(string); ok {
				t, err = util.ParseTimeWithFormat(str, dataTimeConfig.TimeConfig.Format)
				if err != nil {
					return nil, err
				}
				tDate := util.GetDate(t)
				resultT = resultT.Add(t.Sub(tDate))
			}

			log.Println("time config: ", value, resultT)
		}

		result = util.Time(resultT)
	}

	return &result, nil
}
