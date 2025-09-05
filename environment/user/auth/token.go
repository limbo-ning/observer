package auth

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/encrypt"
)

func composeToken(siteID, clientIP string, userID int) string {

	content := fmt.Sprintf("%s#%d#%s#%d", siteID, userID, clientIP, time.Now().Unix())
	checksum := hex.EncodeToString(encrypt.Sha256Hmac([]byte(Config.AuthTokenSalt), []byte(content)))

	return encrypt.Base64Encrypt(content + "#" + checksum)
}

func decomposeToken(token string) (siteID string, userID int, clientIP string, loginTime *time.Time, e error) {

	tokenData, err := encrypt.Base64Decrypt(token)

	if err != nil {
		e = err
		return
	}
	parts := strings.Split(string(tokenData), "#")
	if len(parts) != 5 {
		e = fmt.Errorf("凭证错误:格式错误")
		return
	}

	checksum := parts[4]

	if hex.EncodeToString(encrypt.Sha256Hmac([]byte(Config.AuthTokenSalt), []byte(strings.Join(parts[0:4], "#")))) != checksum {
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
