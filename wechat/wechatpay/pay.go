package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

const PAY_TYPE_MICROPAY = "MICROPAY"
const PAY_TYPE_JSAPI = "JSAPI"
const PAY_TYPE_NATIVE = "NATIVE"
const PAY_TYPE_H5 = "MWEB"
const PAY_TYPE_APP = "APP"

type UnifiedOrderOption struct {
	baseOption
	OpenID      string
	SubOpenID   string
	ProductID   string
	PayAuthCode string
}

type OrderParam struct {
	baseParam
	Body           string `xml:"body"`
	OutTradeNo     string `xml:"out_trade_no"`
	ProductID      string `xml:"product_id,omitempty"`
	TotalFee       int    `xml:"total_fee"`
	SpbillCreateIP string `xml:"spbill_create_ip"`
	NotifyURL      string `xml:"notify_url"`
	TradeType      string `xml:"trade_type"`
	OpenID         string `xml:"openid,omitempty"`
	SubOpenID      string `xml:"sub_openid,omitempty"`
	AuthCode       string `xml:"auth_code,omitempty"`
}

type OrderRet struct {
	baseRet
	TradeType string `xml:"trade_type"`
	PrepayID  string `xml:"prepay_id"`
	MWebURL   string `xml:"mweb_url"`
	CodeURL   string `xml:"code_url"`
}

func OpenUnifiedOrder(orderID, description, clientIP, tradeType, openID string, amountInFen int) (*OrderRet, interface{}, error) {
	return UnifiedOrder(orderID, description, clientIP, tradeType, amountInFen, Config.WechatAppID, Config.WechatPayMchID, Config.WechatPayKey, Config.WechatPayNotifyURL, UnifiedOrderOption{OpenID: openID})
}

func MiniAppOrder(orderID, description, clientIP, openID string, amountInFen int) (*OrderRet, interface{}, error) {
	return UnifiedOrder(orderID, description, clientIP, PAY_TYPE_JSAPI, amountInFen, Config.WechatMiniAppID, Config.WechatPayMchID, Config.WechatPayKey, Config.WechatPayNotifyURL, UnifiedOrderOption{OpenID: openID})
}

func UnifiedOrder(orderID, description, clientIP, tradeType string, amountInFen int, appID, mchID, mchKey, notifyURL string, option UnifiedOrderOption) (*OrderRet, interface{}, error) {

	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)
	param["body"] = description
	param["out_trade_no"] = orderID
	param["total_fee"] = amountInFen
	param["spbill_create_ip"] = clientIP
	if tradeType != PAY_TYPE_MICROPAY {
		param["notify_url"] = notifyURL
		param["trade_type"] = tradeType
		param["openid"] = option.OpenID
		param["sub_openid"] = option.SubOpenID
	} else {
		param["auth_code"] = option.PayAuthCode
	}
	param["product_id"] = option.ProductID

	param["sign"] = Sign(param, mchKey)

	orderParam := new(OrderParam)
	orderParam.AppID = appID
	orderParam.SubAppID = option.SubAppID
	orderParam.MchID = mchID
	orderParam.SubMchID = option.SubMchID
	orderParam.NonceStr = param["nonce_str"].(string)
	orderParam.Body = description
	orderParam.OutTradeNo = orderID
	orderParam.ProductID = option.ProductID
	orderParam.TotalFee = amountInFen
	orderParam.SpbillCreateIP = clientIP
	if tradeType != PAY_TYPE_MICROPAY {
		orderParam.NotifyURL = notifyURL
		orderParam.TradeType = tradeType
		orderParam.OpenID = option.OpenID
		orderParam.SubOpenID = option.SubOpenID
	} else {
		orderParam.AuthCode = option.PayAuthCode
	}
	orderParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(orderParam)
	if err != nil {
		log.Println("error marshal wechat pay order: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client := &http.Client{}

	var URL string

	switch tradeType {
	case PAY_TYPE_MICROPAY:
		URL = "https://api.mch.weixin.qq.com/pay/micropay"
	default:
		URL = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	}

	req, err := http.NewRequest("POST", URL, bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat pay order: ", err)
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error wechat pay order:", err)
		return nil, nil, E_result_unknown
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error wechat pay order:", err)
		return nil, nil, E_result_unknown
	}

	var orderRet OrderRet
	err = xml.Unmarshal(body, &orderRet)
	if err != nil {
		log.Println("error mashal request wechat pay ret: ", err)
		return nil, nil, E_result_unknown
	}

	log.Println("wechat pay ret:", string(body))
	if orderRet.ReturnCode == "SUCCESS" {
		if orderRet.ResultCode == "SUCCESS" {
			return &orderRet, string(body), nil
		}
		switch orderRet.ErrCode {
		case "USERPAYING":
			return &orderRet, string(body), E_user_input
		case "SYSTEMERROR":
			return &orderRet, string(body), E_result_unknown
		}
		return &orderRet, string(body), errors.New(orderRet.ErrCodeDes)
	} else {
		return nil, nil, errors.New(orderRet.ReturnMsg)
	}
}
