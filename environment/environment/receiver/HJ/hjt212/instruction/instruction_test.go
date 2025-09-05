package instruction_test

import (
	"log"
	"obsessiontech/environment/environment/receiver/HJ/hjt212/instruction"
	"testing"
)

func Test_ComposeInstruction(t *testing.T) {

	i := new(instruction.Instruction)
	i.CN = "2011"
	i.PW = "123456"
	i.MN = "TEST001"
	i.ST = "91"
	i.QN = "20230227"
	i.CP = make([]map[string]string, 0)
	i.CP = append(i.CP, map[string]string{
		"DataTime": "20230227100800",
	}, map[string]string{
		"dl01-Rtd": "12",
	}, map[string]string{
		"dy01-Rtd": "550",
	}, map[string]string{
		"yw01-Rtd": "5",
	})

	log.Println(instruction.PackDatagram(instruction.ComposeInstruction(i)))

}
