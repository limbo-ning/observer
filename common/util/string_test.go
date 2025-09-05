package util_test

import (
	"log"
	"testing"

	"obsessiontech/common/util"
)

func TestUnsafeString(t *testing.T) {
	bytes, err := util.UnsafeJsonString("<xml?>>/xml>")

	if err != nil {
		panic(err)
	}

	log.Println(string(bytes))
}

func TestMask(t *testing.T) {
	log.Println("1:", util.Mask("1", "*"))
	log.Println("12:", util.Mask("12", "*"))
	log.Println("123:", util.Mask("123", "*"))
	log.Println("1234:", util.Mask("1234", "*"))
	log.Println("12345:", util.Mask("12345", "*"))
	log.Println("123456:", util.Mask("123456", "*"))
	log.Println("1234567:", util.Mask("1234567", "*"))
	log.Println("15521113114:", util.Mask("15521113114", "*"))
	log.Println("+86-18520278338:", util.Mask("+86-18520278338", "*"))
}
