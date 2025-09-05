package encrypt

import (
	"encoding/base64"
	"strings"
)

func Base64Encrypt(content string) string {
	return strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(content)), "=")
}

func Base64Decrypt(encrypted string) ([]byte, error) {
	remainder := len(encrypted) % 4
	if remainder > 0 {
		for i := remainder; i < 4; i++ {
			encrypted += "="
		}
	}

	return base64.StdEncoding.DecodeString(encrypted)
}
