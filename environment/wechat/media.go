package wechat

import (
	"encoding/json"
	"io"

	"obsessiontech/wechat/util"
)

func UploadMedia(appID string, mediaType string, data []byte) (string, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return "", err
	}

	return util.PlatformUploadMedia(accessToken, mediaType, data)
}

func DownloadMedia(appID string, writer io.Writer, mediaID, amrConvertTo string, setContentType, setFileName func(string)) error {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return err
	}

	return util.PlatformDownloadMedia(accessToken, writer, mediaID, amrConvertTo, setContentType, setFileName)
}

func DownloadMaterial(appID string, mediaID string) (map[string]interface{}, error) {
	accessToken, err := GetAgentAccessToken(appID)
	if err != nil {
		return nil, err
	}

	data, err := util.PlatformDownloadMaterial(accessToken, mediaID)
	if err != nil {
		return nil, err
	}

	var material map[string]interface{}

	if err := json.Unmarshal(data, &material); err != nil {
		return nil, err
	}

	return material, nil
}
