package engine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"obsessiontech/common/encrypt"
	"obsessiontech/environment/user"
	"obsessiontech/wechat/util"
)

const WECHAT_AUTH_MODULE = "user_auth_wechat"
const WECHAT_AUTH = "wechat"

func init() {
	register(WECHAT_AUTH, WECHAT_AUTH_MODULE, func() IAuth {
		return &WechatAuth{}
	})
}

type WechatAuth struct {
	AppConfig []struct {
		AppType   string `json:"appType"`
		AppID     string `json:"appID"`
		AppSecret string `json:"appSecret,omitempty"`
	} `json:"appConfig"`
	RegisterAvailable bool `json:"registerAvailable"`
}

type WechatAuthParam struct {
	EncryptedMobile   string `json:"encryptedMobile"`
	EncryptedUserInfo string `json:"encryptedUserInfo"`
	MobileIV          string `json:"mobileIV"`
	UserInfoIV        string `json:"userInfoIV"`
	Code              string `json:"code"`
	EncryptedSession  string `json:"encryptedSession"`
	AppID             string `json:"appID"`
	RedirectURL       string `json:"redirectURL"`
}

func (a *WechatAuth) Validate() error {
	return nil
}

func (a *WechatAuth) Tip() map[string]any {
	return make(map[string]any)
}

func (a *WechatAuth) CheckExists(siteID, requestID string, txn *sql.Tx, toCheck *user.User) error {

	//todo

	return nil
}

func (a *WechatAuth) login(param *WechatAuthParam) (string, string, interface{}, error) {
	var appID, appType, appSecret string
	for _, appConfig := range a.AppConfig {
		if appConfig.AppID == param.AppID {
			appID = param.AppID
			appType = appConfig.AppType
			appSecret = appConfig.AppSecret
		}
	}
	if appID == "" {
		return "", "", nil, errors.New("需要有效的appID")
	}

	switch appType {
	case util.WECHAT_APP_OPEN:
		if param.EncryptedSession != "" {
			infoData, err := encrypt.Base64Decrypt(param.EncryptedSession)
			if err != nil {
				return "", "", nil, err
			}
			userInfo := new(util.UserInfo)
			if err := json.Unmarshal(infoData, userInfo); err != nil {
				return "", "", nil, err
			}
			return userInfo.Openid, userInfo.Unionid, userInfo, nil
		}
		if param.Code == "" {
			redirect := util.PlatformGetUserCodeRedirectURL(param.AppID, param.RedirectURL)
			log.Println("wechat auth redirect: ", redirect)
			return "", "", map[string]interface{}{
				"retCode":  2,
				"redirect": redirect,
			}, nil
		}
		accessToken, err := util.PlatformGetUserAccessToken(appID, param.Code)
		if err != nil {
			return "", "", nil, err
		}
		userInfo, err := util.GetUserInfo(accessToken.OpenID, accessToken.AccessToken)
		if err != nil {
			return "", "", nil, err
		}
		return userInfo.Openid, userInfo.Unionid, userInfo, nil
	case util.WECHAT_APP_WEB:
		if param.EncryptedSession != "" {
			infoData, err := encrypt.Base64Decrypt(param.EncryptedSession)
			if err != nil {
				return "", "", nil, err
			}
			userInfo := new(util.UserInfo)
			if err := json.Unmarshal(infoData, userInfo); err != nil {
				return "", "", nil, err
			}
			return userInfo.Openid, userInfo.Unionid, userInfo, nil
		}
		if param.Code == "" {
			return "", "", nil, errors.New("微信网站授权登录需要code")
		}
		accessToken, err := util.GetOpenUserAccessToken(appID, appSecret, param.Code)
		if err != nil {
			return "", "", nil, err
		}
		userInfo, err := util.GetUserInfo(accessToken.OpenID, accessToken.AccessToken)
		if err != nil {
			return "", "", nil, err
		}
		return userInfo.Openid, userInfo.Unionid, userInfo, nil
	case util.WECHAT_APP_MINIAPP:
		if param.EncryptedSession != "" {
			sessionData, err := encrypt.Base64Decrypt(param.EncryptedSession)
			if err != nil {
				log.Println("error decrypt wechat encrypted session: ", param.EncryptedSession, err)
				return "", "", nil, err
			}
			sessionKey := new(util.UserSessionKey)
			if err := json.Unmarshal(sessionData, sessionKey); err != nil {
				log.Println("error unmashal decrypted wechat session: ", string(sessionData), err)
				return "", "", nil, err
			}
			return sessionKey.OpenID, sessionKey.UnionID, sessionKey, nil
		}
		if param.Code == "" {
			return "", "", nil, errors.New("需要小程序授权code")
		}
		sessionKey, err := util.PlatformGetUserSessionKey(appID, param.Code)
		if err != nil {
			return "", "", nil, err
		}
		return sessionKey.OpenID, sessionKey.UnionID, sessionKey, nil
	}

	return "", "", nil, errors.New("无效的appType: " + appType)
}

