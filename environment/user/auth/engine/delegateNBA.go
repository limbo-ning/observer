package engine

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"obsessiontech/common/encrypt"
	"obsessiontech/common/util"
	"obsessiontech/environment/user"
)

const DELEGATE_NBA_AUTH_MODULE = "user_auth_delegate_nba"
const DELEGATE_AUTH = "delegate_nba"

func init() {
	register(DELEGATE_AUTH, DELEGATE_NBA_AUTH_MODULE, func() IAuth {
		return &DeletegateNBAAuth{}
	})
}

type DeletegateNBAAuth struct {
	MerkleHost    string `json:"merkleHost"`
	UUID          string `json:"uuid"`
	Secret        string `json:"secret"`
	BlackFishHost string `json:"blackFishHost"`
	CommonCode    string `json:"commonCode"`
}

type DeletegateNBAAuthParam struct {
	Mobile         string `json:"mobile"`
	SMSCode        string `json:"smsCode"`
	Channel        string `json:"channel"`
	BlackFishToken string `json:"bfToken"`
	Vendor         string `json:"vendor"`
	VendorID       string `json:"vendorID"`
}

func (a *DeletegateNBAAuth) Validate() error {
	return nil
}

func (a *DeletegateNBAAuth) Tip() map[string]any {
	return make(map[string]any)
}

func (a *DeletegateNBAAuth) CheckExists(siteID, requestID string, txn *sql.Tx, toCheck *user.User) error {
	return nil
}

func (a *DeletegateNBAAuth) Register(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {

	log.Println("nba delegate: ", string(paramData), a.CommonCode)

	var param DeletegateNBAAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}

	if param.Vendor != "" && param.VendorID != "" {
		u, err := a.syncNBACustomerInfo(siteID, requestID, txn, "", param.Vendor, param.VendorID, false)
		if err != nil {
			return nil, nil, err
		}
		return u, nil, nil
	}

	if param.Mobile == "" {
		return nil, nil, e_wrong_mobile
	}
	if param.BlackFishToken != "" {
		u, err := a.checkBlackFishToken(siteID, requestID, txn, &param)
		if err != nil {
			return nil, nil, err
		}
		return u, nil, nil
	}

	if param.SMSCode == "" {
		return nil, map[string]interface{}{"retCode": 2, "retMsg": "请输入验证码"}, a.sendNBASmsCode(&param)
	}

	if a.CommonCode != "" && param.SMSCode == a.CommonCode {
		log.Println("testing login: ", param.Mobile)
		u, err := user.GetUserWithTxn(siteID, txn, "mobile", param.Mobile, true)
		if err != nil {
			if err == user.E_user_not_exists {
				u = new(user.User)
				u.Mobile = param.Mobile
				u.Profile = make(map[string][]string)
				u.Ext = make(map[string]interface{})
				if err := u.Add(siteID, txn); err != nil {
					log.Println("testing login add error: ", err)
					if err != user.E_user_exists {
						return nil, nil, err
					}
				}
				return u, map[string]interface{}{
					"isNew": true,
				}, nil
			}
			log.Println("testing login error: ", err)
			return nil, nil, err
		}
		log.Println("testing login done: ", u.UserID)
		return u, map[string]interface{}{
			"isNew": true,
		}, nil
	}

	u, isNew, err := a.enrollOrLogin(siteID, requestID, txn, &param)
	if err != nil {
		return nil, nil, err
	}
	return u, map[string]interface{}{
		"isNew": isNew,
	}, nil
}

func (a *DeletegateNBAAuth) Login(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	return nil, nil, E_toggle_method
}

