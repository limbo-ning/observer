package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	year      = 'y'
	month     = 'm'
	day       = 'd'
	dayOfWeek = 'w'
	hour      = 'H'
	minute    = 'M'
	second    = 'S'
)

type Interval struct {
	Year       string
	Month      string
	Day        string
	DayOfWeek  string
	Hour       string
	Minute     string
	Second     string
	Years      valueRanges
	Months     valueRanges
	Days       valueRanges
	DaysOfWeek valueRanges
	Hours      valueRanges
	Minutes    valueRanges
	Seconds    valueRanges
}
type preset struct {
	Field  rune
	Values []int
}

type valueRange struct {
	Values map[int][2]int
	Preset []*preset
}

type valueRanges []*valueRange

func getTimeField(field rune, t time.Time) int {
	switch field {
	case year:
		return t.Year()
	case month:
		return int(t.Month())
	case day:
		return t.Day()
	case dayOfWeek:
		return int(t.Weekday())
	case hour:
		return t.Hour()
	case minute:
		return t.Minute()
	case second:
		return t.Second()
	}
	return -1
}

func (r *valueRange) isInRange(field rune, t time.Time) (inPreset, inRange bool, start, end int) {

	inPreset = true
	start = -1
	end = -1

	if len(r.Preset) > 0 {
		for _, p := range r.Preset {
			tt := getTimeField(p.Field, t)
			checked := false
			for _, pp := range p.Values {
				if pp == tt {
					checked = true
					break
				}
			}
			if !checked {
				inPreset = false
				inRange = false
				return
			}
		}
	} else {
		inPreset = false
	}

	limits, exists := r.Values[getTimeField(field, t)]
	if !exists {
		inRange = false
		for ele := range r.Values {
			if start == -1 || start > ele {
				start = ele
			}
			if end == -1 || end < ele {
				end = ele
			}
		}
		return
	}

	inRange = true
	start = limits[0]
	end = limits[1]
	return
}

func (r valueRanges) isInRange(field rune, t time.Time) (inRange bool, start, end int) {
	for _, vr := range r {
		perInPreset, perInRange, perStart, perEnd := vr.isInRange(field, t)

		if perInPreset {
			if !perInRange {
				inRange = false
				start = perStart
				end = perEnd
				return
			}
		}

		if !perInRange {
			if start >= perStart {
				start = perStart
			}
			if end <= perEnd {
				end = perEnd
			}
			continue
		}

		inRange = true
		start = perStart
		end = perEnd

		return
	}
	inRange = false
	return
}

func (i Interval) MarshalJSON() ([]byte, error) {
	if i.Years == nil {
		if err := i.Validate(); err != nil {
			return nil, err
		}
	}
	if i.IsUnlimited() {
		return []byte("\"\""), nil
	}
	return []byte(fmt.Sprintf("\"%s %s %s %s %s %s %s\"", i.Year, i.Month, i.Day, i.DayOfWeek, i.Hour, i.Minute, i.Second)), nil
}
func (i *Interval) UnmarshalJSON(data []byte) error {

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	if str == "" {
		return i.Init()
	}

	parts := strings.Split(string(str), " ")
	if len(parts) != 7 {
		log.Println("错误格式: ", string(str), len(parts))
		return errors.New("错误格式")
	}

	i.Year = parts[0]
	i.Month = parts[1]
	i.Day = parts[2]
	i.DayOfWeek = parts[3]
	i.Hour = parts[4]
	i.Minute = parts[5]
	i.Second = parts[6]

	return i.Validate()
}

func (i *Interval) Init() error {
	i.Year = "."
	i.Month = "."
	i.Day = "."
	i.DayOfWeek = "."
	i.Hour = "."
	i.Minute = "."
	i.Second = "."
	return i.Validate()
}

func (i *Interval) IsUnlimited() bool {
	return len(i.Years) == 0 && len(i.Months) == 0 && len(i.Days) == 0 && len(i.DaysOfWeek) == 0 && len(i.Hours) == 0 && len(i.Minutes) == 0 && len(i.Seconds) == 0
}

