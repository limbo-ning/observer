package wechat

import (
	"errors"
	"log"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/wechat/util"
)

var Config struct {
	IsWechatPlatformHost bool
}

func init() {

	config.GetConfig("config.yaml", &Config)

	if Config.IsWechatPlatformHost {
		refreshTick := time.Tick(12 * time.Hour)

		go func() {
			if err := syncAuthorizers(); err != nil {
				log.Println("error ticker sync authorizers: ", err)
			}

			for range refreshTick {
				if err := syncAuthorizers(); err != nil {
					log.Println("error ticker sync authorizers: ", err)
				}
			}
		}()
	}
}

func syncAuthorizers() error {
	offset := 0
	count := 500

	for {
		listRet, err := util.GetPlatformAuthorizerList(offset, count)
		if err != nil {
			return err
		}

		for _, authorizer := range listRet.List {
			a, err := GetAgent(authorizer.AuthorizerAppID)
			if err != nil {
				if err == e_not_exist {
					a = &WechatAgent{
						AppID: authorizer.AuthorizerAppID,
					}
					a.AuthorizerRefreshToken = authorizer.RefreshToken

					a.Status = AGENT_AUTHORIZED
					if err := a.Add(); err != nil {
						continue
					}
				} else {
					continue
				}
			}

			a.AuthorizerRefreshToken = authorizer.RefreshToken
			a.RefreshAuthorization()
			a.RefreshInfo()
		}

		offset += len(listRet.List)

		if listRet.TotalCount <= offset {
			break
		}
	}

	return nil
}

// func refreshAuthorizerAccessToken() error {
// 	rows, err := datasource.GetConn().Query(`
// 		SELECT
// 			id, type, status, access_token, refresh_token, expire_time, app_info
// 		FROM
// 			c_wechat_agent
// 		WHERE
// 			expire_time < Now() AND status = ?
// 	`, AGENT_AUTHORIZED)
// 	if err != nil {
// 		return err
// 	}

// 	defer rows.Close()

// 	for rows.Next() {
// 		var a WechatAgent
// 		if err := a.scan(rows); err != nil {
// 			log.Println("error refresh scanning: ", err)
// 		} else {
// 			if err := a.RefreshAuthorization(); err != nil {
// 				log.Println("error refresh authorizer access token: ", a.AppID, err)
// 			}
// 		}
// 	}

// 	return nil
// }

func ReceiveAuthorizationPush(timestamp int, msgSignature, nonce, encryptType string, data []byte) error {

	push, err := util.ReceivePlatformAuthorizationPush(timestamp, msgSignature, nonce, encryptType, data)
	if err != nil {
		return err
	}

	switch push.InfoType {
	case util.PLATFORM_AUTHORIZED:
		fallthrough
	case util.PLATFORM_UPDATEAUTHORIZED:
		if push.AuthorizationCode != "" {
			log.Println("wechat agent authorize after authorization push")
			if err := Authorize("", push.AuthorizationCode); err != nil {
				return err
			}
		}
	case util.PLATFORM_UNAUTHORIZED:
		agent, err := GetAgent(push.AuthorizerAppid)
		if err != nil {
			return err
		}
		agent.Status = AGENT_CANCELED
		if err := agent.Update(); err != nil {
			return err
		}
	case util.PLATFORM_VERIFY_TICKET:
	default:
		return errors.New("unknown authorization notice info type: " + push.InfoType)
	}

	return nil
}
