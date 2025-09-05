package notify

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"obsessiontech/common/random"
)

func encode(d string) string {
	ret := url.QueryEscape(d)
	ret = strings.Replace(ret, "+", "%20", -1)
	ret = strings.Replace(ret, "*", "%2A", -1)
	ret = strings.Replace(ret, "%7E", "~", -1)
	return ret
}

func sign(httpMethod string, params map[string]string) (string, string) {

	params["AccessKeyId"] = Config.AliAccessKey
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureNonce"] = random.GenerateNonce(32)
	params["SignatureVersion"] = "1.0"

	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["Format"] = "XML"

	keys := make([]string, 0)

	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	pairs := make([]string, 0)

	for k, v := range params {
		if v == "" {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", encode(k), encode(v)))
	}

	sort.Strings(pairs)

	paramString := strings.Join(pairs, "&")
	toSign := fmt.Sprintf("%s&%%2F&%s", httpMethod, encode(paramString))

	log.Println("ali notify to sign", toSign)

	key := []byte(Config.AliAccessSecret + "&")
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(toSign))

	signed := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	log.Println("ali notify signed", signed)

	return paramString, encode(signed)
}

func sendToGateway(httpMethod string, param map[string]string) ([]byte, error) {

	paramByte, _ := json.Marshal(param)
	log.Println(string(paramByte))

	requestString, signature := sign(httpMethod, param)

	URL := fmt.Sprintf("http://dysmsapi.aliyuncs.com/?Signature=%s&%s", signature, requestString)

	log.Println(URL)

	client := &http.Client{}

	req, err := http.NewRequest(httpMethod, URL, nil)
	if err != nil {
		log.Println("error request ali gw: ", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request ali gw: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request ali gw: ", err)
		return nil, err
	}

	return body, nil
}
