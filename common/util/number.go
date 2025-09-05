package util

import (
	"strconv"
	"strings"
)

func GetAccuracy(input float64) int {
	fltStr := strconv.FormatFloat(input, 'f', -1, 64)
	parts := strings.Split(fltStr, ".")
	if len(parts) == 1 {
		return 0
	}
	return len(parts[1])
}

func ApplyAccuracy(input float64, accuracy int) float64 {
	fltStr := strconv.FormatFloat(input, 'f', accuracy, 64)
	result, _ := strconv.ParseFloat(fltStr, 64)

	return result
}
