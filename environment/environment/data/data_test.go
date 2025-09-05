package data_test

import (
	"log"
	"testing"
	"time"

	"obsessiontech/common/util"
	// "obsessiontech/environment/environment/data"
)

// func TestInterface(t *testing.T) {
// 	r := &data.RealTimeData{}
// 	h := &data.HourlyData{}

// 	func(d data.IData) {
// 		if _, ok := d.(data.IRealTime); !ok {
// 			t.Error("rtd should be IRealTime")
// 		}
// 	}(r)

// 	func(d data.IData) {
// 		if _, ok := d.(data.IInterval); !ok {
// 			t.Error("hourly should be IInterval")
// 		}
// 		if _, ok := d.(data.IReview); !ok {
// 			t.Error("hourly should be IReview")
// 		}
// 	}(h)
// }

func getDataTimeEntries(beginTime, endTime time.Time) []time.Time {
	result := make([]time.Time, 0)

	var step time.Duration

	// step = time.Hour
	step = time.Hour * 24

	stepper := util.TruncateLocal(beginTime, step)

	for {
		log.Println(stepper, beginTime.Before(stepper))
		if stepper.After(endTime) {
			break
		}
		if !beginTime.After(stepper) {
			result = append(result, time.Time(stepper))
		}
		stepper = stepper.Add(step)
	}

	return result
}

func TestDataTimeEntries(t *testing.T) {
	list := getDataTimeEntries(time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local), time.Date(2021, 10, 7, 0, 0, 0, 0, time.Local))
	log.Println(list)

	// log.Println(time.Now().Zone())

	// log.Println(time.Date(0, 1, 1, 0, 0, 0, 0, time.Local))
	// log.Println(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))

	// log.Println(time.Date(0, 1, 1, 0, 0, 0, 0, time.Local).Sub(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)))
}