func (i *Interval) Validate() error {

	var err error
	i.Years, err = i.validate(year)
	if err != nil {
		return err
	}
	i.Months, err = i.validate(month)
	if err != nil {
		return err
	}
	i.Days, err = i.validate(day)
	if err != nil {
		return err
	}
	i.DaysOfWeek, err = i.validate(dayOfWeek)
	if err != nil {
		return err
	}
	i.Hours, err = i.validate(hour)
	if err != nil {
		return err
	}
	i.Minutes, err = i.validate(minute)
	if err != nil {
		return err
	}
	i.Seconds, err = i.validate(second)
	if err != nil {
		return err
	}

	return nil
}

func (i *Interval) getFieldValue(field rune) string {
	switch field {
	case year:
		return i.Year
	case month:
		return i.Month
	case day:
		return i.Day
	case dayOfWeek:
		return i.DayOfWeek
	case hour:
		return i.Hour
	case minute:
		return i.Minute
	case second:
		return i.Second
	}
	return ""
}

func (i *Interval) validate(field rune, limit ...int) (valueRanges, error) {
	value := i.getFieldValue(field)
	if value == "" {
		return nil, fmt.Errorf("缺少数值[%s]", string(field))
	}
	result := make([]*valueRange, 0)
	if value == "." {
		return result, nil
	}
	min, max := i.getFieldRange(field, limit...)

	inPreset := false
	preset := make([]rune, 0)
	values := make([]rune, 0)

	parse := func() error {
		var err error
		ranges := new(valueRange)
		if len(preset) > 0 {
			ranges.Preset, err = i.parsePresets(string(preset))
			if err != nil {
				return err
			}
		}

		ranges.Values, err = i.parseRange(string(values), min, max)
		if err != nil {
			return err
		}
		result = append(result, ranges)

		return nil
	}

	var err error
	for _, v := range value {
		switch v {
		case '(':
			inPreset = true
			preset = append(preset, v)
		case ')':
			inPreset = false
			preset = append(preset, v)
		case ',':
			if !inPreset {
				if err = parse(); err != nil {
					return nil, err
				}

				preset = make([]rune, 0)
				values = make([]rune, 0)
			} else {
				preset = append(preset, v)
			}
		default:
			if inPreset {
				preset = append(preset, v)
			} else {
				values = append(values, v)
			}
		}
	}

	if err = parse(); err != nil {
		return nil, err
	}

	return result, nil
}

func (i *Interval) getFieldRange(field rune, limit ...int) (min, max int) {
	defaultLimit := true
	if len(limit) >= 2 {
		defaultLimit = false
		min = limit[0]
		max = limit[1]
	}
	switch field {
	case year:
		if defaultLimit {
			min = time.Now().Year() - 100
			max = time.Now().Year() + 100
		}
	case month:
		if defaultLimit {
			min = 1
			max = 12
		}
	case day:
		if defaultLimit {
			min = 1
			max = 31
		}
	case dayOfWeek:
		if defaultLimit {
			min = 0
			max = 6
		}
	case hour:
		if defaultLimit {
			min = 0
			max = 23
		}
	case minute:
		if defaultLimit {
			min = 0
			max = 59
		}
	case second:
		if defaultLimit {
			min = 0
			max = 59
		}
	}
	return
}

func (i *Interval) parsePresets(input string) ([]*preset, error) {
	result := make([]*preset, 0)

	var parsing *preset
	value := make([]rune, 0)

	toParse := func(field rune) error {
		if parsing != nil {
			min, max := i.getFieldRange(parsing.Field)
			slots, err := i.parseRange(string(value), min, max)
			if err != nil {
				return err
			}

			entries := make(map[int]int)
			for _, s := range slots {
				for _, ele := range s {
					entries[ele] = 1
				}
			}

			parsing.Values = make([]int, 0)
			for ele := range entries {
				parsing.Values = append(parsing.Values, ele)
			}

			result = append(result, parsing)
			value = make([]rune, 0)
		}
		if field != ')' {
			parsing = new(preset)
			parsing.Field = field
		}
		return nil
	}

	for _, v := range input {
		switch v {
		case '(':
		case year:
			fallthrough
		case month:
			fallthrough
		case day:
			fallthrough
		case dayOfWeek:
			fallthrough
		case hour:
			fallthrough
		case minute:
			fallthrough
		case second:
			fallthrough
		case ')':
			if err := toParse(v); err != nil {
				return nil, err
			}
		default:
			value = append(value, v)
		}
	}

	return result, nil
}