func encryptSession(toEncrypt interface{}) string {
	data, err := json.Marshal(toEncrypt)
	if err != nil {
		log.Println("error encrypt session: ", err)
		return ""
	}
	return encrypt.Base64Encrypt(string(data))
}

func (a *WechatAuth) getUser(siteID string, txn *sql.Tx, appID, unionID, openID string) (*user.User, error) {
	if existUser, err := user.GetUserWithTxn(siteID, txn, fmt.Sprintf("JSON_EXTRACT(user.wechat_info, '$.%s.openid')", appID), openID, true); err != nil {
		if err != user.E_user_not_exists {
			return nil, err
		}
	} else {
		return existUser, nil
	}
	return nil, user.E_user_not_exists
}

func (a *WechatAuth) Register(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {

	var param struct {
		WechatAuthParam
		user.UserInfo
	}
	if err := json.Unmarshal(paramData, &param); err != nil {
		log.Println("error unmarshal wechat auth param: ", string(paramData), err)
		return nil, nil, err
	}

	openID, unionID, info, err := a.login(&param.WechatAuthParam)
	if err != nil {
		return nil, nil, err
	}

	extra := make(map[string]interface{})
	extra["info"] = encryptSession(info)

	if openID == "" {
		log.Println("无法获取openID")
		if ret, ok := info.(map[string]interface{}); ok {
			return nil, ret, nil
		}
		return nil, extra, errors.New("获取微信登录信息失败")
	}

	newUser := &user.User{WechatInfo: make(map[string]interface{}), UserInfo: param.UserInfo}

	if param.EncryptedMobile != "" && param.MobileIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedMobile, sessionKey.SessionKey, param.MobileIV)
			if err != nil {
				log.Println("error decrypt mobile info: ", param.EncryptedMobile, param.MobileIV, sessionKey.SessionKey, err)
				return nil, extra, err
			}
			var mobileInfo util.MobileInfo
			if err := json.Unmarshal(data, &mobileInfo); err != nil {
				log.Println("error unmarshal mobile info: ", string(data), err)
				return nil, extra, err
			}
			extra["mobileInfo"] = mobileInfo
			newUser.Mobile = mobileInfo.PurePhoneNumber
			log.Println("decrypted user mobile: ", mobileInfo)
		} else {
			log.Println("error info not user sessionkey")
		}
	}

	if param.EncryptedUserInfo != "" && param.UserInfoIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedUserInfo, sessionKey.SessionKey, param.UserInfoIV)
			if err != nil {
				log.Println("error decrypt user info: ", err)
				return nil, extra, err
			}
			var userInfo util.MiniAppUserInfo
			if err := json.Unmarshal(data, &userInfo); err != nil {
				log.Println("error unmarshal user info: ", string(data), err)
				return nil, extra, err
			}
			extra["userInfo"] = userInfo
			info = userInfo
			unionID = userInfo.UnionID
			log.Println("decrypted user userInfo: ", userInfo)
		}
	}
	newUser.WechatInfo[param.AppID] = info

	lockKeys := make([]string, 0)

	lockKeys = append(lockKeys, unionID, openID)
	if newUser.Mobile != "" {
		lockKeys = append(lockKeys, newUser.Mobile)
	}

	if err := user.RequestRegisterLock(siteID, requestID, lockKeys, func() error {
		log.Println("register global lock acquired: ", requestID)

		existUser, err := a.getUser(siteID, txn, param.AppID, unionID, openID)
		if err != nil && err != user.E_user_not_exists {
			return err
		}

		if existUser != nil {
			existUser.WechatInfo[param.AppID] = info
			go func() {
				if err := existUser.Update(siteID, nil); err != nil {
					log.Println("error update user wechatinfo: ", err)
				}
			}()
			return e_user_exists
		}

		if newUser.Mobile != "" {
			if existUser, err := user.GetUserWithTxn(siteID, txn, "mobile", newUser.Mobile, true); err != nil {
				if err != user.E_user_not_exists {
					return err
				}
			} else {

				if wechatInfo := existUser.WechatInfo[param.AppID]; wechatInfo != nil {
					return e_user_exists
				}

				newUser.UserID = existUser.UserID
				existUser.WechatInfo = newUser.WechatInfo
				existUser.UserInfo = newUser.UserInfo

				if err := existUser.Update(siteID, txn); err != nil {
					return err
				}

				newUser = existUser

				return nil
			}
		}

		log.Println("new user : ", newUser)

		if err := newUser.Add(siteID, txn); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Println("request lock action fail: ", requestID, err)
		return nil, extra, err
	}

	return newUser, extra, nil
}

