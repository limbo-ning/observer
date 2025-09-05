package data

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	COMPARATOR_E   = "="
	COMPARATOR_NE  = "!="
	COMPARATOR_LE  = "<="
	COMPARATOR_GE  = ">="
	COMPARATOR_L   = "<"
	COMPARATOR_G   = ">"
	COMPARATOR_BIT = "&"
)

type Criteria struct {
	MonitorID     int         `json:"monitorID"`
	MonitorCodeID int         `json:"monitorCodeID"`
	Field         string      `json:"field"`
	Comparator    string      `json:"comparator"`
	Value         interface{} `json:"value"`

	And Criterias `json:"and"`
}

func (c *Criteria) compare(value interface{}) bool {
	switch c.Field {
	case RTD:
		fallthrough
	case AVG:
		fallthrough
	case MIN:
		fallthrough
	case MAX:
		fallthrough
	case COU:
		cv, ok := c.Value.(float64)
		if !ok {
			return false
		}
		v, ok := value.(float64)
		if !ok {
			return false
		}
		switch c.Comparator {
		case COMPARATOR_E:
			return v == cv
		case COMPARATOR_NE:
			return v != cv
		case COMPARATOR_G:
			return v > cv
		case COMPARATOR_L:
			return v < cv
		case COMPARATOR_LE:
			return v <= cv
		case COMPARATOR_GE:
			return v >= cv
		}

	case FLAG:
		cv, ok := c.Value.(string)
		if !ok {
			return false
		}
		v, ok := value.(string)
		if !ok {
			return false
		}
		switch c.Comparator {
		case COMPARATOR_E:
			return cv == v
		case COMPARATOR_NE:
			return cv != v
		}
	case FLAG_BIT:
		cv, ok := c.Value.(int)
		if !ok {
			return false
		}
		v, ok := value.(int)
		if !ok {
			return false
		}
		switch c.Comparator {
		case COMPARATOR_BIT:
			return cv&v == cv
		}
	}

	return false
}

func (c *Criteria) Validate() error {

	if c.MonitorID <= 0 || c.MonitorCodeID <= 0 {
		return errors.New("错误的监测物")
	}

	switch c.Field {
	case RTD:
		fallthrough
	case AVG:
		fallthrough
	case MIN:
		fallthrough
	case MAX:
		fallthrough
	case COU:
		switch c.Comparator {
		case COMPARATOR_E:
		case COMPARATOR_NE:
		case COMPARATOR_G:
		case COMPARATOR_L:
		case COMPARATOR_LE:
		case COMPARATOR_GE:
		default:
			return errors.New("错误的比较")
		}
		_, ok := c.Value.(float64)
		if !ok {
			return errors.New("错误的数据值")
		}
	case FLAG:
		switch c.Comparator {
		case COMPARATOR_E:
		case COMPARATOR_NE:
		default:
			return errors.New("错误的比较")
		}
		_, ok := c.Value.(string)
		if !ok {
			return errors.New("错误的数据值")
		}
	case FLAG_BIT:
		switch c.Comparator {
		case COMPARATOR_BIT:
		default:
			return errors.New("错误的比较")
		}
		_, ok := c.Value.(int)
		if !ok {
			return errors.New("错误的数据值")
		}
	default:
		return errors.New("错误的数据段")
	}

	return c.And.Validate()
}

func (c *Criteria) ParseSQL(dataType, tableAlias string) (string, []interface{}) {
	values := make([]interface{}, 0)

	switch dataType {
	case REAL_TIME:
		if c.Field != RTD {
			return "", values
		}
	default:
		if c.Field == RTD {
			return "", values
		}
	}

	var SQL string
	if c.MonitorCodeID > 0 {
		SQL = fmt.Sprintf("(%s.%s = ? AND %s.%s %s ?)", tableAlias, MONITOR_CODE_ID, tableAlias, c.Field, c.Comparator)
		values = append(values, c.MonitorCodeID, c.Value)
	} else if c.MonitorID > 0 {
		SQL = fmt.Sprintf("(%s.%s = ? AND %s.%s %s ?)", tableAlias, MONITOR_ID, tableAlias, c.Field, c.Comparator)
		values = append(values, c.MonitorID, c.Value)
	} else {
		return "", values
	}

	subSQL, subValues := c.And.ParseSQL(dataType, tableAlias)
	if subSQL != "" {
		SQL += " AND " + subSQL
		values = append(values, subValues...)
	}

	return SQL, values
}