func (i *Interval) parseRange(input string, min, max int) (result map[int][2]int, err error) {

	result = make(map[int][2]int)

	if input == "*" {
		for i := min; i <= max; i++ {
			result[i] = [2]int{i, i}
		}
		return
	}

	parts := strings.Split(input, ",")
	for _, p := range parts {
		ranges := strings.Split(p, "-")
		if len(ranges) == 1 {
			number, err := strconv.Atoi(p)
			if err != nil {
				return nil, err
			}
			if number < min || number > max {
				return nil, fmt.Errorf("数值取值不在范围内%d: (%d-%d)", number, min, max)
			}
			result[number] = [2]int{number, number}
		} else if len(ranges) == 2 {
			thisMin, err := strconv.Atoi(ranges[0])
			if err != nil {
				return nil, err
			}
			if thisMin < min || thisMin > max {
				return nil, fmt.Errorf("数值取值不在范围内%d: (%d-%d)", thisMin, min, max)
			}
			thisMax, err := strconv.Atoi(ranges[1])
			if err != nil {
				return nil, err
			}
			if thisMax < min || thisMax > max {
				return nil, fmt.Errorf("数值取值不在范围内%d: (%d-%d)", thisMax, min, max)
			}
			if thisMin > thisMax {
				return nil, fmt.Errorf("数值大小范围错误: (%d-%d)", thisMin, thisMax)
			}
			for i := thisMin; i <= thisMax; i++ {
				result[i] = [2]int{thisMin, thisMax}
			}
		} else {
			return nil, fmt.Errorf("错误的范围[%s]", input)
		}
	}

	return
}

var E_not_in_interval = errors.New("所选时间不可用")

