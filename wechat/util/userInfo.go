package util

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"obsessiontech/common/encrypt"
)

type UserInfo struct {
	Openid     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	Headimgurl string `json:"headimgurl"`
	Unionid    string `json:"unionid"`
	ErrCode    int    `json:"errcode,omitempty"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

func GetUserInfo(openID, accessToken string) (*UserInfo, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s", accessToken, openID)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get wechat user info: ", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get wechat user info: ", err)
		return nil, err
	}

	var userInfo UserInfo
	json.Unmarshal(body, &userInfo)

	log.Println("user info:", string(body), userInfo)

	return &userInfo, nil
}

type MiniAppUserInfo struct {
	OpenID    string `json:"openId"`
	UnionID   string `json:"unionId"`
	NickName  string `json:"nickName"`
	Gender    int    `json:"gender"`
	Language  string `json:"language"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
	AvatarURL string `json:"avatarUrl"`
}

type MobileInfo struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
}

func DecryptMiniappData(encrypted, sessionKey, iv string) ([]byte, error) {

	info, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		log.Println("error decrypt user info: encrypted info base64 decode failed")
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		log.Println("error decrypt user info: sessionKey base64 decode failed")
		return nil, err
	}
	initialVector, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		log.Println("error decrypt user info: iv base64 decode failed")
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println("error decrypt user info: initialize cipher by session key failed")
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, initialVector[:blockSize])

	origData := make([]byte, len(info))
	blockMode.CryptBlocks(origData, info)
	origData = encrypt.PKCS7UnPadding(origData)

	log.Println("decrypted: ", string(origData))

	return origData, nil
}
