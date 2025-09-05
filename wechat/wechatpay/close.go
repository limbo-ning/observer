package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type CloseOption struct {
	baseOption
}

type CloseParam struct {
	baseParam
	OutTradeNo string `xml:"out_trade_no"`
}

type CloseRet struct {
	baseRet
}

func Close(orderID, appID, mchID, key string, option CloseOption) (*CloseRet, interface{}, error) {
	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)

	param["out_trade_no"] = orderID

	param["sign"] = Sign(param, key)

	closeParam := new(CloseParam)
	closeParam.AppID = appID
	closeParam.SubAppID = option.SubAppID
	closeParam.MchID = mchID
	closeParam.SubMchID = option.SubMchID
	closeParam.OutTradeNo = orderID
	closeParam.NonceStr = param["nonce_str"].(string)
	closeParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(closeParam)
	if err != nil {
		log.Println("error marshal wechat close order: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/closeorder", bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat close order: ", err)
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

	log.Println("wechat close order ret: ", string(body))

	var closeRet CloseRet
	err = xml.Unmarshal(body, &closeRet)
	if err != nil {
		log.Println("error mashal request wechat close ret: ", err)
		return nil, nil, err
	}

	if closeRet.ReturnCode == "SUCCESS" {
		if closeRet.ResultCode == "SUCCESS" {
			return &closeRet, string(body), nil
		}
		if closeRet.ResultCode == "ORDERCLOSED" {
			return &closeRet, string(body), nil
		}
		return &closeRet, string(body), errors.New(closeRet.ErrCodeDes)
	}
	return nil, nil, errors.New(closeRet.ReturnMsg)
}
