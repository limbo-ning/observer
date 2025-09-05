package wechat

import "obsessiontech/wechat/util"

func CreateOpenServiceAccount(appID string) (string, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return "", err
	}
	return util.PlatformCreateOpenServiceAccount(accessToken, appID)
}

func GetOpenServiceAccount(appID string) (string, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return "", err
	}
	return util.PlatformGetOpenServiceAccount(accessToken, appID)
}

func BindOpenServiceAccount(appID, openAppID string) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}
	return util.PlatformBindOpenServiceAccount(accessToken, appID, openAppID)
}

func UnbindOpenServiceAccount(appID, openAppID string) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}
	return util.PlatformUnbindOpenServiceAccount(accessToken, appID, openAppID)
}
