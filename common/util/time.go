package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	FormatYearCN      = "年"
	FormatYear        = "YYYY"
	FormatShortYear   = "YY"
	FormatMonthCN     = "月"
	FormatMonth       = "MM"
	FormatShortMonth  = "M"
	FormatDayCN       = "日"
	FormatDay         = "DD"
	FormatShortDay    = "D"
	FormatHourCN      = "时"
	FormatHour        = "hh"
	FormatShortHour   = "h"
	FormatMinuteCN    = "分"
	FormatMinute      = "mm"
	FormatShortMinute = "m"
	FormatSecondCN    = "秒"
	FormatSecond      = "ss"
	FormatShortSecond = "s"
)

var DefaultMin = time.Unix(0, 0)
var DefaultMax = time.Date(9999, 12, 31, 0, 0, 0, 0, time.Local)

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return json.Marshal("")
	}
	var stamp = fmt.Sprintf("\"%s\"", FormatDateTime(time.Time(t)))
	return []byte(stamp), nil
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	var ts string
	if err := json.Unmarshal(data, &ts); err != nil {
		return err
	}
	if ts == "" {
		*t = Time(time.Time{})
		return
	}
	tt, err := ParseDateTime(ts)
	*t = Time(tt)
	return
}

func (t Time) GobEncode() ([]byte, error) {
	return time.Time(t).GobEncode()
}