func (a *WechatAuth) Login(siteID, requestID string, txn *sql.Tx, paramData []byte) (*user.User, map[string]interface{}, error) {
	var param WechatAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, nil, err
	}

	openID, unionID, info, err := a.login(&param)
	if err != nil {
		return nil, nil, err
	}

	var mobile string

	extra := make(map[string]interface{})
	extra["info"] = encryptSession(info)
	if openID == "" {
		if ret, ok := info.(map[string]interface{}); ok {
			return nil, ret, nil
		}
		return nil, extra, errors.New("获取微信登录信息失败")
	}

	if param.EncryptedMobile != "" && param.MobileIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedMobile, sessionKey.SessionKey, param.MobileIV)
			if err != nil {
				log.Println("error decrypt mobile info: ", err)
				return nil, extra, err
			}
			var mobileInfo util.MobileInfo
			if err := json.Unmarshal(data, &mobileInfo); err != nil {
				log.Println("error unmarshal mobile info: ", string(data), err)
				return nil, extra, err
			}
			extra["mobileInfo"] = mobileInfo
			mobile = mobileInfo.PurePhoneNumber
		}
	}

	if param.EncryptedUserInfo != "" && param.UserInfoIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedUserInfo, sessionKey.SessionKey, param.UserInfoIV)
			if err != nil {
				log.Println("error decrypt user info: ", err)
				return nil, extra, err
			}
			var userInfo util.MiniAppUserInfo
			if err := json.Unmarshal(data, &userInfo); err != nil {
				log.Println("error unmarshal user info: ", string(data), err)
				return nil, extra, err
			}
			extra["userInfo"] = userInfo
			info = userInfo
			unionID = userInfo.UnionID
		}
	}

	existUser, err := a.getUser(siteID, txn, param.AppID, unionID, openID)
	if err != nil {
		if err == user.E_user_not_exists && mobile != "" {
			existUser, err = user.GetUser(siteID, "mobile", mobile)
			if err != nil {
				return nil, extra, err
			}
			return existUser, extra, nil
		}
		return nil, extra, err
	}

	go func() {
		existUser.WechatInfo[param.AppID] = info
		if err := existUser.Update(siteID, nil); err != nil {
			log.Println("error update user wechatinfo: ", err)
		}
	}()

	return existUser, extra, nil
}

