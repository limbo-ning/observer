package serial_test

import (
	"log"
	"testing"

	"obsessiontech/common/random/serial"
)

func TestSerial(t *testing.T) {
	serial := serial.GenerateSerial()
	log.Println(serial, len(serial))
}
