package instruction_test

import (
	"log"
	"obsessiontech/environment/environment/receiver/noise/instruction"
	"testing"
)

func TestParseInstruction(t *testing.T) {
	raw := "##0049&MN=LGZS0020220803,QN=20220804092805000,Leq=00.0&4100\r\n"

	i, err := instruction.Parse(raw)
	if err != nil {
		log.Println("error parse: ", err)
		t.Error(err)
		return
	}

	log.Println("result: ", i.MN, i.DateTime, i.Data)
}
