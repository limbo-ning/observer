package alipay

import (
	"encoding/json"
	"errors"
)

type RefundResponse struct {
	Code         string      `json:"code"`
	Msg          string      `json:"msg"`
	SubCode      string      `json:"sub_code"`
	SubMsg       string      `json:"sub_msg"`
	TradeNo      string      `json:"tradeNo"`
	OutTradeNo   string      `json:"out_trade_no"`
	BuyerLogonID string      `json:"buyer_logon_id"`
	FundChange   string      `json:"fund_change"`
	RefundFee    string      `json:"refund_fee"`
	GMTPayment   string      `json:"gmt_payment"`
	FundBillList []*FundBill `json:"fund_bill_list"`
	BuyerUserID  string      `json:"buyer_user_id"`
	GMTRefundPay string      `json:"gmt_refund_pay"`
}

type RefundRet struct {
	Response *RefundResponse `json:"alipay_trade_refund_response"`
	Sign     string          `json:"sign"`
}

func Refund(orderID, refundAmount, refundSerial, appID, key, reason, appAuthToken string) (*RefundResponse, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.refund", "", "", appAuthToken)

	bizContent := make(map[string]interface{})
	bizContent["out_trade_no"] = orderID
	bizContent["refund_amount"] = refundAmount
	bizContent["out_request_no"] = refundSerial
	bizContent["refund_reason"] = reason

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, key)

	data, err := execute(param)
	if err != nil {
		return nil, nil, err
	}

	var ret RefundRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}
	if ret.Response.Code != success_code {
		return nil, nil, errors.New(ret.Response.Msg)
	}

	return ret.Response, data, nil
}