func (t *Time) GobDecode(data []byte) error {
	var tt time.Time
	if err := tt.GobDecode(data); err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

func (t Time) String() string {
	return FormatDateTime(time.Time(t))
}

func translateFormat(value, format string) (string, error) {
	format = strings.Replace(format, FormatYearCN, "2006", -1)
	format = strings.Replace(format, FormatMonthCN, "01", -1)
	format = strings.Replace(format, FormatDayCN, "02", -1)
	format = strings.Replace(format, FormatHourCN, "15", -1)
	format = strings.Replace(format, FormatMinuteCN, "04", -1)
	format = strings.Replace(format, FormatSecondCN, "05", -1)

	if len(value) == len(format) {
		format = strings.Replace(format, FormatYear, "2006", -1)
		format = strings.Replace(format, FormatShortYear, "06", -1)
		format = strings.Replace(format, FormatMonth, "01", -1)
		format = strings.Replace(format, FormatShortMonth, "1", -1)
		format = strings.Replace(format, FormatDay, "02", -1)
		format = strings.Replace(format, FormatShortDay, "2", -1)
		format = strings.Replace(format, FormatHour, "15", -1)
		format = strings.Replace(format, FormatShortHour, "3", -1)
		format = strings.Replace(format, FormatMinute, "04", -1)
		format = strings.Replace(format, FormatShortMinute, "4", -1)
		format = strings.Replace(format, FormatSecond, "05", -1)
		format = strings.Replace(format, FormatShortSecond, "5", -1)

		return format, nil
	} else if len(value) > len(format) {
		return "", fmt.Errorf("格式不匹配[%s][%s]", value, format)
	}

	var check = func(i, first, second int) (bool, error) {
		target := value[i : i+first]
		if _, err := strconv.Atoi(target); err != nil {
			target = value[i : i+second]
			if _, err = strconv.Atoi(target); err != nil {
				return true, fmt.Errorf("格式不匹配[%s][%s]", value, format)
			}
			return true, nil
		}

		return false, nil
	}

	var next = func(i int) (int, error) {
		for ; i < len(format); i++ {
			switch format[i] {
			case 'Y':
				if len(format) > i+3 && format[i+3] == 'Y' {
					if len(value) > i+3 {
						if shouldReplace, err := check(i, 4, 2); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatYear, FormatShortYear, 1)
							return i + 2, nil
						} else {
							return i + 4, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			case 'M':
				if len(format) > i+1 && format[i+1] == 'M' {
					if len(value) > i+1 {
						if shouldReplace, err := check(i, 2, 1); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatMonth, FormatShortMonth, 1)
							return i + 1, nil
						} else {
							return i + 2, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			case 'D':
				if len(format) > i+1 && format[i+1] == 'D' {
					if len(value) > i+1 {
						if shouldReplace, err := check(i, 2, 1); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatDay, FormatShortDay, 1)
							return i + 1, nil
						} else {
							return i + 2, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			case 'h':
				if len(format) > i+1 && format[i+1] == 'h' {
					if len(value) > i+1 {
						if shouldReplace, err := check(i, 2, 1); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatHour, FormatShortHour, 1)
							return i + 1, nil
						} else {
							return i + 2, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			case 'm':
				if len(format) > i+1 && format[i+1] == 'm' {
					if len(value) > i+1 {
						if shouldReplace, err := check(i, 2, 1); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatMinute, FormatShortMinute, 1)
							return i + 1, nil
						} else {
							return i + 2, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			case 's':
				if len(format) > i+1 && format[i+1] == 's' {
					if len(value) > i+1 {
						if shouldReplace, err := check(i, 2, 1); err != nil {
							return -1, err
						} else if shouldReplace {
							format = strings.Replace(format, FormatSecond, FormatShortSecond, 1)
							return i + 1, nil
						} else {
							return i + 2, nil
						}
					} else {
						return -1, fmt.Errorf("格式不匹配[%s][%s]", value, format)
					}
				}
			}
		}

		return i, nil
	}

	var err error
	var i int

	for {
		if i >= len(format) {
			break
		}
		i, err = next(i)
		if err != nil {
			return "", err
		}
	}

	if len(format) != len(value) {
		return "", fmt.Errorf("格式不匹配[%s][%s]", value, format)
	}

	format = strings.Replace(format, FormatYear, "2006", -1)
	format = strings.Replace(format, FormatShortYear, "06", -1)
	format = strings.Replace(format, FormatMonth, "01", -1)
	format = strings.Replace(format, FormatShortMonth, "1", -1)
	format = strings.Replace(format, FormatDay, "02", -1)
	format = strings.Replace(format, FormatShortDay, "2", -1)
	format = strings.Replace(format, FormatHour, "15", -1)
	format = strings.Replace(format, FormatShortHour, "3", -1)
	format = strings.Replace(format, FormatMinute, "04", -1)
	format = strings.Replace(format, FormatShortMinute, "4", -1)
	format = strings.Replace(format, FormatSecond, "05", -1)
	format = strings.Replace(format, FormatShortSecond, "5", -1)

	return format, nil
}

func ParseDateTime(t string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
}

func ParseDateTimeWithFormat(t, format string) (time.Time, error) {
	translated, err := translateFormat(t, format)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(translated, t, time.Local)
}

func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func ParseDate(d string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", d, time.Local)
}

func ParseDateWithFormat(d, format string) (time.Time, error) {
	translated, err := translateFormat(d, format)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(translated, d, time.Local)
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func ParseTime(d string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", d, time.Local)
}

func ParseTimeWithFormat(d, format string) (time.Time, error) {
	translated, err := translateFormat(d, format)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(translated, d, time.Local)
}

func FormatTime(t time.Time) string {
	return t.Format("15:04:05")
}

func GetDate(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func GetEndOfDate(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, t.Location())
}

func TruncateLocal(t time.Time, d time.Duration) time.Time {
	_, offset := time.Now().Zone()
	result := t.Truncate(d)

	if d.Seconds() > float64(offset) {
		result = result.Add(-1 * time.Second * time.Duration(offset))
	}

	return result
}

type Duration string

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(d))
}
func (d *Duration) UnmarshalJSON(data []byte) error {
	var ds string
	if err := json.Unmarshal(data, &ds); err != nil {
		return err
	}
	if ds != "" {
		if _, err := time.ParseDuration(ds); err != nil {
			return err
		}
	}
	*d = Duration(ds)

	return nil
}

func (d Duration) GetDuration() time.Duration {
	result, _ := time.ParseDuration(string(d))
	return result
}
