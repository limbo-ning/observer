package util

import "encoding/json"

func Clone(source, destination interface{}) error {

	copy, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(copy, destination)
}
