package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func Mask(s, mask string) string {
	if s == "" || mask == "" || len(s) == 1 {
		return s
	}

	if len(s) == 2 {
		return s[:1] + mask
	}

	toMask := len(s) / 3

	firstRevealEnd := (len(s) - toMask) / 2
	lastRevealStart := firstRevealEnd + toMask

	return s[0:firstRevealEnd] + strings.Repeat(mask, toMask) + s[lastRevealStart:]
}

func Underscore(s string) string {
	var result string
	for i, v := range s {
		if v >= 'A' && v <= 'Z' {
			if i == 0 {
				result = fmt.Sprintf("%s%c", result, v+32)
			} else {
				result = fmt.Sprintf("%s_%c", result, v+32)
			}
		} else {
			result = fmt.Sprintf("%s%c", result, v)
		}
	}
	return result
}

func UnsafeJsonString(obj interface{}) ([]byte, error) {

	bytes := new(bytes.Buffer)

	encoder := json.NewEncoder(bytes)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(obj); err != nil {
		return nil, err
	}

	return bytes.Bytes(), nil

	// data, err := json.Marshal(obj)
	// data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	// data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
	// data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
	// data = bytes.Replace(data, []byte("\\u003d"), []byte("="), -1)

	// return data, err
}
