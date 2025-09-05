package hjt212

import (
	"log"
	"testing"
)

func Test_ParseTime(t *testing.T) {
	time, _ := ParseTime("20170617010334")
	log.Println(time)
	t.Log(time)
}

func Test_GenerateQN(t *testing.T) {
	QN := GenerateQN()
	log.Println(QN)
	t.Log(QN)
}
