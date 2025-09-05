package alipay

import (
	"encoding/json"
	"errors"
	"strings"
)

type QueryResponse struct {
	Code         string      `json:"code"`
	Msg          string      `json:"msg"`
	SubCode      string      `json:"sub_code"`
	SubMsg       string      `json:"sub_msg"`
	TradeNo      string      `json:"tradeNo"`
	OutTradeNo   string      `json:"out_trade_no"`
	BuyerLogonID string      `json:"buyer_logon_id"`
	TradeStatus  string      `json:"trade_status"`
	TotalAmount  string      `json:"total_amount"`
	BuyerUserID  string      `json:"buyer_user_id"`
	FundBillList []*FundBill `json:"fund_bill_list"`
}

type QueryRet struct {
	Response *QueryResponse `json:"alipay_trade_query_response"`
	Sign     string         `json:"sign"`
}

func Query(orderID, tradeNo, appID, key, appAuthToken string) (*QueryResponse, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.query", "", "", appAuthToken)

	bizContent := make(map[string]interface{})
	bizContent["out_trade_no"] = orderID
	bizContent["trade_no"] = tradeNo

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, key)

	data, err := execute(param)
	if err != nil {
		return nil, nil, err
	}

	var ret QueryRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}
	if ret.Response.Code != success_code {
		if strings.EqualFold(ret.Response.SubCode, "ACQ.TRADE_NOT_EXIST") || strings.EqualFold(ret.Response.SubCode, "TRADE_NOT_EXIST") {
			return nil, nil, E_order_not_exist
		}
		return nil, nil, errors.New(ret.Response.Msg)
	}

	return ret.Response, data, nil
}

type RefundQueryResponse struct {
	Code         string `json:"code"`
	Msg          string `json:"msg"`
	SubCode      string `json:"sub_code"`
	SubMsg       string `json:"sub_msg"`
	TradeNo      string `json:"tradeNo"`
	OutTradeNo   string `json:"out_trade_no"`
	OutRequestNo string `json:"out_request_no"`
	RefundReason string `json:"refund_reason"`
	TotalAmount  string `json:"total_amount"`
	RefundAmount string `json:"refund_amount"`
}

type RefundQueryRet struct {
	Response *RefundQueryResponse `json:"alipay_trade_fastpay_refund_query_response"`
	Sign     string               `json:"sign"`
}

func RefundQuery(orderID, tradeNo, outRequestNo, appID, key, appAuthToken string) (*RefundQueryResponse, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.fastpay.refund.query", "", "", appAuthToken)

	bizContent := make(map[string]interface{})
	bizContent["out_trade_no"] = orderID
	bizContent["trade_no"] = tradeNo
	bizContent["out_request_no"] = outRequestNo

	bizContentData, _ := json.Marshal(bizContent)

	param["biz_content"] = string(bizContentData)

	param["sign"] = Sign(param, key)

	data, err := execute(param)
	if err != nil {
		return nil, nil, err
	}

	var ret RefundQueryRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}
	if ret.Response.Code != success_code {
		if strings.EqualFold(ret.Response.SubCode, "ACQ.TRADE_NOT_EXIST") || strings.EqualFold(ret.Response.SubCode, "TRADE_NOT_EXIST") {
			return nil, nil, E_order_not_exist
		}
		return nil, nil, errors.New(ret.Response.Msg)
	}

	return ret.Response, data, nil
}
