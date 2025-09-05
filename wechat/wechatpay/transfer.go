package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"obsessiontech/common/random"
)

type CheckName string

const RETRY_TIMEOUT = time.Second * 5
const NO_CHECK_NAME = CheckName("NO_CHECK")
const FORCE_CHECK_NAME = CheckName("FORCE_CHECK")

type TransferParam struct {
	XMLName        xml.Name `xml:"xml"`
	MchAppID       string   `xml:"mch_appid"`
	MchID          string   `xml:"mchid"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	PartnerTradeNo string   `xml:"partner_trade_no"`
	Amount         int      `xml:"amount"`
	CheckName      string   `xml:"check_name"`
	ReUserName     string   `xml:"re_user_name,omitempty"`
	SpbillCreateIp string   `xml:"spbill_create_ip"`
	OpenID         string   `xml:"openid"`
	Desc           string   `xml:"desc"`
}

type TransferRet struct {
	ReturnCode     string `xml:"return_code"`
	ReturnMsg      string `xml:"return_msg"`
	MchAppID       string `xml:"mch_appid"`
	MchID          string `xml:"mchid"`
	NonceStr       string `xml:"nonce_str"`
	Sign           string `xml:"sign"`
	ResultCode     string `xml:"result_code"`
	ErrCode        string `xml:"err_code"`
	ErrCodeDes     string `xml:"err_code_des"`
	PartnerTradeNo string `xml:"partner_trade_no"`
	PaymentNo      string `xml:"payment_no"`
	PaymentTime    string `xml:"payment_time"`
}

func MiniAppTransfer(orderID, description, clientIp, openID string, amountInFen int, checkName CheckName, userName string) (*TransferRet, error) {
	return transfer(orderID, description, clientIp, openID, Config.WechatMiniAppID, amountInFen, checkName, userName)
}

func transfer(orderID, description, clientIp, openID, appID string, amountInFen int, checkName CheckName, userName string) (*TransferRet, error) {
	param := make(map[string]interface{})
	param["mch_appid"] = appID
	param["mchid"] = Config.WechatPayMchID
	param["nonce_str"] = random.GenerateNonce(16)
	param["desc"] = description
	param["partner_trade_no"] = orderID
	param["amount"] = amountInFen
	param["spbill_create_ip"] = clientIp
	param["check_name"] = string(checkName)
	param["openid"] = openID
	param["re_user_name"] = userName

	param["sign"] = Sign(param, Config.WechatPayKey)

	transferParam := TransferParam{
		MchAppID:       param["mch_appid"].(string),
		MchID:          param["mchid"].(string),
		NonceStr:       param["nonce_str"].(string),
		Desc:           param["desc"].(string),
		PartnerTradeNo: param["partner_trade_no"].(string),
		Amount:         param["amount"].(int),
		SpbillCreateIp: param["spbill_create_ip"].(string),
		CheckName:      param["check_name"].(string),
		ReUserName:     param["re_user_name"].(string),
		OpenID:         param["openid"].(string),
		Sign:           param["sign"].(string),
	}

	data, err := xml.Marshal(transferParam)
	if err != nil {
		log.Println("error marshal wechat transfer order: ", err)
		return nil, err
	}
	log.Println(string(data))

	cert, err := ioutil.ReadFile(Config.WechatPayMchCertPath)
	if err != nil {
		return nil, err
	}
	certKey, err := ioutil.ReadFile(Config.WechatPayMchCertKeyPath)
	if err != nil {
		return nil, err
	}

	client, err := getTlsClientByCertData(cert, certKey)
	if err != nil {
		log.Println("error get tls client: ", err)
		return nil, err
	}

	var retryCount = 0
	var process func() (*TransferRet, error)
	process = func() (*TransferRet, error) {
		req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/mmpaymkttransfers/promotion/transfers", bytes.NewReader(data))
		if err != nil {
			log.Println("error create request wechat transfer order: ", err)
			return nil, err
		}

		req.Header.Set("Content-Type", "application/xml")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var transferRet TransferRet
		err = xml.Unmarshal(body, &transferRet)
		if err != nil {
			log.Println("error mashal request wechat transfer ret: ", err)
			return nil, err
		}

		log.Printf("wechat transfer ret: %+v", transferRet)
		if transferRet.ReturnCode == "SUCCESS" {
			if transferRet.ResultCode == "SUCCESS" {
				return &transferRet, nil
			} else {
				log.Printf("wechat transfer ret fail: param[%+v] ret[%+v]", transferParam, transferRet.ErrCodeDes)
				if transferRet.ErrCode == "SYSTEMERROR" {
					retryCount++
					log.Printf("wechat transfer system error retry: %d", retryCount)
					time.Sleep(RETRY_TIMEOUT)
					return process()
				}
				return &transferRet, errors.New(transferRet.ErrCodeDes)
			}
		} else {
			log.Printf("wechat transfer ret fail: param[%+v] ret[%+v]", transferParam, transferRet.ResultCode)
			if transferRet.ErrCode == "SYSTEMERROR" {
				retryCount++
				log.Printf("wechat transfer system error retry: %d", retryCount)
				time.Sleep(RETRY_TIMEOUT)
				return process()
			}
			return &transferRet, errors.New(transferRet.ReturnMsg)
		}
	}

	return process()
}
