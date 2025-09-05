package odor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment/protocol"

	sLog "obsessiontech/environment/environment/receiver/log"
)

const PROTOCOL_THWATER = "thwater"

type THWater struct {
	protocol.BaseProtocol
}

func init() {
	protocol.Register(PROTOCOL_THWATER, func() protocol.IProtocol {
		return &THWater{}
	})
}

func (p *THWater) Run() {
	sLog.Log(p.MN, "[%s]协议未实现", p.UUID)
}

func (p *THWater) Redirect(redirection, datagram, dataType string, dataTime *time.Time, datas map[string]string) {
	parts := strings.Split(redirection, "#")
	host := parts[0]

	if dataTime == nil {
		sLog.Log(p.MN, "[%s]会话转发失败: 没有时间戳", p.UUID)
		return
	}

	params := make(map[string]interface{})
	if len(parts) > 1 {
		err := json.Unmarshal([]byte(parts[1]), &params)
		if err != nil {
			sLog.Log(p.MN, "[%s]会话转发失败: 解析设置出错 %s", p.UUID, err.Error())
			return
		}
	} else {
		sLog.Log(p.MN, "[%s]会话转发失败: 没有设置", p.UUID)
		return
	}

	equipType := params["equipType"]
	if equipType == nil {
		sLog.Log(p.MN, "[%s]会话转发失败: 没有equipType", p.UUID)
		return
	}
	token := params["token"]
	if token == nil {
		sLog.Log(p.MN, "[%s]会话转发失败: 没有token", p.UUID)
		return
	}

	body := make(map[string]interface{})
	mapping := params["mapping"]
	for k, v := range datas {
		if mapping == nil {
			body[k] = v
			continue
		}
		if mapper, ok := mapping.(map[string]interface{}); ok {
			if to, exists := mapper[k]; exists {
				body[to.(string)] = v
			}
		}
	}

	body["timestamp"] = util.FormatDateTime(*dataTime)

	reqBytes, _ := json.Marshal([]map[string]interface{}{body})

	URL := fmt.Sprintf("https://%s/api/dataExternals/upload?equipType=%s&equipId=%s", host, equipType, p.MN)

	sLog.Log(p.MN, "转发报文[%s]%s %s %s", host, URL, token, string(reqBytes))

	client := &http.Client{}

	req, err := http.NewRequest("POST", URL, bytes.NewReader(reqBytes))
	if err != nil {
		sLog.Log(p.MN, "[%s]会话转发失败: %s", p.UUID, err.Error())
		return
	}

	req.Header.Set("X-TH-TOKEN", token.(string))

	resp, err := client.Do(req)
	if err != nil {
		sLog.Log(p.MN, "[%s]会话转发失败: %s", p.UUID, err.Error())
		return
	}

	defer resp.Body.Close()
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		sLog.Log(p.MN, "[%s]会话转发失败: %s", p.UUID, err.Error())
		return
	}

	sLog.Log(p.MN, "[%s]会话转发回文: %s", p.UUID, string(resBody))
}
