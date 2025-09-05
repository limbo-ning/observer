package protocol

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"obsessiontech/environment/environment/entity"
	sLog "obsessiontech/environment/environment/receiver/log"
)

type IProtocol interface {
	SetSiteID(string)
	GetSiteID() string
	SetProtocol(string)
	GetProtocol() string
	SetUUID(string)
	GetUUID() string
	SetMN(string)
	SetCancel(func())
	GetCancel() func()
	SetCtx(context.Context)
	GetCtx() context.Context
	SetInputChan(chan string)
	SetOutputChan(chan string)
	Run()
	SetStation(*entity.Station)
	GetStation() *entity.Station
	Redirect(redirection, datagram, dataType string, dataTime *time.Time, data map[string]string)
}

type BaseProtocol struct {
	SiteID string

	Protocol   string
	UUID       string
	MN         string
	Cancel     func()
	Ctx        context.Context
	InputChan  chan string
	OutputChan chan string
	Station    *entity.Station

	redirectProtocolLock sync.RWMutex
	redirectProtocol     map[string]IProtocol
}

func (p *BaseProtocol) SetSiteID(siteID string)            { p.SiteID = siteID }
func (p *BaseProtocol) GetSiteID() string                  { return p.SiteID }
func (p *BaseProtocol) SetProtocol(protocol string)        { p.Protocol = protocol }
func (p *BaseProtocol) GetProtocol() string                { return p.Protocol }
func (p *BaseProtocol) SetUUID(uuid string)                { p.UUID = uuid }
func (p *BaseProtocol) GetUUID() string                    { return p.UUID }
func (p *BaseProtocol) SetMN(mn string)                    { p.MN = mn }
func (p *BaseProtocol) GetMN() string                      { return p.MN }
func (p *BaseProtocol) SetCancel(cancel func())            { p.Cancel = cancel }
func (p *BaseProtocol) GetCancel() func()                  { return p.Cancel }
func (p *BaseProtocol) SetCtx(ctx context.Context)         { p.Ctx = ctx }
func (p *BaseProtocol) GetCtx() context.Context            { return p.Ctx }
func (p *BaseProtocol) SetInputChan(c chan string)         { p.InputChan = c }
func (p *BaseProtocol) SetOutputChan(c chan string)        { p.OutputChan = c }
func (p *BaseProtocol) SetStation(station *entity.Station) { p.Station = station }
func (p *BaseProtocol) GetStation() *entity.Station        { return p.Station }

func (p *BaseProtocol) ProcessRedirection(datagram, dataType string, dataTime *time.Time, datas map[string]string) {

	if p.GetStation().Redirect == "" {
		return
	}

	for _, r := range strings.Split(p.GetStation().Redirect, ";") {
		parts := strings.Split(r, "#")
		log.Println("process redirect: ", p.MN, p.UUID, r, datagram, parts)

		p.redirectProtocolLock.RLock()
		var protoInstance IProtocol
		if p.redirectProtocol != nil {
			protoInstance = p.redirectProtocol[parts[0]]
		}
		p.redirectProtocolLock.RUnlock()

		var proto string
		if len(parts) > 1 {
			params := make(map[string]interface{})
			err := json.Unmarshal([]byte(parts[1]), &params)
			if err != nil {
				log.Printf("[%s]会话转发失败: 解析设置出错 %s", p.MN, err.Error())
				sLog.Log(p.MN, "[%s]会话转发失败: 解析设置出错 %s", p.UUID, err.Error())
				continue
			}
			targetType := params["dataType"]
			if targetType != "" && targetType != dataType {
				continue
			}
			proto = params["protocol"].(string)
		} else {
			proto = p.GetProtocol()
		}

		if protoInstance == nil {
			p.redirectProtocolLock.Lock()
			if p.redirectProtocol == nil {
				p.redirectProtocol = make(map[string]IProtocol)
			}
			protoInstance = p.redirectProtocol[parts[0]]
			if protoInstance == nil {
				protoInstance = GetProtocol(proto)
				if protoInstance == nil {
					log.Printf("[%s]会话转发失败: 转发协议[%s]不存在", p.MN, proto)
					sLog.Log(p.MN, "[%s]会话转发失败: 转发协议[%s]不存在", p.UUID, proto)
					p.redirectProtocolLock.Unlock()
					continue
				}
				protoInstance.SetMN(p.MN)
				protoInstance.SetUUID(p.UUID)
				protoInstance.SetCtx(p.Ctx)
				protoInstance.SetCancel(p.Cancel)
				protoInstance.SetStation(p.Station)
			}
			p.redirectProtocol[parts[0]] = protoInstance
			p.redirectProtocolLock.Unlock()
		}

		log.Println("process redirect instance: ", protoInstance.GetProtocol(), protoInstance.GetUUID())
		protoInstance.Redirect(r, datagram, dataType, dataTime, datas)

		log.Println("process redirect done: ", p.MN, p.UUID, r, datagram)
	}
}
