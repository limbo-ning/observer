package auth

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/mission"
	"obsessiontech/environment/site"
	"obsessiontech/environment/user"
	"obsessiontech/environment/user/auth/engine"
)

var e_invalid_method = errors.New("无效的验证方式")
var e_user_inactive = errors.New("账户状态异常")

const REFERER_NA = "NA"

func getEffectiveSiteID(siteID string) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, user.MODULE_USER, true)
	if err != nil {
		return "", err
	}
	return moduleSite, nil
}

func IsLogined(siteID, clientIP, token, referer string) (int, string) {

	log.Println("check login: ", siteID, clientIP, token)

	var err error
	siteID, err = getEffectiveSiteID(siteID)
	if err != nil {
		log.Println("no siteID")
		return 0, ""
	}

	if token == "" {
		log.Println("no token")
		return 0, composeToken(siteID, clientIP, 0)
	}
	tokenSiteID, tokenUserID, tokenIP, loginTime, err := decomposeToken(token)
	if err != nil {
		log.Println("fail decompose token: ", err, token)
		return 0, composeToken(siteID, clientIP, 0)
	}
	if siteID != tokenSiteID {
		log.Println("siteID not match: ", siteID, tokenSiteID)
		return 0, composeToken(siteID, clientIP, 0)
	}

	if tokenUserID == 0 {
		log.Println("no token user id")
		return 0, token
	}

	if tokenIP == Config.SuperIP {
		return tokenUserID, token
	}

	// if clientIP != tokenIP {
	// 	log.Println("client ip not match: ", clientIP, tokenIP)
	// 	return 0, composeToken(siteID, clientIP, 0)
	// }

	userModule, err := user.GetUserModule(siteID)
	if err != nil {
		log.Println("error auth checking get user module: ", err)
		return tokenUserID, token
	}

	if len(userModule.Referers) > 0 && referer != REFERER_NA {
		checked := false
		for _, r := range userModule.Referers {
			if strings.Contains(referer, r) {
				checked = true
				break
			}
		}
		if !checked {
			log.Println("referer not valid: ", referer)
			return 0, composeToken(siteID, clientIP, 0)
		}
	}

	if userModule.LoginExpireMin > 0 {
		expire := loginTime.Add(userModule.LoginExpireMin * time.Minute)
		if expire.Before(time.Now()) {
			log.Println("login expired: ", loginTime, time.Now(), userModule.LoginExpireMin)
			return 0, composeToken(siteID, clientIP, 0)
		}

		if time.Until(expire).Minutes() < (userModule.LoginExpireMin*time.Minute).Minutes()*0.3 {
			log.Println("refresh login token")
			token = composeToken(siteID, clientIP, tokenUserID)
		}
	}

	log.Println("logined: ", siteID, clientIP, token, tokenUserID)
	return tokenUserID, token
}

func Create(siteID, requestID string, toCreate *user.User) error {
	auths := engine.GetAuthMethod(siteID)

	return datasource.Txn(func(txn *sql.Tx) {

		if err := toCreate.Add(siteID, txn); err != nil {
			panic(err)
		}

		for a := range auths {
			auth, err := engine.GetAuth(siteID, a)
			if err != nil {
				panic(err)
			}

			if err := auth.CheckExists(siteID, requestID, txn, toCreate); err != nil {
				panic(err)
			}

		}

		userModule, err := user.GetUserModuleWithTxn(siteID, txn, false)
		if err != nil {
			panic(err)
		}

		for _, mid := range userModule.PostRegisterMission {
			if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: toCreate.UserID, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
				log.Println("error complete register mission: UID-", toCreate.UserID, err)
				panic(err)
			}
		}

	})
}