func (a *DeletegateNBAAuth) checkBlackFishToken(siteID, requestID string, txn *sql.Tx, param *DeletegateNBAAuthParam) (*user.User, error) {
	URL := fmt.Sprintf("%s/qmqifs-web/fspCommon/user/checkToken", a.BlackFishHost)

	data := map[string]interface{}{
		"bizParams": map[string]interface{}{
			"appId":       "NBA",
			"appType":     8,
			"token":       param.BlackFishToken,
			"phoneNumber": param.Mobile,
		},
	}

	dataByte, _ := json.Marshal(data)

	log.Println("check black fish: ", URL, string(dataByte))

	resp, err := http.Post(URL, "application/json", bytes.NewReader(dataByte))
	if err != nil {
		log.Println("error check black fish token:", URL, err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error check black fish token:", err)
		return nil, err
	}

	log.Println("check black fish token ret: ", string(body))

	var ret struct {
		nbaBaseRet
		ErrCode int    `json:"errorCode"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		log.Println("error unmarshal check black fish token:", err)
		return nil, err
	}

	if !ret.Success {
		return nil, errors.New(ret.Msg)
	}

	return a.syncNBACustomerInfo(siteID, requestID, txn, param.Mobile, "", "", false)
}

type nbaBaseRet struct {
	Success bool `json:"success"`
}

type nbaErrRet struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

type nbaEnrollOrLoginRet struct {
	nbaErrRet
	ID                 int    `json:"id"`
	ExternalCustomerID string `json:"external_customer_id"`
	NewMember          bool   `json:"new_member"`
}

func (a *DeletegateNBAAuth) enrollOrLogin(siteID, request string, txn *sql.Tx, param *DeletegateNBAAuthParam) (*user.User, bool, error) {
	URL := fmt.Sprintf("/2018-01-01/api/enroll_or_show_customer.json?uuid=%s&external_customer_id=%s&confirmation_code=%s&channel=%s&status=active&enroll_status=true", a.UUID, param.Mobile, param.SMSCode, param.Channel)

	URL += "&sig=" + a.sign(URL)

	resp, err := http.Get(a.MerkleHost + URL)
	if err != nil {
		log.Println("error enroll or login nba:", URL, err)
		return nil, false, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error enroll or login nba:", err)
		return nil, false, err
	}

	log.Println("enroll nba sms ret: ", string(body))

	var ret struct {
		nbaBaseRet
		Data nbaEnrollOrLoginRet `json:"data"`
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		log.Println("error marshal enroll or login nba ret: ", string(body), err)
		return nil, false, err
	}

	if !ret.Success {
		log.Println("error enroll or login nba ret: ", string(body), err)
		return nil, false, errors.New(ret.Data.Msg)
	}

	u, err := a.syncNBACustomerInfo(siteID, request, txn, param.Mobile, "", "", ret.Data.NewMember)
	if err != nil {
		return nil, false, err
	}
	return u, ret.Data.NewMember, nil
}

type nbaCustomerInfo struct {
	nbaErrRet
	Name               string `json:"name"`
	Status             string `json:"status"`
	Nickname           string `json:"nickname"`
	ImgeURL            string `json:"image_url"`
	ExternalCustomerID string `json:"external_customer_id"`
}

func (a *DeletegateNBAAuth) syncNBACustomerInfo(siteID, requestID string, txn *sql.Tx, mobile, vendor, vendorID string, isNewUser bool) (*user.User, error) {

	// URL := fmt.Sprintf("/2016-12-01/data/customer/show.json?uuid=%s&include_vendors=nickname&include=detail,member_attributes", a.UUID)
	URL := fmt.Sprintf("/2016-12-01/data/customer/show.json?uuid=%s&include_vendors=nickname", a.UUID)

	if mobile != "" {
		URL += "&external_customer_id=" + mobile
	}

	if vendor != "" {
		URL += "&vendor=" + vendor
	}
	if vendorID != "" {
		URL += "&vendor_id=" + vendorID
	}

	URL += "&sig=" + a.sign(URL)

	log.Println("sync nba customer: ", URL)

	resp, err := http.Get(a.MerkleHost + URL)
	if err != nil {
		log.Println("error get nba customer info:", URL, err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get nba customer info:", err)
		return nil, err
	}

	log.Println("get nba customer info ret: ", string(body))

	var ret struct {
		nbaBaseRet
		Data nbaCustomerInfo `json:"data"`
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		log.Println("error marshal get nba customer info ret: ", string(body), err)
		return nil, err
	}

	if !ret.Success {
		log.Println("error get nba customer info ret: ", string(body), err)
		return nil, errors.New(ret.Data.Msg)
	}

	if ret.Data.Status == "pending" {
		return nil, errors.New("用户未绑定手机号")
	}

	if ret.Data.ExternalCustomerID == "" && mobile == "" {
		log.Println("error get nba customer info ret: ", string(body), err)
		return nil, errors.New("没有external_customer_id")
	}

	mobile = ret.Data.ExternalCustomerID

	var u *user.User

	if err := user.RequestRegisterLock(siteID, requestID, []string{mobile}, func() error {

		u, err = user.GetUserWithTxn(siteID, txn, "mobile", mobile, true)
		if err != nil {
			if err != user.E_user_not_exists {
				return err
			}

			u = new(user.User)
			u.Mobile = mobile
			u.Profile = make(map[string][]string)
			if ret.Data.Nickname != "" {
				u.Profile["nickname"] = []string{ret.Data.Nickname}
			} else {
				u.Profile["nickname"] = []string{util.Mask(mobile, "X")}
			}
			if ret.Data.ImgeURL != "" {
				u.Profile["thumbURL"] = []string{ret.Data.ImgeURL}
			}
			u.Ext = make(map[string]interface{})
			u.Ext["nbaCustomerInfo"] = ret.Data

			if isNewUser {
				u.Ext["isNewUser"] = true
			}

			if err := u.Add(siteID, txn); err != nil {
				log.Println("error add new user: ", err)
				if err != user.E_user_exists {
					return err
				}
			}
			return nil
		}

		if ret.Data.Nickname != "" {
			u.Profile["nickname"] = []string{ret.Data.Nickname}
		} else {
			u.Profile["nickname"] = []string{util.Mask(mobile, "X")}
		}
		if ret.Data.ImgeURL != "" {
			u.Profile["thumb"] = []string{ret.Data.ImgeURL}
		}
		u.Ext["nbaCustomerInfo"] = ret.Data

		u.Update(siteID, txn)

		return nil
	}); err != nil {
		log.Println("error request lock action: ", requestID, err)
		return nil, err
	}

	return u, nil
}

func (a *DeletegateNBAAuth) sendNBASmsCode(param *DeletegateNBAAuthParam) error {

	URL := fmt.Sprintf("/api/text_confirmation?uuid=%s&phone=%s", a.UUID, param.Mobile)

	URL += "&sig=" + a.sign(URL)

	resp, err := http.Get(a.MerkleHost + URL)
	if err != nil {
		log.Println("error send nba sms:", URL, err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error send nba sms:", err)
		return err
	}

	log.Println("send nba sms ret: ", string(body))

	var ret struct {
		nbaBaseRet
		Data nbaErrRet `json:"data"`
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		log.Println("error marshal nba sms ret: ", string(body), err)
		return err
	}

	if !ret.Success {
		log.Println("error send nba sms ret: ", string(body), err)
		return errors.New(ret.Data.Msg)
	}

	return nil
}

func (a *DeletegateNBAAuth) sign(URL string) string {
	raw := a.Secret + URL
	return fmt.Sprintf("%x", encrypt.Md5sum([]byte(raw)))
}

func (a *DeletegateNBAAuth) Bind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	return nil, nil
}

func (a *DeletegateNBAAuth) UnBind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {
	return nil, nil
}
