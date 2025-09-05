package util_test

import (
	"log"
	"testing"

	"obsessiontech/common/util"
)

func TestAccuracy(t *testing.T) {
	input := 0.27899999999996

	log.Println(util.GetAccuracy(input), util.ApplyAccuracy(input, 5))

	input = 1591.1
	log.Println(util.GetAccuracy(input), util.ApplyAccuracy(input, 5))

	input = 1291.1003992105
	log.Println(util.GetAccuracy(input), util.ApplyAccuracy(input, 5))

	input = 12911
	log.Println(util.GetAccuracy(input), util.ApplyAccuracy(input, 5))

	input = 302 * 0.1
	log.Println(util.GetAccuracy(input), util.ApplyAccuracy(input, 1))

}