func UpdateInfo(siteID, requestID string, actionAuth authority.ActionAuthSet, toUpdate *user.User) error {
	auths := engine.GetAuthMethod(siteID)

	if actionAuth.GetUID() != toUpdate.UserID {
		if !actionAuth.CheckAction(user.ACTION_ADMIN_EDIT) {
			return errors.New("无权限")
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {

		exists, err := user.GetUserWithTxn(siteID, txn, "id", toUpdate.UserID, true)
		if err != nil {
			panic(err)
		}

		for a := range auths {
			if a == engine.SMS_AUTH {
				if toUpdate.Mobile != "" {
					if !actionAuth.CheckAction(user.ACTION_ADMIN_EDIT) {
						panic(errors.New("无权限"))
					}
				}
			}

			auth, err := engine.GetAuth(siteID, a)
			if err != nil {
				panic(err)
			}

			if err := auth.CheckExists(siteID, requestID, txn, toUpdate); err != nil {
				panic(err)
			}
		}

		if toUpdate.RealName != "" {
			exists.RealName = strings.TrimSpace(toUpdate.RealName)
		}
		if toUpdate.IDentityNo != "" {
			exists.IDentityNo = strings.TrimSpace(toUpdate.IDentityNo)
		}
		if toUpdate.Organization != "" {
			exists.Organization = strings.TrimSpace(toUpdate.Organization)
		}

		if toUpdate.Username != "" {
			exists.Username = strings.TrimSpace(toUpdate.Username)
		}
		if toUpdate.Mobile != "" {
			exists.Mobile = strings.TrimSpace(toUpdate.Mobile)
		}
		if toUpdate.Email != "" {
			exists.Email = strings.TrimSpace(toUpdate.Email)
		}

		if toUpdate.Stats != nil {
			exists.Stats = toUpdate.Stats
		}

		if err := exists.Update(siteID, txn); err != nil {
			panic(err)
		}

		toUpdate = exists
	})
}

func Register(siteID, requestID, clientIP, method string, data []byte) (string, map[string]interface{}, error) {

	var err error
	siteID, err = getEffectiveSiteID(siteID)
	if err != nil {
		return "", nil, err
	}

	var registered *user.User
	var result map[string]interface{}

	if MockConfig.MockRegisterUserID > 0 {

		if err := datasource.Txn(func(txn *sql.Tx) {
			registered, err = user.GetUserWithTxn(siteID, txn, "id", MockConfig.MockRegisterUserID, false)
			if err != nil {
				panic(err)
			}

			userModule, err := user.GetUserModuleWithTxn(siteID, txn, false)
			if err != nil {
				panic(err)
			}

			for _, mid := range userModule.PostRegisterMission {
				if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: registered.UserID, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
					log.Println("error complete register mission: UID-", registered.UserID, err)
					panic(err)
				}
			}

		}); err != nil {
			return "", nil, err
		}
		return composeToken(siteID, clientIP, MockConfig.MockLoginUserID), result, nil
	}

	auth, err := engine.GetAuth(siteID, method)
	if err != nil {
		return "", nil, err
	}

	if err := datasource.Txn(func(txn *sql.Tx) {
		u, ret, err := auth.Register(siteID, requestID, txn, data)
		result = ret
		if err != nil {
			panic(err)
		}

		if u == nil {
			log.Println("error register nil user")
			panic(errors.New("注册未成功，请按提示操作"))
		}

		registered = u

		if registered.UserID == 0 {
			panic(errors.New("注册用户未成功"))
		}

		userModule, err := user.GetUserModuleWithTxn(siteID, txn, false)
		if err != nil {
			panic(err)
		}

		for _, mid := range userModule.PostRegisterMission {
			if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: registered.UserID, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
				log.Println("error complete register mission: UID-", registered.UserID, err)
				panic(err)
			}
		}

	}); err != nil {

		if err == engine.E_toggle_method {
			return Login(siteID, requestID, clientIP, method, data)
		}

		return "", result, err
	}

	if registered != nil {
		return composeToken(siteID, clientIP, registered.UserID), result, nil
	}

	return "", result, nil

}

