package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"obsessiontech/common/util"
)

type MiniAppCodeParam struct {
	Scene     string    `json:"scene"`
	Page      string    `json:"page,omitempty"`
	Path      string    `json:"path,omitempty"`
	Width     int       `json:"width,omitempty"`
	AutoColor bool      `json:"auto_color,omitempty"`
	LineColor lineColor `json:"line_color,omitempty"`
	IsHyaline bool      `json:"is_hyaline,omitempty"`
}

type lineColor struct {
	R string `json:"r"`
	G string `json:"g"`
	B string `json:"b"`
}

type ErrorRet struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func GetMiniAppCodePermanent(param *MiniAppCodeParam) ([]byte, error) {
	return PlatformGetMiniAppCodePermanent(param, GetMiniAppAccessToken())
}

func GetMiniAppCodeUnlimit(param *MiniAppCodeParam) ([]byte, error) {
	return PlatformGetMiniAppCodeUnlimit(param, GetMiniAppAccessToken())
}

func PlatformGetMiniAppCodePermanent(param *MiniAppCodeParam, accessToken string) ([]byte, error) {
	dataBytes, _ := util.UnsafeJsonString(&param)

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacode?access_token=%s", accessToken)

	client := &http.Client{}

	log.Println("request miniapp code:", string(dataBytes))
	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.Println("error request get miniapp code permanent: ", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request get miniapp code permanent: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request get miniapp code permanent: ", err)
		return nil, err
	}

	if bytes.Contains(body, []byte("errmsg")) {
		var ret ErrorRet
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to get miniapp code permanent. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
			return nil, errors.New(ret.ErrMsg)
		}
	}

	return body, nil
}

func PlatformGetMiniAppCodeUnlimit(param *MiniAppCodeParam, accessToken string) ([]byte, error) {

	dataBytes, _ := util.UnsafeJsonString(&param)

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s", accessToken)

	client := &http.Client{}

	log.Println("request miniapp code:", string(dataBytes))
	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.Println("error request get miniapp code unlimit: ", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request get miniapp code unlimit: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request get miniapp code unlimit: ", err)
		return nil, err
	}

	if bytes.Contains(body, []byte("errmsg")) {
		var ret ErrorRet
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to get miniapp code unlimit. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
			return nil, errors.New(ret.ErrMsg)
		}
	}

	return body, nil
}

type MiniAppTemplateListRet struct {
	ErrCode      int                      `json:"errcode"`
	ErrMsg       string                   `json:"errmsg"`
	TemplateList []map[string]interface{} `json:"template_list"`
}

func PlatformGetMiniAppTemplateList() ([]map[string]interface{}, error) {

	accessToken, err := GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/gettemplatelist?access_token=%s", accessToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get miniapp template list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp template list:", err)
		return nil, err
	}

	var ret MiniAppTemplateListRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return ret.TemplateList, nil
}

func PlatformUploadMiniAppTemplateCode(accessToken, templateID, userVersion, userDescription string, extJson map[string]interface{}) error {

	var param struct {
		TemplateID      string `json:"template_id"`
		ExtJSON         string `json:"ext_json"`
		UserVersion     string `json:"user_version"`
		UserDescription string `json:"user_desc"`
	}
	extJSONBytes, err := util.UnsafeJsonString(extJson)
	if err != nil {
		return err
	}

	param.TemplateID = templateID
	param.ExtJSON = string(extJSONBytes)
	param.UserVersion = userVersion
	param.UserDescription = userDescription

	dataBytes, _ := util.UnsafeJsonString(&param)

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/commit?access_token=%s", accessToken)

	client := &http.Client{}

	log.Println("upload miniapp template code:", string(dataBytes))
	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.Println("error upload miniapp template code: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error upload miniapp template code: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error upload miniapp template code: ", err)
		return err
	}

	var ret ErrorRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to upload miniapp template code. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PlatformGetMiniAppPages(accessToken string) ([]string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/get_page?access_token=%s", accessToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get miniapp page list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp page list:", err)
		return nil, err
	}

	var ret struct {
		ErrorRet
		PageList []string `json:"page_list"`
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to get miniapp page list. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return ret.PageList, nil

}

func PlatformGetMiniAppPageCategories(accessToken string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/get_category?access_token=%s", accessToken)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get miniapp page category list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp page category list:", err)
		return nil, err
	}

	var ret struct {
		ErrorRet
		CategoryList []map[string]interface{} `json:"category_list"`
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to get miniapp page category list. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return ret.CategoryList, nil

}

type SubmitMiniAppReq struct {
	ItemList      []*SubmitMiniAppPageItem  `json:"item_list,omitempty"`
	VersionDesc   string                    `json:"version_desc,omitempty"`
	PreviewInfo   *SubmitMiniAppPreviewInfo `json:"preview_info,omitempty"`
	FeedbackInfo  string                    `json:"feedback_info,omitempty"`
	FeedbackStuff string                    `json:"feedback_stuff,omitempty"`
}

