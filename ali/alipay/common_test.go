package alipay_test

import (
	"testing"

	"obsessiontech/ali/alipay"
)

func TestParsePrice(t *testing.T) {
	t.Run("0.10", func(t *testing.T) {
		price := alipay.ParsePrice(10)
		if price != "0.10" {
			t.Error("should be 0.10:", price)
		}
	})
	t.Run("0.05", func(t *testing.T) {
		price := alipay.ParsePrice(5)
		if price != "0.05" {
			t.Error("should be 0.05:", price)
		}
	})
	t.Run("1.00", func(t *testing.T) {
		price := alipay.ParsePrice(100)
		if price != "1.00" {
			t.Error("should be 1.00:", price)
		}
	})
	t.Run("214111.04", func(t *testing.T) {
		price := alipay.ParsePrice(21411104)
		if price != "214111.04" {
			t.Error("should be 214111.04:", price)
		}
	})
}
