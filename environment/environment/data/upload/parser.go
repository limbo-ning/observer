package upload

import (
	"errors"
	"fmt"
	"obsessiontech/common/excel"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
	"strings"
	"time"
)

type ExcelParser struct {
	SiteID string
	Sheets []*SheetParserConfig
}

func (p *ExcelParser) GetSheetParsers() ([]excel.ISheetParser, error) {
	result := make([]excel.ISheetParser, 0)
	for _, config := range p.Sheets {
		parsers, err := config.GetParsers(p.SiteID)
		if err != nil {
			return nil, err
		}
		result = append(result, parsers...)
	}
	return result, nil
}

type SheetParserConfig struct {
	SheetIndices  []int         `json:"sheetIndices"`
	DataType      string        `json:"dataType"`
	EntityConfig  *EntityField  `json:"entityConfig"`
	StationConfig *StationField `json:"stationConfig"`
	TimeConfig    *TimeField    `json:"timeConfig"`
	DataConfig    *DataField    `json:"dataConfig"`
	ContentRange  [2]int        `json:"contentRange"`
}
type EntityField struct {
	EntityID    int `json:"entityID,omitempty"`
	Index       int `json:"index,omitempty"`
	HeaderIndex int `json:"headerIndex,omitempty"`
}
type StationField struct {
	StationID   int `json:"stationID,omitempty"`
	Index       int `json:"index,omitempty"`
	HeaderIndex int `json:"headerIndex,omitempty"`
}
type TimeField struct {
	Indices []int          `json:"indices,omitempty"`
	Formats map[int]string `json:"Formats,omitempty"`
}
type DataField struct {
	Fields             []string       `json:"fields"`
	FieldIndices       map[string]int `json:"fieldIndices"`
	MonitorHeaderIndex int            `json:"monitorHeaderIndex"`
	MonitorIndex       int            `json:"monitorIndex"`
}

type SheetParser struct {
	*SheetParserConfig
	SheetIndex int
	SiteID     string
}

func (s *SheetParserConfig) GetParsers(siteID string) ([]excel.ISheetParser, error) {
	result := make([]excel.ISheetParser, 0)

	if len(s.SheetIndices) == 0 {
		p := new(SheetParser)
		p.SheetIndex = -1
		p.SheetParserConfig = s
		p.SiteID = siteID
		result = append(result, p)
	} else {
		for _, i := range s.SheetIndices {
			p := new(SheetParser)
			p.SheetParserConfig = s
			p.SheetIndex = i
			p.SiteID = siteID
			result = append(result, p)
		}
	}

	return result, nil
}

func (p *SheetParser) GetSheetDirection() excel.SheetDirection {
	return excel.SheetDirectionRow
}
func (p *SheetParser) GetSheetIndex() int {
	return p.SheetIndex
}
func (p *SheetParser) UnMarshalEntry(raw map[string]any) (any, error) {
	var result []data.IData

	// mapping := make(map[int]map[int][]data.IData)
	// switch p.DataType {
	// case data.REAL_TIME:
	// case data.MINUTELY:
	// case data.HOURLY:
	// case data.DAILY:
	// }

	// for k, v := range raw {
	// }

	return result, nil
}

func (p *SheetParser) GetSheetRows() (int, int) {
	return p.ContentRange[0], p.ContentRange[1]
}
func (p *SheetParser) GetSheetHeaderRowIndex() []int {
	result := make([]int, 0)
	if p.EntityConfig.HeaderIndex > 0 {
		result = append(result, p.EntityConfig.HeaderIndex)
	}
	if p.StationConfig.HeaderIndex > 0 {
		result = append(result, p.StationConfig.HeaderIndex)
	}
	return result
}

type EntityScanner struct {
	SiteID   string
	ColIndex int
}

