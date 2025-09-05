package util_test

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"obsessiontech/common/util"
)

func TestJson(t *testing.T) {
	str := "\"* * * * * * *\""

	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}
	data, _ := json.Marshal(&interval)
	log.Println(string(data), interval.IsUnlimited())

	if interval.IsUnlimited() {
		t.Error("should not be unlimited")
	}

	str = "\"\""
	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	data, _ = json.Marshal(&interval)
	log.Println(string(data), interval.IsUnlimited())

	if !interval.IsUnlimited() {
		t.Error("should be unlimited")
		data, _ = json.Marshal(&interval)
		log.Println(string(data), interval.IsUnlimited())
	}
}

func TestInterval1(t *testing.T) {
	str := "\". . . . 8-23 (H8)30-59,(H23)0-29,* .\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 8, 20, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 8, 40, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 23 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 9, 40, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 23 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 23, 29, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 23 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 23, 31, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 23, 59, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}
}

func TestInterval2(t *testing.T) {
	str := "\". . . . 8-11,14-18 (H8,14)30-59,(H11,18)0-29,* .\""
	// str := "\". . . . 8-11,14-18 . .\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 8, 20, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 8, 40, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 11 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 9, 40, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 11 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 11, 31, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 15, 29, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 14 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 18 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 18, 20, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin == nil || begin.Hour() != 14 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 18 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 23, 59, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, true)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if begin != nil {
		t.Error()
		return
	}
}

func TestInterval3(t *testing.T) {
	str := "\". . . . 8-11,14-18 (H8,14)30-59,(H11,18)0-29,* .\""
	// str := "\". . . . 8-11,14-18 . .\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if err != nil || begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 11 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 8, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if err != nil || begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 11 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 19, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if err != nil || begin == nil || begin.Hour() != 8 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 11 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}

	spot = time.Date(2021, 4, 4, 12, 0, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if err != nil || begin == nil || begin.Hour() != 14 || begin.Minute() != 30 || begin.Second() != 0 || end.Hour() != 18 || end.Minute() != 29 || end.Second() != 59 {
		t.Error()
		return
	}
}

func TestInterval4(t *testing.T) {
	str := "\". . . . * 0 0\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v]", str, spot, begin, end)
	if err != nil || begin == nil || begin.Hour() != 8 || begin.Minute() != 0 || begin.Second() != 0 || end.Hour() != 8 || end.Minute() != 0 || end.Second() != 0 {
		t.Error()
		return
	}
}

func TestInterval5(t *testing.T) {
	str := "\"2001 . . . * 0 0\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v] err[%v]", str, spot, begin, end, err)
	if err == nil {
		t.Error()
		return
	}
}

func TestInterval6(t *testing.T) {
	str := "\"2100 . . . * 0 0\""
	var interval util.Interval

	if err := json.Unmarshal([]byte(str), &interval); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}

	var spot time.Time
	var err error

	var begin, end *time.Time

	spot = time.Date(2021, 4, 4, 7, 50, 0, 0, time.Local)
	begin, end, err = interval.GetInterval(spot, false)
	log.Printf("interval[%s] spot[%v] begin[%v] end[%v] err[%v]", str, spot, begin, end, err)
	if err != nil || begin == nil || begin.Year() != 2100 {
		t.Error()
		return
	}
}
