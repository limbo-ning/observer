package engine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"obsessiontech/ali/notify"
	"obsessiontech/common/config"
	"obsessiontech/common/datasource"
	"obsessiontech/common/random"
	"obsessiontech/environment/site/initialization"
	"obsessiontech/environment/user"
)

const SMS_AUTH_MODULE = "user_auth_sms"
const SMS_AUTH = "sms"

var e_wrong_mobile = errors.New("手机号码无效")
var e_code_fail = errors.New("验证码错误或已过期")

const sms_status_new = "NEW"
const sms_status_expired = "EXPIRED"

var smsAuthConfig struct {
	SMSSign             string
	SMSAuthTemplateCode string
	SMSTestMobile       string
}

func init() {
	config.GetConfig("config.yaml", &smsAuthConfig)
	register(SMS_AUTH, SMS_AUTH_MODULE, func() IAuth {
		return &SMSAuth{}
	})

	initialization.Register(SMS_AUTH_MODULE, []string{"sms_code"})
}

type SMSAuth struct {
	SiteName          string `json:"siteName"`
	SMSSignature      string `json:"smsSignature"`
	RegisterAvailable bool   `json:"registerAvailable"`
}

type SMSAuthParam struct {
	user.UserInfo
	SMSCode    string `json:"smsCode"`
	NoRegister bool   `json:"noRegister"`
}

func (a *SMSAuth) Validate() error {
	return nil
}

func (a *SMSAuth) Tip() map[string]any {
	return make(map[string]any)
}

func (a *SMSAuth) CheckExists(siteID, requestID string, txn *sql.Tx, toCheck *user.User) error {

	if toCheck.Mobile == "" {
		return nil
	}

	if err := user.RequestRegisterLock(siteID, requestID, []string{toCheck.Mobile}, func() error {
		if u, err := user.GetUserWithTxn(siteID, txn, "mobile", toCheck.Mobile, true); err != nil {
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

func (a *SMSAuth) Register(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	if !a.RegisterAvailable {
		return nil, nil, e_register_unavailable
	}

	var param SMSAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}
	if param.Mobile == "" {
		return nil, nil, e_wrong_mobile
	}
	if param.SMSCode == "" {
		if err := sendSMSCode(siteID, a.SiteName, a.SMSSignature, param.Mobile); err != nil {
			return nil, nil, err
		}
		return nil, map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码"}, nil
	}
	if err := verifySMSCode(siteID, param.Mobile, param.SMSCode); err != nil {
		return nil, nil, err
	}
	newUser := &user.User{UserInfo: param.UserInfo}

	if err := user.RequestRegisterLock(siteID, requestID, []string{param.Mobile}, func() error {
		if _, err := user.GetUserWithTxn(siteID, txn, "mobile", param.Mobile, true); err != nil {
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

func (a *SMSAuth) Login(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	var param SMSAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}
	if param.Mobile == "" {
		return nil, nil, e_wrong_mobile
	}

	existUser, err := user.GetUserWithTxn(siteID, txn, "mobile", param.Mobile, true)
	if err != nil {
		if err != user.E_user_not_exists {
			return nil, nil, err
		}
		if !param.NoRegister {
			return nil, nil, E_toggle_method
		}
		return nil, nil, user.E_user_not_exists
	}
	if param.SMSCode == "" {
		if err := sendSMSCode(siteID, a.SiteName, a.SMSSignature, param.Mobile); err != nil {
			return nil, nil, err
		}
		return nil, map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码"}, nil
	}
	if err := verifySMSCode(siteID, param.Mobile, param.SMSCode); err != nil {
		return nil, nil, err
	}

	return existUser, nil, nil
}

func (a *SMSAuth) Bind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	var param SMSAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, err
	}
	if param.Mobile == "" {
		return nil, e_wrong_mobile
	}

	if param.SMSCode == "" {
		if err := sendSMSCode(siteID, a.SiteName, a.SMSSignature, param.Mobile); err != nil {
			return nil, err
		}
		return map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码"}, nil
	}
	if err := verifySMSCode(siteID, param.Mobile, param.SMSCode); err != nil {
		return nil, err
	}

	existMobileUser, err := user.GetUserWithTxn(siteID, txn, "mobile", param.Mobile, true)
	if err != nil {
		if err != user.E_user_not_exists {
			return nil, err
		}
	} else if existMobileUser != nil && existMobileUser.UserID != existUser.UserID {
		return nil, e_user_exists
	}

	existUser.Mobile = param.Mobile
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *SMSAuth) UnBind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	var param SMSAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, err
	}
	if param.Mobile == "" || param.Mobile != existUser.Mobile {
		return nil, e_wrong_mobile
	}

	if param.SMSCode == "" {
		if err := sendSMSCode(siteID, a.SiteName, a.SMSSignature, param.Mobile); err != nil {
			return nil, err
		}
		return map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码"}, nil
	}
	if err := verifySMSCode(siteID, param.Mobile, param.SMSCode); err != nil {
		return nil, err
	}

	existUser.Mobile = ""
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	return nil, nil
}

func sendSMSCode(siteID, siteName, sign, mobile string) error {

	if mobile == smsAuthConfig.SMSTestMobile {
		return nil
	}

	code := random.GenerateRandomNumber(6)

	if sign == "" {
		sign = smsAuthConfig.SMSSign
	}

	param := make(map[string]string)
	param["name"] = siteName
	param["code"] = code

	return datasource.Txn(func(txn *sql.Tx) {
		stmt, err := txn.Prepare(`
			INSERT INTO ` + siteID + `_sms_code
				(mobile, code, status)
			VALUES
				(?,?,?)
		`)
		if err != nil {
			log.Println("error insert sms code: ", err)
			panic(err)
		}
		defer stmt.Close()

		if _, err := stmt.Exec(mobile, code, sms_status_new); err != nil {
			log.Println("error insert sms code: ", err)
			panic(err)
		}

		if err := notify.SendSMS(mobile, sign, smsAuthConfig.SMSAuthTemplateCode, param); err != nil {
			panic(err)
		}
	})
}

func verifySMSCode(siteID, mobile, code string) error {

	if mobile == smsAuthConfig.SMSTestMobile {
		return nil
	}

	if ret, err := datasource.GetConn().Exec(`
		UPDATE
			`+siteID+`_sms_code
		SET
			status = ?
		WHERE
			mobile = ? AND code = ? AND status = ? AND create_time > DATE_ADD(CURRENT_TIMESTAMP, INTERVAL -10 Minute)
	`, sms_status_expired, mobile, code, sms_status_new); err != nil {
		log.Println("error update sms code: ", err)
		return err
	} else if rows, _ := ret.RowsAffected(); rows == 0 {
		return e_code_fail
	}
	return nil
}