func (s *EntityScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {
	c, err := r.Get(s.ColIndex)
	if err != nil {
		return err
	}
	name, err := c.GetString()
	if err != nil {
		return err
	}
	entityList, err := entity.GetEntityList(s.SiteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, "", "", nil, nil, name)
	if err != nil {
		return err
	}

	for _, e := range entityList {
		if e.Name == name {
			(*entry)["entityID"] = e.ID
			return nil
		}
	}

	return nil
}

type StationScanner struct {
	SiteID   string
	EntityID int
	ColIndex int
}

func (s *StationScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {

	var entityID int
	if s.EntityID > 0 {
		entityID = s.EntityID
	} else if id, exists := (*entry)["entityID"]; exists {
		entityID, exists = id.(int)
		if !exists {
			return errors.New("未采集到组织ID 无法采集站点ID: 组织ID不合法")
		}
	}

	if entityID <= 0 {
		return errors.New("未采集到组织ID 无法采集站点ID")
	}

	c, err := r.Get(s.ColIndex)
	if err != nil {
		return err
	}
	name, err := c.GetString()
	if err != nil {
		return err
	}
	stationList, err := entity.GetStations(s.SiteID, nil, []int{entityID}, "", "", name)
	if err != nil {
		return err
	}

	for _, e := range stationList {
		if e.Name == name {
			(*entry)["stationID"] = e.ID
			return nil
		}
	}

	return nil
}

type MonitorScanner struct {
	SiteID   string
	ColIndex int
}

func (s *MonitorScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {
	c, err := r.Get(s.ColIndex)
	if err != nil {
		return err
	}
	name, err := c.GetString()
	if err != nil {
		return err
	}

	monitorList := monitor.GetAllMonitor(s.SiteID)
	for _, e := range monitorList {
		if e.Name == name {
			(*entry)["monitorID"] = e.ID
			return nil
		}
	}

	return nil
}

type StringScanner struct {
	StationID int
	MonitorID int
	Field     string
	ColIndex  int
}

func (s *StringScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {
	c, err := r.Get(s.ColIndex)
	if err != nil {
		return err
	}
	str, err := c.GetString()
	if err != nil {
		return err
	}

	var keys []string
	if s.StationID > 0 {
		keys = append(keys, fmt.Sprintf("%d", s.StationID))
	}
	if s.MonitorID > 0 {
		keys = append(keys, fmt.Sprintf("%d", s.MonitorID))
	}
	if s.Field == "" {
		return errors.New("未录入数值字段")
	}
	keys = append(keys, s.Field)
	(*entry)[strings.Join(keys, "#")] = str

	return nil
}

type FloatScanner struct {
	StationID int
	MonitorID int
	Field     string
	ColIndex  int
}

func (s *FloatScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {
	c, err := r.Get(s.ColIndex)
	if err != nil {
		return err
	}
	fl, err := c.GetFloat()
	if err != nil {
		return err
	}

	var keys []string
	if s.StationID > 0 {
		keys = append(keys, fmt.Sprintf("%d", s.StationID))
	}
	if s.MonitorID > 0 {
		keys = append(keys, fmt.Sprintf("%d", s.MonitorID))
	}
	if s.Field == "" {
		return errors.New("未录入数值字段")
	}
	keys = append(keys, s.Field)
	(*entry)[strings.Join(keys, "#")] = fl

	return nil
}

type TimeScanner struct {
	*TimeField
}

func (s *TimeScanner) ScanCell(r *excel.SheetRow, entry *map[string]any) error {

	result := new(time.Time)

	for _, i := range s.Indices {
		c, err := r.Get(i)
		if err != nil {
			return err
		}
		format := s.Formats[i]

		t := new(time.Time)
		if format != "" {
			str, err := c.GetString()
			if err != nil {
				return err
			}

			*t, err = util.ParseDateTimeWithFormat(str, format)
			if err != nil {
				return err
			}
		} else {
			t, err = c.GetTime()
			if err != nil {
				return err
			}
		}

		y, m, d := t.Date()
		if y == 0 {
			y = result.Year()
		}
		if m == 0 {
			m = result.Month()
		}
		if d == 0 {
			d = result.Day()
		}
		h, M, s := t.Clock()
		if h == 0 {
			h = result.Hour()
		}
		if M == 0 {
			M = result.Minute()
		}
		if s == 0 {
			s = result.Second()
		}

		*result = time.Date(y, m, d, h, M, s, 0, time.Local)
	}

	(*entry)["dataTime"] = result

	return nil
}

func (p *SheetParser) GetRowCellScanners(headers []*excel.SheetRow) ([]excel.ICellScaner[excel.SheetRow], error) {

	result := make([]excel.ICellScaner[excel.SheetRow], 0)

	if p.StationConfig.StationID > 0 {
		dataScanners, err := p.parseMonitorScanners(headers, p.StationConfig.StationID, -1, -1)
		if err != nil {
			return nil, err
		}

		result = append(result, dataScanners...)
	} else {
		if p.EntityConfig.EntityID > 0 {
			dataScanners, err := p.parseStationScanners(headers, p.EntityConfig.EntityID, -1, -1)
			if err != nil {
				return nil, err
			}

			result = append(result, dataScanners...)
		} else if p.EntityConfig.HeaderIndex > 0 {
			if len(headers) <= p.EntityConfig.HeaderIndex {
				return nil, errors.New("entity header index out of range")
			}
			entityHeader := headers[p.EntityConfig.HeaderIndex]

			entityCols := make([][2]int, 0)

			for i := 0; i < entityHeader.Len(); i++ {
				c, err := entityHeader.Get(i)
				if err != nil {
					return nil, err
				}

				entityName, err := c.GetString()
				if err != nil {
					return nil, err
				}

				entityList, err := entity.GetEntityList(p.SiteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_VIEW}}, "", "", nil, nil, entityName)
				if err != nil {
					return nil, err
				}

				for _, e := range entityList {
					if e.Name == entityName {
						entityCols = append(entityCols, [2]int{e.ID, i})
						break
					}
				}
			}

			for i, entityCol := range entityCols {
				var end = -1
				if i+1 < len(entityCols) {
					end = entityCols[i+1][1]
				}
				dataScanners, err := p.parseStationScanners(headers, entityCol[0], entityCol[1], end)
				if err != nil {
					return nil, err
				}

				result = append(result, dataScanners...)
			}

		} else if p.EntityConfig.Index > 0 {
			entityScanner := new(EntityScanner)
			entityScanner.SiteID = p.SiteID
			entityScanner.ColIndex = p.StationConfig.Index

			result = append(result, entityScanner)

			dataScanners, err := p.parseStationScanners(headers, -1, -1, -1)
			if err != nil {
				return nil, err
			}

			result = append(result, dataScanners...)
		} else {
			return nil, errors.New("未设置有效的组织站点录入")
		}
	}

	timeScanner := new(TimeScanner)
	timeScanner.TimeField = p.TimeConfig
	result = append(result, timeScanner)

	return result, nil
}

func (p *SheetParser) parseStationScanners(headers []*excel.SheetRow, entityID, colIndexStart, colIndexEnd int) ([]excel.ICellScaner[excel.SheetRow], error) {

	result := make([]excel.ICellScaner[excel.SheetRow], 0)

	if p.StationConfig.HeaderIndex > 0 {
		if entityID <= 0 {
			return nil, errors.New("未设置有效的站点录入: 无组织ID")
		}
		if len(headers) <= p.StationConfig.HeaderIndex {
			return nil, errors.New("station header index out of range")
		}
		stationHeader := headers[p.StationConfig.HeaderIndex]

		if colIndexStart < 0 {
			colIndexStart = 0
		}

		if colIndexEnd < 0 {
			colIndexEnd = stationHeader.Len()
		}

		for i := colIndexStart; i < colIndexEnd; i++ {
			c, err := stationHeader.Get(i)
			if err != nil {
				return nil, err
			}

			stationName, err := c.GetString()
			if err != nil {
				return nil, err
			}

			stationList, err := entity.GetStations(p.SiteID, nil, []int{entityID}, "", "", stationName)
			if err != nil {
				return nil, err
			}

			stationCols := make([][2]int, 0)

			for _, e := range stationList {
				if e.Name == stationName {
					stationCols = append(stationCols, [2]int{e.ID, i})
					break
				}
			}

			for i, col := range stationCols {
				var end = -1
				if i+1 < len(stationCols) {
					end = stationCols[i+1][1]
				}
				dataScanners, err := p.parseMonitorScanners(headers, col[0], col[1], end)
				if err != nil {
					return nil, err
				}

				result = append(result, dataScanners...)
			}
		}
	} else if p.StationConfig.Index > 0 {

		stationScanner := new(StationScanner)
		stationScanner.SiteID = p.SiteID
		stationScanner.ColIndex = p.StationConfig.Index
		stationScanner.EntityID = entityID

		result = append(result, stationScanner)

		dataScanners, err := p.parseMonitorScanners(headers, -1, -1, -1)
		if err != nil {
			return nil, err
		}
		result = append(result, dataScanners...)
	} else {
		return nil, errors.New("未设置有效的站点录入")
	}

	return result, nil
}

func (p *SheetParser) parseMonitorScanners(headers []*excel.SheetRow, stationID, colIndexStart, colIndexEnd int) ([]excel.ICellScaner[excel.SheetRow], error) {
	result := make([]excel.ICellScaner[excel.SheetRow], 0)

	if err := monitor.LoadMonitor(p.SiteID); err != nil {
		return nil, err
	}

	if p.DataConfig.MonitorHeaderIndex > 0 {
		if len(headers) <= p.DataConfig.MonitorHeaderIndex {
			return nil, errors.New("station header index out of range")
		}
		monitorHeader := headers[p.DataConfig.MonitorHeaderIndex]

		if colIndexStart < 0 {
			colIndexStart = 0
		}

		if colIndexEnd < 0 {
			colIndexEnd = monitorHeader.Len()
		}

		monitorList := monitor.GetAllMonitor(p.SiteID)

		for i := colIndexStart; i < colIndexEnd; i++ {
			c, err := monitorHeader.Get(i)
			if err != nil {
				return nil, err
			}

			name, err := c.GetString()
			if err != nil {
				return nil, err
			}

			monitorCols := make([][2]int, 0)
			for _, e := range monitorList {
				if e.Name == name {
					monitorCols = append(monitorCols, [2]int{e.ID, i})
					break
				}
			}

			for i, col := range monitorCols {
				var end = -1
				if i+1 < len(monitorCols) {
					end = monitorCols[i+1][1]
				}

				monitorID := col[0]
				colIndex := col[1]

				for _, f := range p.DataConfig.Fields {

					if colIndex >= end {
						break
					}

					switch f {
					case data.FLAG:
						stringScanner := new(StringScanner)
						stringScanner.ColIndex = colIndex
						stringScanner.MonitorID = monitorID
						stringScanner.Field = f

						result = append(result, stringScanner)
					case data.RTD:
						fallthrough
					case data.AVG:
						fallthrough
					case data.MIN:
						fallthrough
					case data.COU:
						fallthrough
					case data.MAX:
						floatScanner := new(FloatScanner)
						floatScanner.ColIndex = colIndex
						floatScanner.MonitorID = monitorID
						floatScanner.Field = f

						result = append(result, floatScanner)
					default:
					}

					colIndex++
				}

			}
		}
	} else if p.DataConfig.MonitorIndex > 0 {
		monitorScanner := new(MonitorScanner)
		monitorScanner.SiteID = p.SiteID
		monitorScanner.ColIndex = p.DataConfig.MonitorIndex

		result = append(result, monitorScanner)

		for f, colIndex := range p.DataConfig.FieldIndices {
			switch f {
			case data.FLAG:
				stringScanner := new(StringScanner)
				stringScanner.ColIndex = colIndex
				stringScanner.Field = f

				result = append(result, stringScanner)
			case data.RTD:
				fallthrough
			case data.AVG:
				fallthrough
			case data.MIN:
				fallthrough
			case data.COU:
				fallthrough
			case data.MAX:
				floatScanner := new(FloatScanner)
				floatScanner.ColIndex = colIndex
				floatScanner.Field = f

				result = append(result, floatScanner)
			default:
			}
		}

	} else {
		return nil, errors.New("未设置有效的检测物录入")
	}

	return result, nil
}
