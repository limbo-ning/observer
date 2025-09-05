package notify

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"log"
)

var e_wrong_mobile = errors.New("非法手机号")
var e_frequent_mobile = errors.New("当前号码发送过频，请稍后再试")
var e_system_error = errors.New("系统繁忙，请稍后重试")

func SendSMS(mobile, signature, templateCode string, templateParam map[string]string) error {
	param := make(map[string]string)

	param["Action"] = "SendSms"
	param["Version"] = "2017-05-25"
	param["RegionID"] = "cn-hangzhou"
	param["PhoneNumbers"] = mobile
	param["SignName"] = signature
	param["TemplateCode"] = templateCode
	templateParamBytes, _ := json.Marshal(templateParam)
	param["TemplateParam"] = string(templateParamBytes)

	data, err := sendToGateway("GET", param)
	if err != nil {
		log.Println("error send sms: ", err)
		return e_system_error
	}

	var resp smsResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		log.Printf("error unmarshal sms response: %v %s", err, string(data))
		return e_system_error
	}

	switch resp.Code {
	case "OK":
		return nil
	case "isv.MOBILE_NUMBER_ILLEGAL":
		return e_wrong_mobile
	case "isv.MOBILE_COUNT_OVER_LIMIT":
		return e_frequent_mobile
	default:
		log.Printf("error send sms: %+v", resp)
		return e_system_error
	}
}

type smsResponse struct {
	XMLName   xml.Name `xml:"SendSmsResponse"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestID"`
	BizID     string   `xml:"BizID"`
	Code      string   `xml:"Code"`
}