func Login(siteID, requestID, clientIP, method string, data []byte) (string, map[string]interface{}, error) {

	var err error
	siteID, err = getEffectiveSiteID(siteID)
	if err != nil {
		log.Println("error get effective siteID", err)
		return "", nil, err
	}

	var logined *user.User
	var result map[string]interface{}

	if MockConfig.MockLoginUserID > 0 {
		if err := datasource.Txn(func(txn *sql.Tx) {
			logined, err = user.GetUserWithTxn(siteID, txn, "id", MockConfig.MockLoginUserID, false)
			if err != nil {
				log.Println("error get user:", err)
				panic(err)
			}

			userModule, err := user.GetUserModuleWithTxn(siteID, txn, false)
			if err != nil {
				log.Println("error get user module:", err)
				panic(err)
			}

			for _, mid := range userModule.PostLoginMission {
				if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: logined.UserID, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
					log.Println("error complete login mission: UID-", logined.UserID, err)
				}
			}
		}); err != nil {
			return "", nil, err
		}
		return composeToken(siteID, clientIP, MockConfig.MockLoginUserID), result, nil
	}

	auth, err := engine.GetAuth(siteID, method)
	if err != nil {
		return "", nil, err
	}

	if err := datasource.Txn(func(txn *sql.Tx) {
		u, ret, err := auth.Login(siteID, requestID, txn, data)
		result = ret
		if err != nil {
			log.Println("error login", err)
			panic(err)
		}

		if u == nil {
			if _, exists := result["retCode"]; !exists {
				result["retCode"] = 1
			}
			return
		}

		logined = u
		if logined.UserID == 0 {
			panic(errors.New("登录用户未成功"))
		}

		userModule, err := user.GetUserModuleWithTxn(siteID, txn, false)
		if err != nil {
			log.Println("error get user module", err)
			panic(err)
		}

		for _, mid := range userModule.PostLoginMission {
			if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: logined.UserID, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
				log.Println("error complete login mission: UID-", logined.UserID, err)
			}
		}
	}); err != nil {

		log.Println("error login: ", err)

		if err == engine.E_toggle_method {
			return Register(siteID, requestID, clientIP, method, data)
		}

		return "", result, err
	}

	if logined == nil {
		return "", result, errors.New("未完成登录")
	}

	if logined.Status == user.USER_INACTIVE {
		return "", nil, e_user_inactive
	}

	logined.UpdateLoginTime(siteID)
	return composeToken(siteID, clientIP, logined.UserID), result, nil
}

func Logout(siteID string, userID int) {
	//do nothing
}

func Bind(siteID, method string, userID int, data []byte) (map[string]interface{}, error) {

	var err error
	siteID, err = getEffectiveSiteID(siteID)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}

	if err := datasource.Txn(func(txn *sql.Tx) {
		auth, err := engine.GetAuth(siteID, method)
		if err != nil {
			panic(err)
		}

		existUser, err := user.GetUserWithTxn(siteID, txn, "id", userID, true)
		if err != nil {
			panic(err)
		}

		ret, err := auth.Bind(siteID, txn, existUser, data)
		result = ret
		if err != nil {
			panic(err)
		}
	}); err != nil {
		log.Println("error bind: ", err)
		return nil, err
	}

	return result, nil
}

func UnBind(siteID, method string, userID int, data []byte) (map[string]interface{}, error) {

	var err error
	siteID, err = getEffectiveSiteID(siteID)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}

	if err := datasource.Txn(func(txn *sql.Tx) {
		auth, err := engine.GetAuth(siteID, method)
		if err != nil {
			panic(err)
		}

		existUser, err := user.GetUserWithTxn(siteID, txn, "id", userID, true)
		if err != nil {
			panic(err)
		}

		ret, err := auth.UnBind(siteID, txn, existUser, data)
		result = ret
		if err != nil {
			panic(err)
		}
	}); err != nil {
		return nil, err
	}

	return result, nil
}
