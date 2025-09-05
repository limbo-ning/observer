package wechatpay

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"obsessiontech/common/encrypt"
	"obsessiontech/common/random"
)

var E_user_input = errors.New("等待用户操作")
var E_result_unknown = errors.New("支付结果未知")
var E_retry = errors.New("重试")
var E_order_not_exists = errors.New("订单未成功提交")

type baseOption struct {
	SubAppID string
	SubMchID string
}

type baseParam struct {
	XMLName  xml.Name `xml:"xml"`
	AppID    string   `xml:"appid"`
	SubAppID string   `xml:"sub_appid,omitempty"`
	MchID    string   `xml:"mch_id"`
	SubMchID string   `xml:"sub_mch_id,omitempty"`
	NonceStr string   `xml:"nonce_str"`
	Sign     string   `xml:"sign"`
}

type baseRet struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	AppID      string `xml:"appid"`
	SubAppID   string `xml:"sub_appid"`
	MchID      string `xml:"mch_id"`
	SubMchID   string `xml:"sub_mch_id"`
	NonceStr   string `xml:"nonce_str"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	ErrCode    string `xml:"err_code"`
	ErrCodeDes string `xml:"err_code_des"`
}

func commonParam(appID, subAppID, mchID, subMchID string) map[string]interface{} {
	param := make(map[string]interface{})
	param["appid"] = appID
	param["sub_appid"] = subAppID
	param["mch_id"] = mchID
	param["sub_mch_id"] = subMchID
	param["nonce_str"] = random.GenerateNonce(16)
	return param
}

func Sign(params map[string]interface{}, key string) string {
	pairs := make([]string, 0)

	for k, v := range params {
		if vs, ok := v.(string); ok {
			if vs == "" {
				continue
			}
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(pairs)

	pairs = append(pairs, fmt.Sprintf("key=%s", key))

	var toSign = strings.Join(pairs, "&")

	log.Println("wechat content to sign", toSign)

	signed := fmt.Sprintf("%x", md5.Sum([]byte(toSign)))
	signed = strings.ToUpper(signed)

	log.Println("wechat signed", signed)

	return signed
}

func ECBDecrypt(msg, key string) ([]byte, error) {

	crypted, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		log.Println("error decode using base64 std encoding: ", err)
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(key))
	aesKey := strings.ToLower(hex.EncodeToString(hash.Sum(nil)))

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return nil, err
	}

	decrypted := make([]byte, len(crypted))

	for bs, be := 0, block.BlockSize(); bs < len(crypted); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(decrypted[bs:be], crypted[bs:be])
	}

	return encrypt.PKCS7UnPadding(decrypted), nil
}

func ECBEncrypt(origData []byte, key []byte) ([]byte, error) {
	hash := md5.New()
	hash.Write([]byte(key))
	aesKey := strings.ToLower(hex.EncodeToString(hash.Sum(nil)))

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return nil, err
	}

	length := (len(origData) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, origData)
	pad := byte(len(plain) - len(origData))
	for i := len(origData); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted := make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, block.BlockSize(); bs <= len(origData); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted, nil
}
