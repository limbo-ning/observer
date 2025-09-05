package ipc_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"obsessiontech/common/encrypt"
)

func TestLock(t *testing.T) {
	go register()
	register()
	time.Sleep(5 * time.Second)
}

func register() {
	url := "http://education.site4u.design/projectc/stress/user/register/password"

	client := &http.Client{}

	param := map[string]interface{}{
		"username": "locktesting2",
		"password": encrypt.Base64Encrypt("locktesting2"),
	}

	data, _ := json.Marshal(param)

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Println("error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error", err)
		return
	}

	log.Println(string(body))

	return
}
