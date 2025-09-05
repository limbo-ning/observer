package util_test

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"obsessiontech/common/util"
)

func TestParseTime(t *testing.T) {
	d, err := util.ParseDate("2017-09-12")
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%v", d)
	}

	d, err = util.ParseDateTimeWithFormat("9-12", "MM-YY")
	if err != nil {
		t.Error(err)
	} else {
		log.Println(d)
		t.Logf("%v", d)
	}

	d, err = util.ParseDateTimeWithFormat("17-9-12 9:12:15", "MM-DD hh:mm:ss")
	if err != nil {
		t.Error(err)
	} else {
		log.Println(d)
		t.Logf("%v", d)
	}
}

func TestTimeGob(t *testing.T) {
	origin := util.Time(time.Now())

	var tt util.Time

	if err := util.Clone(origin, &tt); err != nil {
		log.Println(err)
	} else {
		log.Println("ok", tt)
	}
}

func TestDurationGob(t *testing.T) {
	origin := util.Duration("1h")

	var tt util.Duration

	if err := util.Clone(origin, &tt); err != nil {
		log.Println(err)
	} else {
		log.Println("ok", tt)
	}
}

func TestParseDuration(t *testing.T) {

	d, err := time.ParseDuration("1h")
	if err != nil {
		panic(err)
	}

	log.Println(d)

	var param struct {
		D util.Duration `json:"d"`
	}

	if err := json.Unmarshal([]byte("{\"d\":\"1h\"}"), &param); err != nil {
		panic(err)
	}

	log.Println(param)
}

func TestParseDuration1(t *testing.T) {

	var d util.Duration

	if err := json.Unmarshal([]byte("\"10m\""), &d); err != nil {
		panic(err)
	}

	log.Println(d.GetDuration().Minutes())
}

func TestTruncateLocal(t *testing.T) {
	log.Println(util.TruncateLocal(time.Now(), time.Hour))
}
