package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var reg *regexp.Regexp

func init() {
	reg = regexp.MustCompile("filename=\"(.*)\"")
}

func extractFilename(header string) string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("error extract filename :", header)
		}
	}()
	matches := reg.FindAllStringSubmatch(header, 1)
	return matches[0][1]
}

func convertAMR(amr []byte, fileName, toFormat string) ([]byte, string, error) {

	if err := ioutil.WriteFile(os.TempDir()+"/"+fileName, amr, os.ModePerm); err != nil {
		return nil, "", err
	}
	defer os.Remove(os.TempDir() + "/" + fileName)

	toFileName := strings.Replace(fileName, "amr", toFormat, 1)

	cmd := exec.Command("ffmpeg", "-i", os.TempDir()+"/"+fileName, os.TempDir()+"/"+toFileName)
	err := cmd.Start()
	if err != nil {
		return nil, "", err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, "", err
	}
	defer os.Remove(os.TempDir() + "/" + toFileName)

	converted, err := ioutil.ReadFile(os.TempDir() + "/" + toFileName)
	if err != nil {
		return nil, "", err
	}

	return converted, toFileName, nil
}

func DownloadMedia(writer io.Writer, mediaID, amrConvertTo string, setContentType, setFileName func(string)) error {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/get?access_token=%s&media_id=%s", GetOpenAccessToken(), mediaID)

	log.Println("dowload media: ", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get user media:", err)
		return err
	}

	if resp.StatusCode > 400 {
		return fmt.Errorf("微信返回: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	fileName := strings.ToLower(extractFilename(resp.Header.Get("Content-Disposition")))

	if contentType == "audio/amr" && amrConvertTo != "" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		converted, convertedFileName, err := convertAMR(body, fileName, amrConvertTo)
		if err != nil {
			return err
		}

		setContentType(strings.Replace(contentType, "amr", amrConvertTo, 1))
		setFileName(convertedFileName)

		if _, err := writer.Write(converted); err != nil {
			return err
		}

	} else {
		setContentType(contentType)
		setFileName(fileName)

		if _, err := io.Copy(writer, resp.Body); err != nil {
			return err
		}
	}

	return nil
}

func DownloadVoice(writer io.Writer, mediaID string, setContentType, setFileName func(string)) error {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/get/jssdk?access_token=%s&media_id=%s", GetOpenAccessToken(), mediaID)

	log.Println("dowload voice: ", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get user voice:", err)
		return err
	}

	if resp.StatusCode > 400 {
		return fmt.Errorf("微信返回: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	setContentType(resp.Header.Get("Content-Type"))
	setFileName(extractFilename(resp.Header.Get("Content-Disposition")))

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return err
	}
	return nil
}

func PlatformUploadMedia(accessToken, mediaType string, data []byte) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=%s", accessToken, mediaType)

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Println("error upload tmp media: ", err)
		return "", err
	}

	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error upload tmp media: ", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error upload tmp media: ", err)
		return "", err
	}

	var ret struct {
		ErrorRet
		MediaID string `json:"mediaid"`
	}
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to upload tmp media. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return "", errors.New(ret.ErrMsg)
	}

	return ret.MediaID, nil
}

func PlatformDownloadMedia(accessToken string, writer io.Writer, mediaID, amrConvertTo string, setContentType, setFileName func(string)) error {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/get?access_token=%s&media_id=%s", accessToken, mediaID)

	log.Println("dowload media: ", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("error get user media:", err)
		return err
	}

	if resp.StatusCode > 400 {
		return fmt.Errorf("微信返回: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	fileName := strings.ToLower(extractFilename(resp.Header.Get("Content-Disposition")))

	if contentType == "audio/amr" && amrConvertTo != "" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		converted, convertedFileName, err := convertAMR(body, fileName, amrConvertTo)
		if err != nil {
			return err
		}

		setContentType(strings.Replace(contentType, "amr", amrConvertTo, 1))
		setFileName(convertedFileName)

		if _, err := writer.Write(converted); err != nil {
			return err
		}

	} else {
		setContentType(contentType)
		setFileName(fileName)

		if _, err := io.Copy(writer, resp.Body); err != nil {
			return err
		}
	}

	return nil
}

func PlatformDownloadMaterial(accessToken, mediaID string) ([]byte, error) {
	url := fmt.Sprintf(" https://api.weixin.qq.com/cgi-bin/material/get_material?access_token=%s", accessToken)

	client := &http.Client{}

	param := map[string]interface{}{
		"media_id": mediaID,
	}

	data, _ := json.Marshal(param)

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Println("error download material: ", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error download materia: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error download materia: ", err)
		return nil, err
	}

	var ret struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.Unmarshal(body, &ret)

	if ret.ErrCode != 0 {
		log.Printf("failed to download materia. errCode[%d] errMsg[%s]", ret.ErrCode, ret.ErrMsg)
		return nil, errors.New(ret.ErrMsg)
	}

	return body, nil
}
