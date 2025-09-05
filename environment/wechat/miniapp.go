package wechat

import (
	"bytes"
	"encoding/json"
	"image/jpeg"
	"image/png"

	"obsessiontech/wechat/util"
)

type MiniAppCodeParam struct {
	*util.MiniAppCodeParam
	AppID     string `json:"appID"`
	ImageType string `json:"imageType"`
}

func GetMiniAppCodePermanent(param *MiniAppCodeParam) ([]byte, error) {
	accessToken, err := GetAgentAccessToken(param.AppID)
	if err != nil {
		return nil, err
	}

	imageData, err := util.PlatformGetMiniAppCodePermanent(param.MiniAppCodeParam, accessToken)

	switch param.ImageType {
	case "jpg":
		fallthrough
	case "jpeg":
		img, err := png.Decode(bytes.NewReader(imageData))
		if err != nil {
			return nil, err
		}
		buff := new(bytes.Buffer)
		if err := jpeg.Encode(buff, img, nil); err != nil {
			return nil, err
		}
		imageData = buff.Bytes()
	}

	return imageData, err
}

func GetMiniAppCodeUnlimit(param *MiniAppCodeParam) ([]byte, error) {
	accessToken, err := GetAgentAccessToken(param.AppID)
	if err != nil {
		return nil, err
	}

	imageData, err := util.PlatformGetMiniAppCodeUnlimit(param.MiniAppCodeParam, accessToken)

	switch param.ImageType {
	case "jpg":
		fallthrough
	case "jpeg":
		img, err := png.Decode(bytes.NewReader(imageData))
		if err != nil {
			return nil, err
		}
		buff := new(bytes.Buffer)
		if err := jpeg.Encode(buff, img, nil); err != nil {
			return nil, err
		}
		imageData = buff.Bytes()
	}

	return imageData, err
}

func UploadMiniAppTemplateCode(appID string, templateID, userVersion, userDescription string, extJson map[string]interface{}) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	return util.PlatformUploadMiniAppTemplateCode(accessToken, templateID, userVersion, userDescription, extJson)
}

func GetMiniAppPageList(appID string) ([]string, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformGetMiniAppPages(accessToken)
}

func GetMiniAppPageCategoryList(appID string) ([]map[string]interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformGetMiniAppPageCategories(accessToken)
}

func SubmitMiniAppMedia(appID string, data []byte) (string, string, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return "", "", err
	}

	return util.PlatformSubmitMiniAppMedia(accessToken, data)
}

func SubmitMiniApp(appID string, param *util.SubmitMiniAppReq) (string, error) {

	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return "", err
	}

	return util.PlatformSubmitMiniApp(accessToken, param)
}

func RetreatMiniAppSubmit(appID string) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	return util.PlatformRetreatMiniAppSubmit(accessToken)
}

func SpeedupMiniAppSubmit(appID, auditID string) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	return util.PlatformSpeedupMiniAppSubmit(accessToken, auditID)
}

func QueryMiniAppSubmit(appID, auditID string) (map[string]interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	data, err := util.PlatformQueryMiniAppSubmit(accessToken, auditID)
	if err != nil {
		return nil, err
	}

	var status map[string]interface{}

	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func ReleaseMiniApp(appID string) error {

	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	return util.PlatformReleaseMiniApp(accessToken)
}

func PreviewMiniApp(appID, path string) ([]byte, error) {

	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformPreviewMiniApp(accessToken, path)
}

func GetMiniAppDomain(appID string) (*util.DomainRet, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformSetMiniAppDomain(accessToken, "get", nil)
}

func SetMiniAppDomain(appID string, param *util.DomainReq) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	_, err = util.PlatformSetMiniAppDomain(accessToken, "set", param)
	return err
}

func GetMiniAppWebviewDomain(appID string) (*util.WebviewDomainRet, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformSetMiniAppWebviewDomain(accessToken, "get", nil)
}

func SetMiniAppWebviewDomain(appID string, param *util.WebviewDomainReq) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	_, err = util.PlatformSetMiniAppWebviewDomain(accessToken, "set", param)
	return err
}

func SetMiniAppPrivacySetting(appID string, param *util.SetPrivacySettingReq) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	_, err = util.PlatformSetMiniAppPrivacySetting(accessToken, param)
	return err
}

func QueryMiniAppPrivacySetting(appID string, param *util.QueryPrivacySettingReq) (interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformQueryMiniAppPrivacySetting(accessToken, param)
}

func ApplyMiniAppApi(appID string, param *util.ApplyApiParam) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	_, err = util.PlatformApplyMiniAppAPI(accessToken, param)
	return err
}

func GetMiniAppApiSetting(appID string) (interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformGetMiniAppApiSetting(accessToken)
}

func GetMiniAppApiReview(appID string) (interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	return util.PlatformGetMiniAppApiReview(accessToken)
}
