package excel_test

import (
	"encoding/json"
	"io"
	"log"
	"obsessiontech/common/excel"
	"os"
	"testing"
)

func TestParser(t *testing.T) {
	parser := new(excel.Uploader)

	if err := json.Unmarshal([]byte(`
	{"name": "汽修企业", "sheets": [{"content": [2, -1], "entries": [{"type": "string", "field": "name", "index": 2}, {"type": "string", "field": "street", "index": 3}, {"type": "string", "field": "address", "index": 4}, {"type": "string", "field": "person_in_charge", "index": 5}, {"type": "string", "field": "contact", "index": 6}], "direction": 0, "sheetIndices": [0]}]}
	`), &parser); err != nil {
		t.Error(err)
		return
	}

	log.Println("parser unmarsheled: ", parser.Name, len(parser.Sheets))

	var data []byte

	f, err := os.OpenFile("/Users/limbo/Documents/华澜环保/开福区涉气企业名单2022.4.8.xlsx", os.O_RDONLY, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}

	data, err = io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}

	ret, err := excel.ParseExcel(data, parser)
	if err != nil {
		t.Error(err)
	}

	log.Println("parse done: ", ret)
}
