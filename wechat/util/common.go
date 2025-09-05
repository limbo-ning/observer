package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"obsessiontech/common/encrypt"
	"obsessiontech/common/random"
)

func PlatformDecrypt(msg string) ([]byte, error) {
	crypted, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		log.Println("error decode using base64 std encoding: ", err)
		return nil, err
	}

	block, err := aes.NewCipher(platformAesKey)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, platformAesKey[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = encrypt.PKCS7UnPadding(origData)

	if len(origData) < 20 {
		log.Println("error decrypt data. decrypted data too short: ", msg)
		return nil, errors.New("error decrypt data")
	}

	// return origData[20 : len(origData)-len(platformAesKey)], nil
	return origData[20:], nil
}

func PlatformEncrypt(msg []byte) (encrypted, signature string, timestamp int, nonce string, err error) {

	dataLen := len(msg)

	nonce = random.GenerateNonce(16)
	timestamp = int(time.Now().Unix())

	bytesBuffer := bytes.NewBuffer([]byte{})
	if err = binary.Write(bytesBuffer, binary.BigEndian, []byte(nonce)); err != nil {
		return
	}
	if err = binary.Write(bytesBuffer, binary.BigEndian, int32(dataLen)); err != nil {
		return
	}
	if err = binary.Write(bytesBuffer, binary.BigEndian, msg); err != nil {
		return
	}
	if err = binary.Write(bytesBuffer, binary.BigEndian, []byte(Config.WechatPlatformAppID)); err != nil {
		return
	}

	var block cipher.Block
	block, err = aes.NewCipher(platformAesKey)
	if err != nil {
		return
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCEncrypter(block, platformAesKey[:blockSize])

	toEncrypt := encrypt.PKCS7Padding(bytesBuffer.Bytes(), 32)
	encryptedBytes := make([]byte, len(toEncrypt))
	blockMode.CryptBlocks(encryptedBytes, toEncrypt)

	encrypted = base64.StdEncoding.EncodeToString(encryptedBytes)

	signature = PlatformSign(timestamp, nonce, encrypted)

	return
}

func PlatformSign(timestamp int, nonce, msg string) string {
	toSign := []string{Config.WechatPlatformEncryptToken, fmt.Sprintf("%d", timestamp), nonce, msg}

	sort.Strings(toSign)

	sha := sha1.New()
	sha.Write([]byte(strings.Join(toSign, "")))
	signatureBytes := sha.Sum(nil)

	return hex.EncodeToString(signatureBytes)
}
