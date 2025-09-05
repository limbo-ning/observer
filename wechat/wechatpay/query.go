package wechatpay

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	TRADE_STATE_SUCCESS    = "SUCCESS"
	TRADE_STATE_REFUND     = "REFUND"
	TRADE_STATE_NOTPAY     = "NOTPAY"
	TRADE_STATE_CLOSED     = "CLOSED"
	TRADE_STATE_REVOKED    = "REVOKED"
	TRADE_STATE_USERPAYING = "USERPAYING"
	TRADE_STATE_PAYERROR   = "PAYERROR"

	REFUND_STATE_SUCCESS     = "SUCCESS"
	REFUND_STATE_REFUNDCLOSE = "REFUNDCLOSE"
	REFUND_STATE_PROCESSING  = "PROCESSING"
	REFUND_STATE_CHANGE      = "CHANGE"
)

type QueryOption struct {
	baseOption
}

type QueryParam struct {
	baseParam
	OutTradeNo    string `xml:"out_trade_no"`
	TransactionID string `xml:"transaction_id,omitempty"`
}

type QueryRet struct {
	baseRet
	TradeState     string `xml:"trade_state"`
	TradeStateDesc string `xml:"trade_state_desc"`
	OutTradeNo     string `xml:"out_trade_no"`
	TransactionID  string `xml:"transaction_id"`
}

