package wechatpay_test

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"log"
	"strings"
	"testing"
)

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
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

	log.Println(string(aesKey), len(aesKey))

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return nil, err
	}

	decrypted := make([]byte, len(crypted))

	for bs, be := 0, block.BlockSize(); bs < len(crypted); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(decrypted[bs:be], crypted[bs:be])
	}

	return PKCS7UnPadding(decrypted), nil
}
func TestECB(t *testing.T) {
	decrypted, _ := ECBDecrypt("ZaKo1JMFcG+2qMdQRaZeK4pRvyJnvsg6xyzyKvGwMxaYXyJ2cqE+GRuVn+oIEiixd42Iw/0vNlVd7f1mheP5b3qOKULEN14Ole1yWUEZh/iS2heEI7RkRlKtkZRI4z+JBZ1O7q0yZlQ63Dj3oAGmTr8qAupQ6JWOSB0IQWkphh8R2FwlgGNi1K/A6b7W5esVhO1Ki3ukXXAfNqBq790RNEbYYZ0rvt6fMRz0nJ55NZWavEclHamYjRBD31HPMmALQzdRpqktoNxQpCkogDcKFJxVOxYDHFBA/V9CI8JLVVMJz1YJjl4JtPNCwZlKLDdNfCRyA3MP/7mD3H8k50iHP3KHTe8W+0fjH9sJhDyXMc/BJohosKxMKJqiwRIxJWE0OpyJI7So12cPgoZ1cNosAtF7PgyQPA9wStjn9iV9kLtcPr9+j4YRSJSCxQcDH5FLWWn1fjaA4DAaiutsbEThnz8ffHEDt8x+d8eBDFcjoh014Sc0/dgBdOBHFZ9OA3pQQu8ykO4Qg+huxm3EFfpAuyvYHy9AlVvZG5UqOB+Al7kPv+mpDfa4uJsXSGILfVr6pF1dEtdMfSGjO7As/ZUgFJ4iNymoPvYBguU4O2k7O0gewfB9IOwMkb53GLlK7NO5LIs5zJ8ypm3EXvaK84yTxG0N72I/ZjImOzou1okSLqc2qt3EcIpJcPr8vL+R7mvkCESns6L4vRZU3MOH4AUImwdql4SgKBYx9DDt6hf4EzpHxNHEIfBhX6CsL8Wcnz7ukZBN1SyzoLFXIquPXTTaPrOPjH6LFzpCRy1C/I17uqxSt6VqXxUqz257e+hxonkRdZePYIsikcVYqb5uVlf0XuTJzzoQp93YQ4MG9NLxYOWKXmG3BtVoCh82yQeknIsJeC27vLIKl57D0SJRKH8BZJnQtk6UUX3Whg9BVSAlstm8K8L/ETZzo8wOsJaL0QNR15ZyLvUE+M8JtmdClvSP22U44rvnRb3ObkoItGR7EVtzOZI+96DsKa4ZT62Ihar9CQZQtWmE2exb0yqLGgtkT5B2X9CgrmcMnYpugFzyCxESGQBgCT3H9qs94IxTYkjM", "ObsessiontechUNcyijZJwaBk9Zg0jfG")

	log.Println(string(decrypted))
}
