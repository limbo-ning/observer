package util_test

import (
	"encoding/json"
	"log"
	"testing"

	"obsessiontech/common/util"
)

func TestLang1(t *testing.T) {
	str := "\"可恶\""

	var l util.Lang

	if err := json.Unmarshal([]byte(str), &l); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}
	data, _ := json.Marshal(&l)
	log.Println(string(data))

	l.Selected = []string{"default"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))

	l.Selected = []string{"cn"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))

	l.Selected = []string{"cn", "en"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))
}

func TestLang2(t *testing.T) {
	str := "{\"default\":\"可恶\",\"cn\":\"可恶\",\"en\":\"shit\"}"

	var l util.Lang

	if err := json.Unmarshal([]byte(str), &l); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}
	data, _ := json.Marshal(&l)
	log.Println(string(data))

	l.Selected = []string{"default"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))

	l.Selected = []string{"cn"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))

	l.Selected = []string{"cn", "en"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))

	l.Selected = []string{"cn", "jp"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))
}

func TestLang3(t *testing.T) {

	var l *util.Lang
	l = new(util.Lang)
	l.Init()

	data, _ := json.Marshal(&l)
	log.Println(string(data))

	l.Selected = []string{"default"}
	data, _ = json.Marshal(&l)
	log.Println(string(data))
}

func TestLang4(t *testing.T) {
	str := "{}"

	var l util.Lang

	if err := json.Unmarshal([]byte(str), &l); err != nil {
		log.Println("error : ", err)
		t.Error(err)
	}
	data, _ := json.Marshal(&l)
	log.Println(string(data))

	l.Selected = []string{"default"}
	data, _ = json.Marshal(&l)
	log.Println(l.Selected, string(data))
}
