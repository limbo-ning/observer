package alipay

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/util"
)

const success_code = "10000"
const user_input = "10003"
const timeout = "20000"
const not_exists = "40004"

var E_order_not_exist = errors.New("订单未提交")
var E_user_input = errors.New("等待用户操作")
var E_result_unknown = errors.New("支付结果未知")

func ParsePrice(amountInFen int) string {
	str := strconv.FormatInt(int64(amountInFen), 10)
	if len(str) < 3 {
		str = strings.Repeat("0", 3-len(str)) + str
	}

	return str[:len(str)-2] + "." + str[len(str)-2:]
}

func Sign(params map[string]string, key string) string {

	pairs := make([]string, 0)

	for k, v := range params {
		if v == "" {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(pairs)

	var toSign = strings.Join(pairs, "&")
	fmt.Println("alipay content to sign", toSign)

	log.Println("private key", key)
	block, _ := pem.Decode([]byte(fmt.Sprintf(`
-----BEGIN RSA PRIVATE KEY-----
%s
-----END RSA PRIVATE KEY-----
	`, key)))
	if block == nil {
		log.Panic("alipay private key parsing error")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Panic(err)
	}

	h := sha256.New()
	h.Write([]byte(toSign))
	digest := h.Sum(nil)

	s, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, digest)
	if err != nil {
		log.Panic(err)
	}
	signed := base64.StdEncoding.EncodeToString(s)

	log.Println("alipay signed", signed)

	return signed
}

func getPublicParam(appID, method, notifyURL, returnURL, appAuthToken string) map[string]string {
	param := make(map[string]string)

	param["app_id"] = appID
	param["method"] = method
	param["format"] = "JSON"
	param["return_url"] = returnURL
	param["charset"] = "utf-8"
	param["sign_type"] = "RSA2"
	param["timestamp"] = util.FormatDateTime(time.Now())
	param["version"] = "1.0"
	param["notify_url"] = notifyURL
	param["app_auth_token"] = appAuthToken

	return param
}

func parseURL(param map[string]string) string {
	form := url.Values{}
	for k, v := range param {
		if v != "" {
			form.Set(k, v)
		}
	}

	URL := fmt.Sprintf("https://openapi.alipay.com/gateway.do?%s", form.Encode())
	log.Println("alipay request URL", URL)

	return URL
}

func execute(param map[string]string) ([]byte, error) {

	print, _ := json.Marshal(param)
	log.Println("alipay execute: ", string(print))

	URL := parseURL(param)

	client := &http.Client{}

	req, err := http.NewRequest("GET", URL, bytes.NewReader([]byte{}))
	if err != nil {
		log.Println("error request alipay: ", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request alipay: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request alipay: ", err)
		return nil, err
	}

	log.Println("request alipay ret: ", string(body))

	return body, nil
}