type SubmitMiniAppPageItem struct {
	Address     string `json:"address,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Title       string `json:"title,omitempty"`
	FirstClass  string `json:"first_class,omitempty"`
	FirstID     string `json:"first_id,omitempty"`
	SecondClass string `json:"second_class,omitempty"`
	SecondID    string `json:"second_id,omitempty"`
	ThirdClass  string `json:"third_class,omitempty"`
	ThirdID     string `json:"third_id,omitempty"`
}

type SubmitMiniAppPreviewInfo struct {
	PicIDList   []string `json:"pic_id_list"`
	VidioIDList []string `json:"video_id_list"`
}

func PlatformSubmitMiniAppMedia(accessToken string, data []byte) (string, string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/submit_audit?access_token=%s", accessToken)

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Println("error submit miniapp media: ", err)
		return "", "", err
	}

	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error submit miniapp media: ", err)
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error submit miniapp media: ", err)
		return "", "", err
	}

	var ret struct {
		ErrorRet
		MediaType string `json:"type"`
		MediaID   string `json:"mediaid"`
	}
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to submit miniapp media. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return "", "", errors.New(ret.ErrMsg)
	}

	return ret.MediaType, ret.MediaID, nil
}

func PlatformSubmitMiniApp(accessToken string, param *SubmitMiniAppReq) (string, error) {

	dataBytes, _ := util.UnsafeJsonString(&param)

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/submit_audit?access_token=%s", accessToken)

	client := &http.Client{}

	log.Println("submit miniapp:", string(dataBytes))
	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.Println("error submit miniapp: ", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error submit miniapp: ", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error submit miniapp: ", err)
		return "", err
	}

	log.Println("submit miniapp ret: ", string(body))

	var ret struct {
		ErrorRet
		AuditID string `json:"auditid"`
	}
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to submit miniapp. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return "", errors.New(ret.ErrMsg)
	}

	return ret.AuditID, nil
}

func PlatformRetreatMiniAppSubmit(accessToken string) error {
	var resp *http.Response
	var err error

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/undocodeaudit?access_token=%s", accessToken)
	resp, err = http.Get(url)

	if err != nil {
		log.Println("error retreat miniapp submit: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error retreat miniapp submit: ", err)
		return err
	}

	var ret ErrorRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("error failed to retreat miniapp submit. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PlatformSpeedupMiniAppSubmit(accessToken, auditID string) error {

	var resp *http.Response
	var err error

	client := &http.Client{}
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/speedupaudit?access_token=%s", accessToken)

	param := map[string]interface{}{
		"auditid": auditID,
	}

	dataBytes, _ := json.Marshal(&param)

	req, err := http.NewRequest("POST", url, bytes.NewReader(dataBytes))
	if err != nil {
		log.Println("error speedup miniapp submit: ", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)

	if err != nil {
		log.Println("error speedup miniapp submit: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error speedup miniapp submit: ", err)
		return err
	}

	var ret ErrorRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("error failed to speedup miniapp submit. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PlatformQueryMiniAppSubmit(accessToken, auditID string) ([]byte, error) {

	var resp *http.Response
	var err error

	if auditID == "" {
		url := fmt.Sprintf("https://api.weixin.qq.com/wxa/get_latest_auditstatus?access_token=%s", accessToken)
		resp, err = http.Get(url)
	} else {
		client := &http.Client{}
		url := fmt.Sprintf("https://api.weixin.qq.com/wxa/get_auditstatus?access_token=%s", accessToken)

		param := map[string]interface{}{
			"auditid": auditID,
		}

		dataBytes, _ := json.Marshal(&param)

		var req *http.Request
		req, err = http.NewRequest("POST", url, bytes.NewReader(dataBytes))
		if err != nil {
			log.Println("error query miniapp submit: ", err)
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
	}

	if err != nil {
		log.Println("error query miniapp submit: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error query miniapp submit: ", err)
		return nil, err
	}

	var ret ErrorRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("error failed to query miniapp submit. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return body, nil
}

func PlatformReleaseMiniApp(accessToken string) error {

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/release?access_token=%s", accessToken)

	client := &http.Client{}

	log.Println("release miniapp")
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		log.Println("error release miniapp: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error release miniapp: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error release miniapp: ", err)
		return err
	}

	log.Println("release miniapp ret: ", string(body))

	var ret ErrorRet
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to release miniapp. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return errors.New(ret.ErrMsg)
	}

	return nil
}

func PlatformPreviewMiniApp(accessToken, path string) ([]byte, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/get_qrcode?access_token=%s&path=%s", accessToken, url.QueryEscape(path))

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error preview miniapp:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error preview miniapp:", err)
		return nil, err
	}

	if bytes.Contains(body, []byte("errmsg")) {
		var ret ErrorRet
		json.Unmarshal(body, &ret)

		if ret.ErrCode > 0 {
			log.Printf("failed to preview miniapp. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
			return nil, errors.New(ret.ErrMsg)
		}
	}

	return body, nil
}

type WebviewDomainReq struct {
	WebviewDomain []string `json:"webviewdomain"`
}

type WebviewDomainRet struct {
	ErrorRet
	WebviewDomainReq
}

func PlatformSetMiniAppWebviewDomain(accessToken, action string, param *WebviewDomainReq) (*WebviewDomainRet, error) {
	var ret WebviewDomainRet

	req := make(map[string]interface{})
	if action != "" {
		req["action"] = action
	}
	if param != nil && len(param.WebviewDomain) > 0 {
		req["webviewdomain"] = param.WebviewDomain
	}

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/wxa/setwebviewdomain?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error "+action+" webview domain:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error "+action+" webview domain:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

type DomainReq struct {
	RequestDomain   []string `json:"requestdomain"`
	WsRequestDomain []string `json:"wsrequestdomain"`
	UploadDomain    []string `json:"uploaddomain"`
	DownloadDomain  []string `json:"downloaddomain"`
}

type DomainRet struct {
	ErrorRet
	DomainReq
}

func PlatformSetMiniAppDomain(accessToken, action string, param *DomainReq) (*DomainRet, error) {
	var ret DomainRet

	req := make(map[string]interface{})
	req["action"] = action

	if action != "get" && param != nil {
		req["requestdomain"] = param.RequestDomain
		req["wsrequestdomain"] = param.WsRequestDomain
		req["uploaddomain"] = param.UploadDomain
		req["downloaddomain"] = param.DownloadDomain
	}

	reqBytes, _ := json.Marshal(req)

	resp, err := http.Post("https://api.weixin.qq.com/wxa/modify_domain?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error "+action+" webview domain:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error "+action+" webview domain:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

type SetPrivacySettingReq struct {
	OwnerSetting *privacyOwnerSetting `json:"owner_setting"`
	PrivacyVer   int                  `json:"privacy_ver,omitempty"`
	SettingList  []*privacySetting    `json:"setting_list,omitempty"`
}

type privacyOwnerSetting struct {
	ContactEmail         string `json:"contact_email"`
	ContactPhone         string `json:"contact_phone"`
	ContactQQ            string `json:"contact_qq"`
	ContactWeixin        string `json:"contact_weixin"`
	ExtFileMediaID       string `json:"ext_file_media_id"`
	NoticeMethod         string `json:"notice_method"`
	StoreExpireTimestamp string `json:"store_expire_timestamp"`
}

type privacySetting struct {
	PrivacyKey  string `json:"privacy_key"`
	PrivacyText string `json:"privacy_text"`
}

type SetPrivacySettingRet struct {
	ErrorRet
}

func PlatformSetMiniAppPrivacySetting(accessToken string, param *SetPrivacySettingReq) (*SetPrivacySettingRet, error) {
	var ret SetPrivacySettingRet

	reqBytes, _ := json.Marshal(param)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/setprivacysetting?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error set miniapp privacy setting:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error set miniapp privacy setting:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

type QueryPrivacySettingReq struct {
	PrivacyVer int `json:"privacy_ver,omitempty"`
}

func PlatformQueryMiniAppPrivacySetting(accessToken string, param *QueryPrivacySettingReq) (interface{}, error) {
	var ret ErrorRet

	reqBytes, _ := json.Marshal(param)

	resp, err := http.Post("https://api.weixin.qq.com/cgi-bin/component/getprivacysetting?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error query miniapp privacy setting:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error query miniapp privacy setting:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	result := make(map[string]interface{})
	json.Unmarshal(body, &result)

	return result, nil
}

type ApplyApiParam struct {
	ApiName   string   `json:"api_name"`
	Content   string   `json:"content"`
	UrlList   []string `json:"url_list"`
	PicList   []string `json:"pic_list"`
	VideoList []string `json:"video_list"`
}

type ApplyApiRet struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	AuditID int    `json:"audit_id"`
}

func PlatformApplyMiniAppAPI(accessToken string, param *ApplyApiParam) (*ApplyApiRet, error) {

	var ret ApplyApiRet

	reqBytes, _ := json.Marshal(param)

	resp, err := http.Post("https://api.weixin.qq.com/wxa/security/apply_privacy_interface?access_token="+accessToken, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		log.Println("error apply miniapp api:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error apply miniapp api:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	return &ret, nil
}

func PlatformGetMiniAppApiSetting(accessToken string) (interface{}, error) {
	var ret ErrorRet

	resp, err := http.Get("https://api.weixin.qq.com/wxa/security/get_privacy_interface?access_token=" + accessToken)
	if err != nil {
		log.Println("error get miniapp api list:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp api list:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	result := make(map[string]interface{})
	json.Unmarshal(body, &result)

	return result, nil
}

func PlatformGetMiniAppApiReview(accessToken string) (interface{}, error) {
	var ret ErrorRet

	resp, err := http.Get("https://api.weixin.qq.com/wxa/security/get_code_privacy_info?access_token=" + accessToken)
	if err != nil {
		log.Println("error get miniapp api review:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp api review:", err)
		return nil, err
	}

	json.Unmarshal(body, &ret)

	if ret.ErrCode > 0 {
		return nil, errors.New(ret.ErrMsg)
	}

	result := make(map[string]interface{})
	json.Unmarshal(body, &result)

	return result, nil
}