type Criterias []*Criteria

func (cs Criterias) Validate() error {

	if len(cs) == 0 {
		return nil
	}

	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (cs Criterias) ParseSQL(dataType, tableAlias string) (string, []interface{}) {

	ors := make([]string, 0)
	values := make([]interface{}, 0)

	for _, c := range cs {
		subSQL, subValues := c.ParseSQL(dataType, tableAlias)

		if subSQL != "" {
			ors = append(ors, subSQL)
			values = append(values, subValues...)
		}
	}

	if len(ors) == 0 {
		return "", values
	}

	return fmt.Sprintf("(%s)", strings.Join(ors, " OR ")), values
}

func (c Criteria) checkData(data IData) bool {

	if c.MonitorCodeID > 0 && c.MonitorCodeID != data.GetMonitorCodeID() {
		return false
	}

	if c.MonitorID > 0 && data.GetMonitorID() != c.MonitorID {
		return false
	}

	switch c.Field {
	case FLAG:
		return c.compare(data.GetFlag())
	case FLAG_BIT:
		return c.compare(data.GetFlagBit())
	}

	if rtd, ok := data.(IRealTime); ok {
		if c.Field != RTD {
			return false
		}

		return c.compare(rtd.GetRtd())
	} else if interval, ok := data.(IInterval); ok {
		if c.Field == RTD {
			return false
		}

		switch c.Field {
		case MIN:
			return c.compare(interval.GetMin())
		case MAX:
			return c.compare(interval.GetMax())
		case AVG:
			return c.compare(interval.GetAvg())
		case COU:
			return c.compare(interval.GetCou())
		default:
			return false
		}
	} else {
		return false
	}
}

func (c Criteria) checkDataGroup(dataList []IData) bool {

	checked := false

	for _, d := range dataList {
		if d.GetMonitorID() != c.MonitorID {
			continue
		}
		checked = true
		if !c.checkData(d) {
			return false
		}
	}

	if !checked {
		return false
	}

	if len(c.And) > 0 {
		for _, c := range c.And {
			if !c.checkDataGroup(dataList) {
				return true
			}
		}

		return false
	}

	return true
}

func (cs Criterias) FilterData(dataList []IData, isTimeGroup bool) []IData {

	if len(cs) == 0 {
		return dataList
	}

	filtered := make([]IData, 0)

	if isTimeGroup {
		timeGroups := make(map[time.Time]map[int][]IData)

		for _, d := range dataList {
			dataTime := time.Time(d.GetDataTime())
			stations, exists := timeGroups[dataTime]
			if !exists {
				stations = make(map[int][]IData)
				timeGroups[dataTime] = stations
			}
			list, exists := stations[d.GetStationID()]
			if !exists {
				list = make([]IData, 0)
			}
			list = append(list, d)
			stations[d.GetStationID()] = list
		}

		for _, stations := range timeGroups {
			for _, list := range stations {
				for _, c := range cs {
					if c.checkDataGroup(list) {
						filtered = append(filtered, list...)
						break
					}
				}
			}
		}

	} else {
		for _, d := range dataList {
			for _, c := range cs {
				if c.checkData(d) {
					if len(c.And) > 0 {
						if len(c.And.FilterData([]IData{d}, isTimeGroup)) == 1 {
							filtered = append(filtered, d)
						}
					} else {
						filtered = append(filtered, d)
					}
					break
				}
			}
		}
	}

	return filtered
}
