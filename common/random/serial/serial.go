package serial

import (
	"bytes"
	"encoding/binary"
	"math"
	"obsessiontech/common/config"
	"obsessiontech/common/random"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var Config struct {
	MachineCode int64
}

func init() {
	config.GetConfig("config.yaml", &Config)
}

var increment uint32

func GenerateSerial() string {
	bytesBuffer := bytes.NewBuffer([]byte{})

	i := atomic.AddUint32(&increment, 1)

	if i > math.MaxUint16 {
		atomic.CompareAndSwapUint32(&increment, i, 0)
	}
	now := time.Now()
	result := now.Format("20060102")

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(now.Sub(now.Truncate(24*time.Hour)).Seconds()))

	if err := binary.Write(bytesBuffer, binary.BigEndian, ts[5:]); err != nil {
		panic(err)
	}

	if err := binary.Write(bytesBuffer, binary.BigEndian, uint16(i)); err != nil {
		panic(err)
	}
	if err := binary.Write(bytesBuffer, binary.BigEndian, uint8(Config.MachineCode)); err != nil {
		panic(err)
	}
	if err := binary.Write(bytesBuffer, binary.BigEndian, uint16(random.GetRandomNumber(math.MaxInt16))); err != nil {
		panic(err)
	}

	num := binary.BigEndian.Uint64(bytesBuffer.Bytes())
	numformatted := strconv.FormatUint(num, 10)

	if len(numformatted) < 17 {
		numformatted = strings.Repeat("0", 17-len(numformatted)) + numformatted
	}

	return result + numformatted
}
