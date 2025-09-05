package util_test

import (
	"crypto/sha1"
	"fmt"
	"testing"
)

func TestSha1(t *testing.T) {
	var toSign = "jsapi_ticket=HoagFKDcsGMVCIY2vOjf9rE-4Rf8_rMgGa9iahRIrnFflrGNBHEmafSx_CsZlcBYv0a7xu8wi2fy48Y1ivIowg&noncestr=ydkUJBweVTVxNKzT&timestamp=1534324313&url=https://ssl.179yule.com/jdh5/?from=singlemessage&isappinstalled=0"
	s := sha1.New()
	s.Write([]byte(toSign))

	sd := fmt.Sprintf("%x", s.Sum(nil))

	if sd != "013daef3fb5c5ba66c9a5438a2aa025bea9b9d62" {
		t.Error("wrong signature")
	}
}
