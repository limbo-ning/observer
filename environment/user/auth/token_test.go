package auth_test

import (
	"encoding/hex"
	"fmt"
	"log"
	"obsessiontech/common/encrypt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func composeTokenOld(siteID, clientIP string, userID int) string {
	content := fmt.Sprintf("%s#%d#%s#%d", siteID, userID, clientIP, time.Now().Unix())
	return encrypt.Base64Encrypt(content)
}

func composeToken(siteID, clientIP string, userID int) string {

	content := fmt.Sprintf("%s#%d#%s#%d", siteID, userID, clientIP, time.Now().Unix())
	checksum := hex.EncodeToString(encrypt.Sha256Hmac([]byte("ob2022"), []byte(content)))
	// checksum := hex.EncodeToString(encrypt.Sha256Hmac([]byte(""), []byte(content)))

	return encrypt.Base64Encrypt(content + "#" + checksum)
}

func decomposeToken(token string) (siteID string, userID int, clientIP string, loginTime *time.Time, e error) {

	tokenData, err := encrypt.Base64Decrypt(token)

	if err != nil {
		e = err
		return
	}

	log.Println(string(tokenData))

	parts := strings.Split(string(tokenData), "#")
	if len(parts) != 5 {
		e = fmt.Errorf("凭证错误:格式错误")
		return
	}

	checksum := parts[4]
	if hex.EncodeToString(encrypt.Sha256Hmac([]byte("szvoc"), []byte(strings.Join(parts[0:4], "#")))) != checksum {
		e = fmt.Errorf("凭证错误:校验失败")
		return
	}

	siteID = parts[0]
	userID, err = strconv.Atoi(parts[1])
	if err != nil {
		e = fmt.Errorf("凭证错误:%s", err.Error())
		return
	}

	clientIP = parts[2]
	ts, err := strconv.ParseInt(parts[3], 0, 64)
	if err != nil {
		e = fmt.Errorf("凭证错误:%s", err.Error())
		return
	}
	loginTime = new(time.Time)
	*loginTime = time.Unix(ts, 0)

	return
}

func TestComposeToken(t *testing.T) {

	// result := composeTokenOld("keqin", "0bsessi0n", 385)
	// log.Println(result)

	result := composeToken("keqin", "pr0je(t(", 789)

	// result := composeToken("workout", "obsessiontechdev", 151)
	log.Println(result)
}

func TestDecomposeToken(t *testing.T) {
	// decomposeToken("ZW52aXJvbm1lbnQjMSM1OS40MS4xNjEuMTg1IzE2Mzk1Njg3NjQj184fy/oSVOjVCKNqHCxhXT0PI++Nd6yXmjFfPRyP8wA")
	_, _, _, _, err := decomposeToken("c3p2b2MjMTYjc3p2b2MjMTY0NTE0OTI1NyM2NTU2MWVhNzdmNTI0N2JkMWMwZjY5YjQ2ODg0ZGE5MGQ1ZGJjYjFlYWE2MjYyMDNlNGI1NDQ4NzkyZDVkZGY5")
	if err != nil {
		t.Error(err)
	}
}
