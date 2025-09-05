package alipay

import (
	"encoding/json"
	"errors"
)

type CloseResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	SubCode    string `json:"sub_code"`
	SubMsg     string `json:"sub_msg"`
	TradeNo    string `json:"tradeNo"`
	OutTradeNo string `json:"out_trade_no"`
}

type CloseRet struct {
	Response *CloseResponse `json:"alipay_trade_close_response"`
	Sign     string         `json:"sign"`
}

func Close(orderID, tradeNo, appID, key, notifyURL, appAuthToken string) (*CloseResponse, []byte, error) {
	param := getPublicParam(appID, "alipay.trade.close", notifyURL, "", appAuthToken)

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

	var ret CloseRet
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}

	if ret.Response.Code == not_exists {
		return ret.Response, data, nil
	}

	if ret.Response.Code != success_code {
		return nil, nil, errors.New(ret.Response.Msg)
	}

	return ret.Response, data, nil
}
