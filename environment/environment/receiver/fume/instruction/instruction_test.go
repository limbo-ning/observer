package instruction_test

import (
	"testing"

	"obsessiontech/environment/environment/receiver/fume/instruction"
)

func Test_ParseInstruction(t *testing.T) {
	i, err := instruction.Parse("##MN=GM100000100014;DateTime=20181008124100&&a301=0.37;a302=0.37;a303=0.37;a304=0;Dr=1;Cl=1;Flag=1;Wa=1&&9741")
	if err != nil {
		t.Error(err)
	}

	t.Log(i)
}