func (a *WechatAuth) Bind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {

	var param WechatAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, err
	}

	openID, unionID, info, err := a.login(&param)
	if err != nil {
		return nil, err
	}

	extra := make(map[string]interface{})
	extra["info"] = encryptSession(info)
	if openID == "" {
		if ret, ok := info.(map[string]interface{}); ok {
			return ret, nil
		}
		return extra, errors.New("获取微信登录信息失败")
	}

	if param.EncryptedUserInfo != "" && param.UserInfoIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedUserInfo, sessionKey.SessionKey, param.UserInfoIV)
			if err != nil {
				log.Println("error decrypt user info: ", err)
				return extra, err
			}
			var userInfo util.MiniAppUserInfo
			if err := json.Unmarshal(data, &userInfo); err != nil {
				log.Println("error unmarshal user info: ", string(data), err)
				return extra, err
			}
			extra["userInfo"] = userInfo
			info = userInfo
			unionID = userInfo.UnionID
		}
	}

	var mobile string
	if param.EncryptedMobile != "" && param.MobileIV != "" {
		if sessionKey, ok := info.(*util.UserSessionKey); ok {
			data, err := util.DecryptMiniappData(param.EncryptedMobile, sessionKey.SessionKey, param.MobileIV)
			if err != nil {
				log.Println("error decrypt mobile info: ", err)
				return extra, err
			}
			var mobileInfo util.MobileInfo
			if err := json.Unmarshal(data, &mobileInfo); err != nil {
				log.Println("error unmarshal mobile info: ", string(data), err)
				return extra, err
			}
			extra["mobileInfo"] = mobileInfo
			mobile = mobileInfo.PurePhoneNumber
		}
	}

	wechatExistUser, err := a.getUser(siteID, txn, param.AppID, unionID, openID)
	if err != nil && err != user.E_user_not_exists {
		return extra, err
	}

	if wechatExistUser != nil {
		if wechatExistUser.UserID != existUser.UserID {
			return nil, e_user_exists
		}
	}

	if mobile != "" {
		if mobileExistUser, err := user.GetUserWithTxn(siteID, txn, "mobile", mobile, true); err != nil {
			if err != user.E_user_not_exists {
				return extra, err
			}
		} else if mobileExistUser.UserID != existUser.UserID {
			return extra, e_user_exists
		}

		existUser.Mobile = mobile
	}

	existUser.WechatInfo[param.AppID] = info
	if err := existUser.Update(siteID, txn); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *WechatAuth) UnBind(siteID string, txn *sql.Tx, existUser *user.User, paramData []byte) (map[string]interface{}, error) {

	var param WechatAuthParam
	if err := json.Unmarshal(paramData, &param); err != nil {
		return nil, err
	}

	openID, _, info, err := a.login(&param)
	if err != nil {
		return nil, err
	}

	extra := make(map[string]interface{})
	extra["info"] = encryptSession(info)
	if openID == "" {
		if ret, ok := info.(map[string]interface{}); ok {
			return ret, nil
		}
		return extra, errors.New("获取微信登录信息失败")
	}

	if info, exists := existUser.WechatInfo[param.AppID]; exists {
		openid, exists := info.(map[string]interface{})["openid"]
		if !exists {
			openid, exists = info.(map[string]interface{})["openId"]
		}
		if !exists || openid.(string) != openID {
			return extra, errors.New("解绑微信账号不匹配")
		}

		delete(existUser.WechatInfo, param.AppID)

		if err := existUser.Update(siteID, txn); err != nil {
			return extra, err
		}

	} else {
		return extra, errors.New("未绑定微信账号")
	}

	return extra, nil
}
