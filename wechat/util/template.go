package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func GetTemplateCategory(accessToken string) (interface{}, error) {
	var ret ErrorRet

	resp, err := http.Get("https://api.weixin.qq.com/wxaapi/newtmpl/getcategory?access_token=" + accessToken)
	if err != nil {
		log.Println("error get miniapp template category:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp template category:", err)
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

func GetPublicTemplate(accessToken string, categoryIDs []string, start, limit int) (interface{}, error) {
	var ret ErrorRet

	resp, err := http.Get(fmt.Sprintf("https://api.weixin.qq.com/wxaapi/newtmpl/getpubtemplatetitles?ids=%s&start=%d&limit=%daccess_token=%s", strings.Join(categoryIDs, ","), start, limit, accessToken))
	if err != nil {
		log.Println("error get miniapp public templates:", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error get miniapp public templates:", err)
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
