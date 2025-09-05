package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type RefundParam struct {
	baseParam
	OutTradeNo    string `xml:"out_trade_no"`
	TotalFee      int    `xml:"total_fee"`
	NotifyURL     string `xml:"notify_url"`
	TransactionID string `xml:"transaction_id,omityempty"`
	OutRefundNo   string `xml:"out_refund_no"`
	RefundFee     int    `xml:"refund_fee"`
	RefundDesc    string `xml:"refund_desc,omityempty"`
}

type RefundOption struct {
	baseOption
	RefundDesc string
}

type RefundRet struct {
	baseRet
	OutRefundNo   string `xml:"out_refund_no"`
	RefundID      string `xml:"refund_id"`
	RefundFee     int    `xml:"refund_fee"`
	CashFee       int    `xml:"cash_fee"`
	CashRefundFee int    `xml:"cash_refund_fee"`
}

func Refund(orderID, transactionID, refundOrderID string, orderAmountInFen, refundAmountInFen int, appID, mchID, mchKey, notifyURL, cert, key string, option RefundOption) (*RefundRet, interface{}, error) {

	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)

	param["refund_desc"] = option.RefundDesc
	param["out_trade_no"] = orderID
	param["out_refund_no"] = refundOrderID
	param["total_fee"] = orderAmountInFen
	param["refund_fee"] = refundAmountInFen
	param["transaction_id"] = transactionID
	param["notify_url"] = notifyURL

	param["sign"] = Sign(param, mchKey)

	refundParam := new(RefundParam)
	refundParam.AppID = appID
	refundParam.SubAppID = option.SubAppID
	refundParam.MchID = mchID
	refundParam.SubMchID = option.SubMchID
	refundParam.NonceStr = param["nonce_str"].(string)
	refundParam.OutTradeNo = orderID
	refundParam.TransactionID = transactionID
	refundParam.OutRefundNo = refundOrderID
	refundParam.RefundDesc = option.RefundDesc
	refundParam.TotalFee = orderAmountInFen
	refundParam.RefundFee = refundAmountInFen
	refundParam.NotifyURL = notifyURL
	refundParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(refundParam)
	if err != nil {
		log.Println("error marshal wechat refund order: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client, err := getTlsClient(cert, key)
	if err != nil {
		log.Println("error get tls client: ", err)
		return nil, nil, err
	}

	var retryCount int

process:
	retryCount++

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/secapi/pay/refund", bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat refund order: ", err)
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error wechat refund order: ", err)
		if retryCount < 3 {
			goto process
		}
		return nil, nil, E_result_unknown
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error wechat refund order: ", err)
		if retryCount < 3 {
			goto process
		}
		return nil, nil, E_result_unknown
	}

	log.Println("wechat refund order ret: ", string(body))

	var refundRet RefundRet
	err = xml.Unmarshal(body, &refundRet)
	if err != nil {
		log.Println("error mashal request wechat refund ret: ", err)
		if retryCount < 3 {
			goto process
		}
		return nil, nil, E_result_unknown
	}

	log.Printf("wechat pay ret: %+v", refundRet)
	if refundRet.ReturnCode == "SUCCESS" {
		if refundRet.ResultCode == "SUCCESS" {
			return &refundRet, string(body), nil
		}

		switch refundRet.ResultCode {
		case "BIZERR_NEED_RETRY":
			fallthrough
		case "SYSTEMERROR":
			if retryCount < 3 {
				goto process
			}
			return &refundRet, string(body), E_retry
		}
		return &refundRet, string(body), errors.New(refundRet.ErrCodeDes)
	}

	return nil, nil, errors.New(refundRet.ReturnMsg)
}
