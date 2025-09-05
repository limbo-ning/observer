package message

import (
	"encoding/xml"
	"log"

	"obsessiontech/wechat/util"
)

type EncryptedMessage struct {
	ToUserName string `xml:"ToUserName"`
	Encrypt    string `xml:"Encrypt"`
}

func PlatformReceive(timestamp int, msgSignature, nonce, encryptType string, data []byte) ([]byte, error) {

	if encryptType != "aes" {
		log.Println("error unsupported platform message push encrypt type: ", encryptType)
	}

	var encryted EncryptedMessage

	if err := xml.Unmarshal(data, &encryted); err != nil {
		log.Println("error unmarshal platform message push: ", err)
		return nil, err
	}

	decrypted, err := util.PlatformDecrypt(encryted.Encrypt)
	if err != nil {
		return nil, err
	}

	log.Println("decrypted: ", string(decrypted))
	return decrypted, nil
}

type EncryptedReply struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      string   `xml:"Encrypt"`
	MsgSignature string   `xml:"MsgSignature"`
	TimeStamp    int      `xml:"TimeStamp"`
	Nonce        string   `xml:"Nonce"`
}

func PlatformReplyOpen(msg Message) ([]byte, error) {
	data, err := xml.Marshal(msg)
	if err != nil {
		return nil, err
	}
	encryted, signature, timestamp, nonce, err := util.PlatformEncrypt(data)

	var reply EncryptedReply
	reply.Encrypt = encryted
	reply.MsgSignature = signature
	reply.TimeStamp = timestamp
	reply.Nonce = nonce

	return xml.Marshal(reply)
}
