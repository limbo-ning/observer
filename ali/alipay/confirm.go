package alipay

import (
	"log"
	"net/url"
	"reflect"
)

const (
	TRADE_STATUS_CLOSED         = "TRADE_CLOSED"
	TRADE_STATUS_FINISHED       = "TRADE_FINISHED"
	TRADE_STATUS_SUCCESS        = "TRADE_SUCCESS"
	TRADE_STATUS_WAIT_BUYER_PAY = "WAIT_BUYER_PAY"
)

/*
只有交易通知状态为 TRADE_SUCCESS 或 TRADE_FINISHED 时，支付宝才会认定为买家付款成功。

注意：

状态 TRADE_SUCCESS 的通知触发条件是商户签约的产品支持退款功能的前提下，买家付款成功；

交易状态 TRADE_FINISHED 的通知触发条件是商户签约的产品不支持退款功能的前提下，买家付款成功；或者，商户签约的产品支持退款功能的前提下，交易已经成功并且已经超过可退款期限。
*/

type ConfirmData struct {
	NotifyTime  string `form:"notify_time" json:"notify_time"`
	NotifyType  string `form:"notify_type" json:"notify_type"`
	NotifyID    string `form:"notify_id" json:"notify_id"`
	AppID       string `form:"app_id" json:"app_id"`
	Charset     string `form:"charset" json:"-"`
	Version     string `form:"version" json:"-"`
	SignType    string `form:"sign_type" json:"-"`
	Sign        string `form:"sign" json:"-"`
	TradeNo     string `form:"trade_no" json:"trade_no"`
	OutTradeNo  string `form:"out_trade_no" json:"out_trade_no"`
	TradeStatus string `form:"trade_status" json:"trade_status"`
	OutBizNo    string `form:"out_biz_no" json:"out_biz_no,omitempty"`
}

func (c *ConfirmData) IsFinal() bool {
	return c.TradeStatus == TRADE_STATUS_SUCCESS || c.TradeStatus == TRADE_STATUS_FINISHED || c.TradeStatus == TRADE_STATUS_CLOSED
}

func PayConfirm(data []byte) (*ConfirmData, map[string]interface{}, func(err error) (contentType string, response []byte), error) {

	log.Println("receive alipay confirm: ", string(data))

	var confirmData ConfirmData

	rawData := make(map[string]interface{})

	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, nil, func(error) (string, []byte) {
			return "", []byte("fail")
		}, err
	}

	for k := range values {
		rawData[k] = values.Get(k)
	}

	v := reflect.ValueOf(&confirmData).Elem()
	for i := 0; i < v.NumField(); i++ {

		fieldInfo := v.Type().Field(i)
		tag := fieldInfo.Tag

		formName := tag.Get("form")

		v.Field(i).SetString(values.Get(formName))
	}

	return &confirmData, rawData, func(err error) (string, []byte) {
		if err != nil {
			return "", []byte("fail")
		}
		return "", []byte("success")
	}, nil
}
