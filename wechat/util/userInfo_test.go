package util_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"log"
	"testing"

	"obsessiontech/common/encrypt"
)

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

func TestDecrypt(t *testing.T) {
	DecryptMiniappData("lYx6YT5zfHTY8vxdUTYhu+5g57VBZ4ujKe1f/rVwZNpaAJGK7uv9WjqcuyhhzbCybYxSGtqUr9pYKBG87JS7am01x2tV5fVoAWcFRewVNlbYQh6l/JzCw33AHHVR+yYHPHHiEDXnbvZe58HcBz36upvf8MQ0oy7WbcL4MrN0hYJdsQSTiEgYKJP1A3VmF0bBQWLbPgyONL4q9I+q71CqGQ==", "y+T88YAJqFosobfTDxI08A==", "pwiYb041JJyMySbMzpxPGQ==")
}
