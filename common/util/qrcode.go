package util

import (
	"fmt"
	"net/url"
	"net/http"
	"io/ioutil"
)

func GetQrCode(content string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("http://qr.liantu.com/api.php?text=%s", url.QueryEscape(content)))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}