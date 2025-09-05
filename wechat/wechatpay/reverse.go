package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type ReverseParam struct {
	baseParam
	OutTradeNo    string `xml:"out_trade_no"`
	TransactionID string `xml:"transaction_id"`
}

type ReverseOption struct {
	baseOption
}

type ReverseRet struct {
	baseRet
	Recall string `xml:"recall"`
}

func Reverse(orderID, transactionID, appID, mchID, key string, option ReverseOption) (*ReverseRet, interface{}, error) {
	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)

	param["out_trade_no"] = orderID
	param["transaction_id"] = transactionID

	param["sign"] = Sign(param, key)

	reverseParam := new(ReverseParam)
	reverseParam.AppID = appID
	reverseParam.SubAppID = option.SubAppID
	reverseParam.MchID = mchID
	reverseParam.SubMchID = option.SubMchID
	reverseParam.OutTradeNo = orderID
	reverseParam.TransactionID = transactionID
	reverseParam.NonceStr = param["nonce_str"].(string)
	reverseParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(reverseParam)
	if err != nil {
		log.Println("error marshal wechat reverse order: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/secapi/pay/reverse", bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat reverse order: ", err)
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	log.Println("wechat reverse order ret: ", string(body))

	var reverseRet ReverseRet
	err = xml.Unmarshal(body, &reverseRet)
	if err != nil {
		log.Println("error mashal request wechat reverse ret: ", err)
		return nil, nil, err
	}

	if reverseRet.ReturnCode == "SUCCESS" {
		if reverseRet.ResultCode == "SUCCESS" {
			return &reverseRet, string(body), nil
		}
		if reverseRet.ResultCode == "SYSTEMERROR" {
			return &reverseRet, string(body), E_result_unknown
		}
		return &reverseRet, string(body), errors.New(reverseRet.ErrCodeDes)
	}
	return nil, nil, errors.New(reverseRet.ReturnMsg)
}