func (i *Interval) GetInterval(spot time.Time, isStrict bool) (start *time.Time, end *time.Time, err error) {

	if i.Years == nil {
		if err = i.Validate(); err != nil {
			return
		}
	}

	if i.IsUnlimited() {
		return &DefaultMin, &DefaultMax, nil
	}

	if len(i.Years) != 0 {
		inRange, min, max := i.Years.isInRange(year, spot)
		if inRange {
			start = new(time.Time)
			end = new(time.Time)
			*start = time.Date(min, 1, 1, 0, 0, 0, 0, time.Local)
			*end = time.Date(max, 1, 1, 0, 0, 0, 0, time.Local).AddDate(1, 0, 0).Add(-1 * time.Second)
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}

			if max < spot.Year() {
				return nil, nil, E_not_in_interval
			}

			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), 1, 1, 0, 0, 0, 0, time.Local).AddDate(1, 0, 0), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}
	}

	if len(i.Months) != 0 {
		inRange, min, max := i.Months.isInRange(month, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), time.Month(min), 1, 0, 0, 0, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), time.Month(min), 1, 0, 0, 0, 0, time.Local)
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), time.Month(max), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0).Add(-1 * time.Second)
			} else {
				*end = time.Date(end.Year(), time.Month(max), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0).Add(-1 * time.Second)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.Months.isInRange(month, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), time.Month(min), 1, 0, 0, 0, 0, time.Local)
		}

		_, _, endMax := i.Months.isInRange(month, *end)
		if endMax != max {
			*end = time.Date(end.Year(), time.Month(max), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0).Add(-1 * time.Second)
		}

	}

	if len(i.Days) != 0 {
		inRange, min, max := i.Days.isInRange(day, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), spot.Month(), min, 0, 0, 0, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), start.Month(), min, 0, 0, 0, 0, time.Local)
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), spot.Month(), max, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
			} else {
				*end = time.Date(end.Year(), end.Month(), max, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), spot.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.Days.isInRange(day, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), start.Month(), beginMin, 0, 0, 0, 0, time.Local)
		}

		_, _, endMax := i.Days.isInRange(day, *end)
		if endMax != max {
			*end = time.Date(end.Year(), end.Month(), endMax, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
		}
	}

	if len(i.DaysOfWeek) != 0 {
		inRange, min, max := i.DaysOfWeek.isInRange(dayOfWeek, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), spot.Month(), spot.Day()+int(spot.Weekday())-min, 0, 0, 0, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), start.Month(), spot.Day()+int(spot.Weekday())-min, 0, 0, 0, 0, time.Local)
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), spot.Month(), spot.Day()+max-int(spot.Weekday()), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
			} else {
				*end = time.Date(end.Year(), end.Month(), spot.Day()+max-int(spot.Weekday()), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), spot.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.DaysOfWeek.isInRange(dayOfWeek, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), start.Month(), spot.Day()+int(spot.Weekday())-beginMin, 0, 0, 0, 0, time.Local)
		}

		_, _, endMax := i.DaysOfWeek.isInRange(dayOfWeek, *end)
		if endMax != max {
			*end = time.Date(end.Year(), end.Month(), spot.Day()+endMax-int(spot.Weekday()), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1).Add(-1 * time.Second)
		}
	}

	if len(i.Hours) != 0 {
		inRange, min, max := i.Hours.isInRange(hour, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), spot.Month(), spot.Day(), min, 0, 0, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), start.Month(), start.Day(), min, 0, 0, 0, time.Local)
			}

			hourEnd := max
			timeOffset := time.Hour - time.Second
			if max > 23 {
				hourEnd -= 24
				timeOffset += 24 * time.Hour
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), spot.Month(), spot.Day(), hourEnd, 0, 0, 0, time.Local).Add(timeOffset)
			} else {
				*end = time.Date(end.Year(), end.Month(), end.Day(), hourEnd, 0, 0, 0, time.Local).Add(timeOffset)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), 0, 0, 0, time.Local).Add(time.Hour), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.Hours.isInRange(hour, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), start.Month(), start.Day(), beginMin, 0, 0, 0, time.Local)
		}

		_, endMax, _ := i.Hours.isInRange(hour, *end)
		if endMax != max {
			hourEnd := max
			timeOffset := time.Hour - time.Second
			if max > 23 {
				hourEnd -= 24
				timeOffset += 24 * time.Hour
			}
			*end = time.Date(end.Year(), end.Month(), end.Day(), hourEnd, 0, 0, 0, time.Local).Add(timeOffset)
		}
	}

	if len(i.Minutes) != 0 {
		inRange, min, max := i.Minutes.isInRange(minute, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), min, 0, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), min, 0, 0, time.Local)
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), max, 0, 0, time.Local).Add(time.Minute - time.Second)
			} else {
				*end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), max, 0, 0, time.Local).Add(time.Minute - time.Second)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), spot.Minute(), 0, 0, time.Local).Add(time.Minute), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.Minutes.isInRange(minute, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), beginMin, 0, 0, time.Local)
		}

		_, _, endMax := i.Minutes.isInRange(minute, *end)
		if endMax != max {
			*end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), endMax, 0, 0, time.Local).Add(time.Minute - time.Second)
		}
	}

	if len(i.Seconds) != 0 {
		inRange, min, max := i.Seconds.isInRange(second, spot)
		if inRange {
			if start == nil {
				start = new(time.Time)
				*start = time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), spot.Minute(), min, 0, time.Local)
			} else {
				*start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), min, 0, time.Local)
			}

			if end == nil {
				end = new(time.Time)
				*end = time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), spot.Minute(), max, 0, time.Local)
			} else {
				*end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), max, 0, time.Local)
			}
		} else {
			if isStrict {
				return nil, nil, E_not_in_interval
			}
			nextStart, nextEnd, _ := i.GetInterval(time.Date(spot.Year(), spot.Month(), spot.Day(), spot.Hour(), spot.Minute(), spot.Second(), 0, time.Local).Add(time.Second), isStrict)
			return nextStart, nextEnd, E_not_in_interval
		}

		_, beginMin, _ := i.Seconds.isInRange(second, *start)
		if beginMin != min {
			*start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), beginMin, 0, time.Local)
		}

		_, _, endMax := i.Seconds.isInRange(second, *end)
		if endMax != max {
			*end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), endMax, 0, time.Local)
		}
	}

	return
}
