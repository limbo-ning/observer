package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"

	"obsessiontech/common/random"
)

var PAY_TYPE_JSAPI = "JSAPI"

type OrderParam struct {
	XMLName        xml.Name `xml:"xml"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	Body           string   `xml:"body"`
	OutTradeNo     string   `xml:"out_trade_no"`
	TotalFee       int      `xml:"total_fee"`
	SpbillCreateIp string   `xml:"spbill_create_ip"`
	NotifyURL      string   `xml:"notify_url"`
	TradeType      string   `xml:"trade_type"`
	OpenID         string   `xml:"openid"`
}

type OrderRet struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	AppID      string `xml:"appid"`
	MchID      string `xml:"mch_id"`
	NonceStr   string `xml:"nonce_str"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	ErrCode    string `xml:"err_code"`
	ErrCodeDes string `xml:"err_code_des"`
	TradeType  string `xml:"trade_type"`
	PrepayID   string `xml:"prepay_id"`
}

func Sign(params map[string]interface{}) string {
	pairs := make([]string, 0)

	for k, v := range params {
		if vs, ok := v.(string); ok {
			if vs == "" {
				continue
			}
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(pairs)

	pairs = append(pairs, fmt.Sprintf("key=%s", Config.WechatKey))

	var toSign = strings.Join(pairs, "&")

	log.Println("wechat content to sign", toSign)

	signed := fmt.Sprintf("%x", md5.Sum([]byte(toSign)))
	signed = strings.ToUpper(signed)

	log.Println("wechat signed", signed)

	return signed
}

func UnifiedOrder(orderID, description, clientIp, tradeType, openID string, amountInFen int) (bool, string) {

	param := make(map[string]interface{})
	param["appid"] = Config.WechatAppID
	param["mch_id"] = Config.WechatPayMchID
	param["nonce_str"] = random.GenerateNonce(16)
	param["body"] = description
	param["out_trade_no"] = orderID
	param["total_fee"] = amountInFen
	param["spbill_create_ip"] = clientIp
	param["notify_url"] = Config.WechatPayNotifyURL
	param["trade_type"] = tradeType

	if tradeType == PAY_TYPE_JSAPI {
		param["openid"] = openID
	}

	param["sign"] = Sign(param)

	orderParam := OrderParam{
		AppID:          param["appid"].(string),
		MchID:          param["mch_id"].(string),
		NonceStr:       param["nonce_str"].(string),
		Body:           param["body"].(string),
		OutTradeNo:     param["out_trade_no"].(string),
		TotalFee:       param["total_fee"].(int),
		SpbillCreateIp: param["spbill_create_ip"].(string),
		NotifyURL:      param["notify_url"].(string),
		TradeType:      param["trade_type"].(string),
		OpenID:         param["openid"].(string),
		Sign:           param["sign"].(string),
	}

	data, err := xml.Marshal(orderParam)
	if err != nil {
		log.Panic(err)
	}
	log.Println(string(data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader(data))

	if err != nil {
		log.Panic(err)
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var orderRet OrderRet
	err = xml.Unmarshal(body, &orderRet)
	if err != nil {
		log.Panic(err)
	}

	if orderRet.ReturnCode == "SUCCESS" {
		if orderRet.ResultCode == "SUCCESS" {
			return true, orderRet.PrepayID
		} else {
			return false, orderRet.ErrCodeDes
		}
	} else {
		return false, orderRet.ReturnMsg
	}
}

type ConfirmData struct {
	ReturnCode    string `xml:"return_code"`
	ReturnMsg     string `xml:"return_msg"`
	AppID         string `xml:"appid"`
	MchID         string `xml:"mch_id"`
	NonceStr      string `xml:"nonce_str"`
	Sign          string `xml:"sign"`
	ResultCode    string `xml:"result_code"`
	ErrCode       string `xml:"err_code"`
	ErrCodeDes    string `xml:"err_code_des"`
	OpenID        string `xml:"openid"`
	TradeType     string `xml:"trade_type"`
	PrepayID      string `xml:"prepay_id"`
	IsSubscribe   string `xml:"is_subscribe"`
	BankType      string `xml:"bank_type"`
	TotalFee      int    `xml:"total_fee"`
	CashFee       int    `xml:"cash_fee"`
	TransactionID string `xml:"transaction_id"`
	OutTradeNo    string `xml:"out_trade_no"`
	TimeEnd       string `xml:"time_end"`
}

type PayConfirmResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
}

func PayConfirm(data []byte) (*ConfirmData, []byte) {
	var confirmData ConfirmData
	err := xml.Unmarshal(data, &confirmData)

	response := PayConfirmResponse{}

	if err != nil {
		log.Println(err)
		response.ReturnCode = "FAIL"
	} else {
		response.ReturnCode = "SUCCESS"
	}

	responseXml, _ := xml.Marshal(response)

	return &confirmData, responseXml
}