func Query(orderID, transactionID, appID, mchID, key string, option QueryOption) (*QueryRet, interface{}, error) {

	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)

	param["out_trade_no"] = orderID
	param["transaction_id"] = transactionID

	param["sign"] = Sign(param, key)

	queryParam := new(QueryParam)
	queryParam.AppID = appID
	queryParam.SubAppID = option.SubAppID
	queryParam.MchID = mchID
	queryParam.SubMchID = option.SubMchID
	queryParam.OutTradeNo = orderID
	queryParam.TransactionID = transactionID
	queryParam.NonceStr = param["nonce_str"].(string)
	queryParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(queryParam)
	if err != nil {
		log.Println("error marshal wechat query order: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/orderquery", bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat query order: ", err)
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

	log.Println("wechat query order ret: ", string(body))

	var queryRet QueryRet
	err = xml.Unmarshal(body, &queryRet)
	if err != nil {
		log.Println("error mashal request wechat query ret: ", err)
		return nil, nil, err
	}

	if queryRet.ReturnCode == "SUCCESS" {
		if queryRet.ResultCode == "SUCCESS" {
			return &queryRet, string(body), nil
		}
		if queryRet.ResultCode == "ORDERNOTEXIST" {
			return &queryRet, string(body), E_order_not_exists
		}
		return &queryRet, string(body), errors.New(queryRet.ErrCodeDes)
	}
	return nil, nil, errors.New(queryRet.ReturnMsg)
}

type QueryRefundOption struct {
	baseOption
}

type QueryRefundParam struct {
	baseParam
	OutRefundNo string `xml:"out_trade_no"`
	RefundID    string `xml:"refund_id,omitempty"`
}

type QueryRefundRet struct {
	baseRet
	Results []*QueryRefundResult
}

type QueryRefundResult struct {
	OutRefundNo       string
	RefundID          string
	RefundStatus      string
	RefundFee         int
	RefundRecvAccount string
}

func QueryRefund(refundOrderID, refundID, appID, mchID, key string, option QueryRefundOption) (*QueryRefundRet, interface{}, error) {

	param := commonParam(appID, option.SubAppID, mchID, option.SubMchID)

	param["out_refund_no"] = refundOrderID
	param["refund_id"] = refundID

	param["sign"] = Sign(param, key)

	queryParam := new(QueryRefundParam)
	queryParam.AppID = appID
	queryParam.SubAppID = option.SubAppID
	queryParam.MchID = mchID
	queryParam.SubMchID = option.SubMchID
	queryParam.OutRefundNo = refundOrderID
	queryParam.RefundID = refundID
	queryParam.NonceStr = param["nonce_str"].(string)
	queryParam.Sign = param["sign"].(string)

	data, err := xml.Marshal(queryParam)
	if err != nil {
		log.Println("error marshal wechat query refund: ", err)
		return nil, nil, err
	}
	log.Println(string(data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/refundquery", bytes.NewReader(data))
	if err != nil {
		log.Println("error create request wechat query refund: ", err)
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

	log.Println("wechat query refund ret: ", string(body))

	decoder := xml.NewDecoder(bytes.NewReader(body))

	ret := new(QueryRefundRet)
	ret.Results = make([]*QueryRefundResult, 0)
	refundResult := make(map[int]*QueryRefundResult)

	var token xml.Token
	var currentField string

	for {
		token, err = decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Println("error unmarshal wechat query refund ret: ", err)
				return nil, nil, err
			}
		}
		switch token.(type) {
		case xml.StartElement:
			tt := token.(xml.StartElement)
			currentField = tt.Name.Local
		case xml.EndElement:
			currentField = ""
		case xml.CharData:
			tt := token.(xml.CharData)
			switch {
			case currentField == "return_code":
				ret.ReturnCode = string(tt)
			case currentField == "return_msg":
				ret.ReturnMsg = string(tt)
			case currentField == "result_code":
				ret.ResultCode = string(tt)
			case currentField == "err_code":
				ret.ErrCode = string(tt)
			case currentField == "err_code_des":
				ret.ErrCodeDes = string(tt)
			case currentField == "sign":
				ret.Sign = string(tt)
			case currentField == "appid":
				ret.AppID = string(tt)
			case currentField == "sub_appid":
				ret.SubAppID = string(tt)
			case currentField == "mch_id":
				ret.MchID = string(tt)
			case currentField == "sub_mch_id":
				ret.SubMchID = string(tt)
			case currentField == "nonce_str":
				ret.NonceStr = string(tt)
			case strings.HasPrefix(currentField, "out_refund_no_"):
				id, err := strconv.Atoi(currentField[len("out_refund_no_"):])
				if err != nil {
					log.Println("error unmarshal wechat query refund ret: ", err)
					return nil, nil, err
				}
				result, exists := refundResult[id]
				if !exists {
					result = new(QueryRefundResult)
				}
				result.OutRefundNo = string(tt)
				refundResult[id] = result
			case strings.HasPrefix(currentField, "refund_id_"):
				id, err := strconv.Atoi(currentField[len("refund_id_"):])
				if err != nil {
					log.Println("error unmarshal wechat query refund ret: ", err)
					return nil, nil, err
				}
				result, exists := refundResult[id]
				if !exists {
					result = new(QueryRefundResult)
				}
				result.RefundID = string(tt)
				refundResult[id] = result
			case strings.HasPrefix(currentField, "refund_fee_"):
				id, err := strconv.Atoi(currentField[len("refund_fee_"):])
				if err != nil {
					log.Println("error unmarshal wechat query refund ret: ", err)
					return nil, nil, err
				}
				result, exists := refundResult[id]
				if !exists {
					result = new(QueryRefundResult)
				}
				result.RefundFee, _ = strconv.Atoi(string(tt))
				refundResult[id] = result
			case strings.HasPrefix(currentField, "refund_status_"):
				id, err := strconv.Atoi(currentField[len("refund_status_"):])
				if err != nil {
					log.Println("error unmarshal wechat query refund ret: ", err)
					return nil, nil, err
				}
				result, exists := refundResult[id]
				if !exists {
					result = new(QueryRefundResult)
				}
				result.RefundStatus = string(tt)
				refundResult[id] = result
			case strings.HasPrefix(currentField, "refund_recv_account_"):
				id, err := strconv.Atoi(currentField[len("refund_recv_account_"):])
				if err != nil {
					log.Println("error unmarshal wechat query refund ret: ", err)
					return nil, nil, err
				}
				result, exists := refundResult[id]
				if !exists {
					result = new(QueryRefundResult)
				}
				result.RefundRecvAccount = string(tt)
				refundResult[id] = result
			}
		}
	}

	for _, result := range refundResult {
		ret.Results = append(ret.Results, result)
	}

	if ret.ReturnCode == "SUCCESS" {
		if ret.ResultCode == "SUCCESS" {
			return ret, string(body), nil
		}
		if ret.ResultCode == "REFUNDNOTEXIST" {
			return ret, string(body), E_order_not_exists
		}
		return ret, string(body), errors.New(ret.ErrCodeDes)
	}
	return nil, nil, errors.New(ret.ReturnMsg)
}
