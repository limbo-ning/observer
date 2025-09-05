package encrypt

import (
	"crypto/hmac"
	"crypto/sha256"
)

func Sha256Hmac(salt, data []byte) []byte {
	crypt := hmac.New(sha256.New, salt)
	crypt.Write(data)
	return crypt.Sum(nil)
}
