package engine

import (
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"obsessiontech/common/encrypt"
	"obsessiontech/common/img/chaostoken"
	"obsessiontech/common/random"
	"obsessiontech/environment/user"
)

const pw_salt = "0b5e55-0n"

const PASSWORD_AUTH_MODULE = "user_auth_password"
const PASSWORD_AUTH = "password"

var e_password_fail = errors.New("用户名或密码有误")
var e_code_incorrect = errors.New("验证码不正确")
var e_need_password = errors.New("请输入密码")
var e_need_username = errors.New("请输入用户名")

func init() {
	register(PASSWORD_AUTH, PASSWORD_AUTH_MODULE, func() IAuth {
		return &PasswordAuth{}
	})
}

type PasswordAuth struct {
	UsernameColumn     string   `json:"usernameColumn"`
	IsAesRequired      bool     `json:"isAesRequired"`
	IsCodeRequired     bool     `json:"isCodeRequired"`
	RegisterAvailable  bool     `json:"registerAvailable"`
	PasswordPattern    []string `json:"passwordPattern"`
	PasswordPatternTip string   `json:"passwordPatternTip"`
}

type PasswordAuthParam struct {
	user.UserInfo
	Password       string `json:"password"`
	UpdatePassword string `json:"updatePassword"`
	Code           string `json:"code"`
}

func (a *PasswordAuth) Tip() map[string]any {
	return map[string]any{
		"isCodeRequired":  a.IsCodeRequired || a.IsAesRequired,
		"passwordPattern": a.PasswordPattern,
		"passwordTip":     a.PasswordPatternTip,
	}
}

func (p *PasswordAuth) descryptPasswordParam(crypted, sault string) (string, error) {

	var decrypted string

	if !p.IsAesRequired {
		decrypt, err := encrypt.Base64Decrypt(crypted)
		if err != nil {
			return "", err
		}
		decrypted = string(decrypt)
	} else {
		if sault == "" {
			return "", e_code_incorrect
		}
		key := encrypt.Md5sum([]byte(strings.ToLower(sault)))
		keyBytes := make([]byte, 16)
		for i, k := range key {
			keyBytes[i] = k
		}

		var cryptBytes []byte
		cryptBytes, err := hex.DecodeString(crypted)
		if err != nil {
			return "", err
		}

		decrypt, err := encrypt.AesCBCDecode(keyBytes, cryptBytes)
		if err != nil {
			return "", err
		}
		decrypted = string(decrypt)
	}

	return decrypted, nil
}

func EncryptPassword(password string) (string, error) {
	encrypted := encrypt.Sha256Hmac([]byte(pw_salt), []byte(password))

	return hex.EncodeToString(encrypted), nil
}

func (p *PasswordAuth) requestCode(siteID string, txn *sql.Tx, u *user.User) (string, error) {
	code := random.GenerateNonce(4)

	u.Ext["password_code"] = code

	if err := u.Update(siteID, txn); err != nil {
		return "", err
	}

	return code, nil
}

func (p *PasswordAuth) verifyCode(siteID string, txn *sql.Tx, u *user.User, code string) error {
	set := u.Ext["password_code"]

	if set == nil {
		log.Println("error no code set")
		return e_code_incorrect
	}

	delete(u.Ext, "password_code")

	if !strings.EqualFold(set.(string), code) {
		log.Printf("code dismatch: [%s] [%s]", set.(string), code)
		return e_code_incorrect
	}

	return nil
}

func (p *PasswordAuth) ValidatePassword(password string) error {

	if len(p.PasswordPattern) == 0 {
		return nil
	}

	for _, pattern := range p.PasswordPattern {
		matched, err := regexp.Match(pattern, []byte(password))
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("密码强度不符合:%s", p.PasswordPatternTip)
		}
	}

	return nil
}

func (a *PasswordAuth) Validate() error {

	switch a.UsernameColumn {
	case "username":
	case "mobile":
	case "email":
	default:
		a.UsernameColumn = "username"
	}
	return nil
}

func (a *PasswordAuth) CheckExists(siteID, requestID string, txn *sql.Tx, toCheck *user.User) error {

	var toLock string
	switch a.UsernameColumn {
	case "username":
		toLock = toCheck.Username
	case "mobile":
		toLock = toCheck.Mobile
	case "email":
		toLock = toCheck.Email
	}

	if toLock == "" {
		return nil
	}

	if err := user.RequestRegisterLock(siteID, requestID, []string{toLock}, func() error {
		if u, err := user.GetUserWithTxn(siteID, txn, a.UsernameColumn, toLock, true); err != nil {
			if err != user.E_user_not_exists {
				return err
			}
		} else if u.UserID != toCheck.UserID {
			return e_user_exists
		}

		return nil
	}); err != nil {
		log.Println("error request lock action: ", requestID, err)
		return err
	}

	return nil
}

