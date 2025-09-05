package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"log"
)

func AesCBCEncode(key, origData []byte) (crypted []byte, err error) {

	func() {
		if e := recover(); e != nil {
			log.Println("error encode aes: ", e)
			if e1, ok := e.(error); ok {
				err = e1
			} else {
				err = errors.New("加密失败")
			}
		}
	}()

	// 分组秘钥
	// NewCipher该函数限制了输入k的长度必须为16, 24或者32
	block, err := aes.NewCipher(key)
	if err != nil {
		println("error get cipher: ", err)
		return nil, err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	// 创建数组
	crypted = make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AesCBCDecode(key, encryptedData []byte) (decrypted []byte, err error) {

	func() {
		if e := recover(); e != nil {
			log.Println("error decode aes: ", e)
			if e1, ok := e.(error); ok {
				err = e1
			} else {
				err = errors.New("解密失败")
			}
		}
	}()

	// 分组秘钥
	// NewCipher该函数限制了输入k的长度必须为16, 24或者32
	block, err := aes.NewCipher(key)
	if err != nil {
		println("error get cipher: ", err)
		return nil, err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()

	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	// 创建数组
	decrypted = make([]byte, len(encryptedData))
	// 加密
	blockMode.CryptBlocks(decrypted, encryptedData)
	return PKCS7UnPadding(decrypted), nil
}
