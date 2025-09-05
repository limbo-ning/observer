package encrypt_test

import (
	"crypto"
	"encoding/hex"
	"log"
	"strings"
	"testing"

	"obsessiontech/common/encrypt"
)

func TestSha(t *testing.T) {
	result := encrypt.Sha256Hmac([]byte("0b5e55-0n"), []byte("Limbo123456"))
	log.Println(hex.EncodeToString(result))
}

func TestAes(t *testing.T) {
	key := encrypt.Md5sum([]byte(strings.ToLower("jxff")))

	keyBytes := make([]byte, 16)
	for i, b := range key {
		keyBytes[i] = b
	}

	log.Println("md5: ", hex.EncodeToString(keyBytes))
	result, err := encrypt.AesCBCEncode(keyBytes, []byte("Limbo123456"))
	if err != nil {
		t.Error(err)
	}
	log.Println(hex.EncodeToString(result))

	inputBytes, err := hex.DecodeString("b767daa2adc99fdc9e2b37daacd2d999")
	if err != nil {
		t.Error(err)
	}

	result, err = encrypt.AesCBCDecode(keyBytes, inputBytes)
	if err != nil {
		t.Error(err)
	}
	log.Println(string(result))
}

func TestJWS(t *testing.T) {

	key1 := "\n-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgK8iV3qGflfuLMiT5\ntHIFsq69SQVodRHVVPO6FQ7HnwKgCgYIKoZIzj0DAQehRANCAASvgDZ0Xib02PxM\nMYcBc+FS6xmFLx5JIdC3bXvfwpC9GvCf+VqsD3v1yvSeeJX1Rb4BGEpELEhRHeUC\nRQhLn4BQ\n-----END PRIVATE KEY-----"
	key2 := `
-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgK8iV3qGflfuLMiT5
tHIFsq69SQVodRHVVPO6FQ7HnwKgCgYIKoZIzj0DAQehRANCAASvgDZ0Xib02PxM
MYcBc+FS6xmFLx5JIdC3bXvfwpC9GvCf+VqsD3v1yvSeeJX1Rb4BGEpELEhRHeUC
RQhLn4BQ
-----END PRIVATE KEY-----`

	log.Println(key1, len(key1))
	log.Println(key2, len(key2))
	log.Println(key1 == key2)

	// result, err := encrypt.JWSECDSASign(crypto.SHA256, []byte(key1), []byte("12345"))

	result, err := encrypt.JWSECDSASign(crypto.SHA256, []byte(key1), "12345")

	if err != nil {
		t.Error(err)
	}

	log.Println(result)
}