func (p *PasswordAuth) Register(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {

	if !p.RegisterAvailable {
		return nil, nil, e_register_unavailable
	}

	var param PasswordAuthParam

	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}

	if param.Username == "" {
		return nil, nil, e_need_username
	}

	if param.Password == "" {
		return nil, nil, e_need_password
	}

	decrypted, err := p.descryptPasswordParam(param.Password, param.Username)
	if err != nil {
		return nil, nil, err
	}

	if err := p.ValidatePassword(decrypted); err != nil {
		return nil, nil, err
	}

	pw, err := EncryptPassword(decrypted)
	if err != nil {
		return nil, nil, e_need_password
	}

	newUser := &user.User{Password: pw, UserInfo: param.UserInfo}
	switch p.UsernameColumn {
	case "username":
		newUser.Username = param.Username
	case "mobile":
		newUser.Mobile = param.Username
	case "email":
		newUser.Email = param.Username
	}

	if err := user.RequestRegisterLock(siteID, requestID, []string{param.Username}, func() error {
		if _, err := user.GetUserWithTxn(siteID, txn, p.UsernameColumn, param.Username, true); err != nil {
			if err != user.E_user_not_exists {
				return err
			}
			if err := newUser.Add(siteID, txn); err != nil {
				return err
			}
		} else {
			return e_user_exists
		}

		return nil
	}); err != nil {
		log.Println("error request lock action: ", requestID, err)
		return nil, nil, err
	}

	return newUser, nil, nil
}

func (p *PasswordAuth) Login(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	var param PasswordAuthParam

	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}

	if param.Username == "" {
		return nil, nil, e_need_username
	}

	existUser, err := user.GetUserWithTxn(siteID, txn, p.UsernameColumn, param.Username, true)
	if err != nil {
		return nil, nil, err
	}

	if p.IsCodeRequired || p.IsAesRequired {
		if param.Code == "" {
			code, err := p.requestCode(siteID, txn, existUser)
			if err != nil {
				return nil, nil, err
			}

			codeImgBytes, err := chaostoken.ChaosToken(code, "", 300, 100)
			if err != nil {
				return nil, nil, err
			}

			return nil, map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码", "code": fmt.Sprintf("data:image/jpg;base64,%s", base64.StdEncoding.EncodeToString(codeImgBytes))}, nil
		} else {
			defer func() {
				go existUser.Update(siteID, nil)
			}()
			if err := p.verifyCode(siteID, txn, existUser, param.Code); err != nil {
				return nil, nil, err
			}
		}
	}
	if param.Password == "" {
		return nil, nil, e_need_password
	}

	decrypted, err := p.descryptPasswordParam(param.Password, param.Code)
	if err != nil {
		return nil, nil, err
	}

	pw, err := EncryptPassword(decrypted)
	if err != nil {
		log.Println("error encrypt: ", err)
		return nil, nil, e_need_password
	}
	log.Println("check password: ", decrypted, pw)

	if existUser.Password != pw {
		log.Println("check password failed: ", existUser.Password)
		return nil, nil, e_password_fail
	}

	result := make(map[string]interface{})

	if err := p.ValidatePassword(decrypted); err != nil {
		result["tip"] = err.Error()
	}

	return existUser, result, nil
}

func (p *PasswordAuth) Bind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	var param PasswordAuthParam

	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, err
	}

	if p.IsCodeRequired || p.IsAesRequired {
		if param.Code == "" {
			code, err := p.requestCode(siteID, txn, existUser)
			if err != nil {
				return nil, err
			}
			codeImgBytes, err := chaostoken.ChaosToken(code, "", 300, 100)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码", "code": fmt.Sprintf("data:image/jpg;base64,%s", base64.StdEncoding.EncodeToString(codeImgBytes))}, nil
		} else {
			defer func() {
				go existUser.Update(siteID, nil)
			}()
			if err := p.verifyCode(siteID, txn, existUser, param.Code); err != nil {
				return nil, err
			}
		}
	}

	if param.Password == "" {
		return nil, e_need_password
	}

	decrypted, err := p.descryptPasswordParam(param.Password, param.Code)
	if err != nil {
		return nil, err
	}

	pw, err := EncryptPassword(decrypted)
	if err != nil {
		return nil, e_need_password
	}

	if param.Username != "" {
		usernameUser, err := user.GetUser(siteID, p.UsernameColumn, param.Username)
		if err != nil {
			if err != user.E_user_not_exists {
				return nil, err
			}
		} else {
			if usernameUser.UserID != existUser.UserID {
				return nil, e_user_exists
			}
		}
		switch p.UsernameColumn {
		case "username":
			existUser.Username = param.Username
		case "mobile":
			existUser.Mobile = param.Username
		case "email":
			existUser.Email = param.Username
		}
	}

	if existUser.Password != "" {

		if param.UpdatePassword != "" {
			if existUser.Password != pw {
				return nil, e_password_fail
			}
			decrypted, err = p.descryptPasswordParam(param.UpdatePassword, param.Code)
			if err != nil {
				return nil, err
			}

			log.Println("new password decryted: ", decrypted)
			pw, err = EncryptPassword(decrypted)
			if err != nil {
				return nil, e_need_password
			}
			log.Println("new password encrypted: ", pw)
		}
	}

	if err := p.ValidatePassword(decrypted); err != nil {
		return nil, err
	}
	log.Println("bind password: ", decrypted, pw)

	existUser.Password = pw
	if err := existUser.Update(siteID, txn); err != nil {
		log.Println("bind update user err: ", err)
		return nil, err
	}

	log.Println("bind pw complete: ", existUser)

	return nil, nil
}

func (p *PasswordAuth) UnBind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {

	existUser.Password = ""
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	return nil, nil
}
