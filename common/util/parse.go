package util

import (
	"fmt"
	"strconv"
	"strings"
)

func StringToInts(str, delimeter string) ([]int, error) {
	if str == "" || delimeter == "" {
		return nil, nil
	}

	result := make([]int, 0)

	parts := strings.Split(str, delimeter)
	for _, d := range parts {
		id, err := strconv.Atoi(d)
		if err != nil {
			return nil, fmt.Errorf("%s is not number", d)
		}
		result = append(result, id)
	}

	return result, nil
}
