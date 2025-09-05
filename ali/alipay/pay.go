package alipay

import (
	"encoding/json"
	"errors"
)

const (
	WAP     = "WAP"
	BARCODE = "BARCODE"
	QRCODE  = "QRCODE"
	PC      = "PC"
)

func WapPay(subject, orderID, totalAmount string) string {
	param := getPublicParam(Config.AlipayAppID, "alipay.trade.wap.pay", Config.AlipayNotifyURL, Config.AlipayWapPayReturnURL, "")

	bizContent := make(map[string]interface{})
	bizContent["subject"] = subject
	bizContent["out_trade_no"] = orderID
	bizContent["total_amount"] = totalAmount
	bizContent["product_code"] = "QUICK_WAP_WAY"

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, Config.AlipayPrivateKey)

	return parseURL(param)
}

type FundBill struct {
	FundChannel string `json:"fund_channel"`
	BankCode    string `json:"bank_code"`
	Amount      string `json:"amount"`
	RealAmount  string `json:"real_amount"`
}

type BarCodePayResponse struct {
	Code          string      `json:"code"`
	Msg           string      `json:"msg"`
	SubCode       string      `json:"sub_code"`
	SubMsg        string      `json:"sub_msg"`
	TradeNo       string      `json:"tradeNo"`
	OutTradeNo    string      `json:"out_trade_no"`
	BuyerLogonID  string      `json:"buyer_logon_id"`
	TotalAmount   string      `json:"total_amount"`
	ReceiptAmount string      `json:"receipt_amount"`
	GMTPayment    string      `json:"gmt_payment"`
	FundBillList  []*FundBill `json:"fund_bill_list"`
	BuyerUserID   string      `json:"buyer_user_id"`
}

type BarCodePayRet struct {
	Response *BarCodePayResponse `json:"alipay_trade_pay_response"`
	Sign     string              `json:"sign"`
}

func BarCodePay(orderID, subject, totalAmount, payAuthCode, appID, key, notifyURL, appAuthToken string) (*BarCodePayResponse, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.pay", notifyURL, "", appAuthToken)

	bizContent := make(map[string]interface{})
	bizContent["subject"] = subject
	bizContent["scene"] = "bar_code"
	bizContent["auth_code"] = payAuthCode
	bizContent["out_trade_no"] = orderID
	bizContent["total_amount"] = totalAmount

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, key)

	data, err := execute(param)
	if err != nil {
		return nil, nil, err
	}

	var ret BarCodePayRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}
	if ret.Response.Code != success_code {
		switch ret.Response.Code {
		case user_input:
			return ret.Response, data, E_user_input
		case timeout:
			return ret.Response, data, E_result_unknown
		}
		return ret.Response, data, errors.New(ret.Response.Msg)
	}

	return ret.Response, data, nil
}

type QRCodePayResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	SubCode    string `json:"sub_code"`
	SubMsg     string `json:"sub_msg"`
	OutTradeNo string `json:"out_trade_no"`
	QRCode     string `json:"qr_code"`
}

type QRCodePayRet struct {
	Response *QRCodePayResponse `json:"alipay_trade_precreate_response"`
	Sign     string             `json:"sign"`
}

func QRCodePay(orderID, subject, totalAmount, appID, key, notifyURL, appAuthToken string) (string, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.precreate", notifyURL, "", appAuthToken)

	bizContent := make(map[string]interface{})
	bizContent["subject"] = subject
	bizContent["out_trade_no"] = orderID
	bizContent["total_amount"] = totalAmount

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, key)

	data, err := execute(param)
	if err != nil {
		return "", nil, err
	}

	var ret QRCodePayRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return "", nil, err
	}
	if ret.Response.Code != success_code {
		return "", nil, errors.New(ret.Response.Msg)
	}
	if ret.Response.OutTradeNo != orderID {
		return "", nil, errors.New("支付宝返回订单ID不匹配")
	}

	return ret.Response.QRCode, data, nil
}
