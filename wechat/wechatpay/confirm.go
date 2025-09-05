package wechatpay

import (
	"encoding/xml"
	"log"
)

type ConfirmData struct {
	baseRet
	ReqInfo             string `xml:"req_info"`
	OpenID              string `xml:"openid"`
	TradeType           string `xml:"trade_type"`
	PrepayID            string `xml:"prepay_id"`
	IsSubscribe         string `xml:"is_subscribe"`
	BankType            string `xml:"bank_type"`
	TotalFee            int    `xml:"total_fee"`
	CashFee             int    `xml:"cash_fee"`
	TransactionID       string `xml:"transaction_id"`
	OutTradeNo          string `xml:"out_trade_no"`
	TimeEnd             string `xml:"time_end"`
	OutRefundNo         string `xml:"out_refund_no"`
	RefundID            string `xml:"refund_id"`
	RefundFee           int    `xml:"refund_fee"`
	SettlementRefundFee int    `xml:"settlement_refund_fee"`
	RefundStatus        string `xml:"refund_status"`
	RefundRecvAccount   string `xml:"refund_recv_account"`
	RefundAccount       string `xml:"refund_account"`
	RefundRequestSource string `xml:"refund_request_source"`
}

type PayConfirmResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
}

var fail_response []byte
var success_response []byte

func init() {
	success := PayConfirmResponse{ReturnCode: "SUCCESS"}
	fail := PayConfirmResponse{ReturnCode: "FAIL"}
	success_response, _ = xml.Marshal(success)
	fail_response, _ = xml.Marshal(fail)
}

func PayConfirm(data []byte) (*ConfirmData, interface{}, func(err error) (contentType string, response []byte), error) {
	log.Println("receive wechatpay confirm: ", string(data))
	var confirmData ConfirmData
	if err := xml.Unmarshal(data, &confirmData); err != nil {
		if err != nil {
			return nil, nil, func(error) (string, []byte) {
				return "application/xml", fail_response
			}, err
		}
	}

	return &confirmData, string(data), func(err error) (string, []byte) {
		if err != nil {
			return "application/xml", fail_response
		}
		return "application/xml", success_response
	}, nil
}
