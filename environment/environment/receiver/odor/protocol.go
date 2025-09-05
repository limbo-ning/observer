package odor

import (
	"log"
	"net"
	"strings"
	"time"

	"obsessiontech/environment/environment/protocol"
	"obsessiontech/environment/environment/receiver/odor/instruction"

	sLog "obsessiontech/environment/environment/receiver/log"
)

const PROTOCOL_ODOR = "keqin_odor"

type Odor struct {
	protocol.BaseProtocol
}

func init() {
	protocol.Register(PROTOCOL_ODOR, func() protocol.IProtocol {
		return &Odor{}
	})
}

func (p *Odor) Run() {
	sLog.Log(p.MN, "[%s]通讯启动", p.UUID)

	for {
		select {
		case datagram := <-p.InputChan:
			i, err := instruction.Parse(datagram)
			if err != nil {
				sLog.Log(p.MN, "数据错误 [%s] [%s]", p.UUID, err.Error())
				log.Printf("数据错误 [%s] [%s] [%s]", p.MN, p.UUID, err.Error())
				continue
			}

			if err := p.uploadData(i); err != nil {
				sLog.Log(p.MN, "上传错误 [%s]: %s", p.UUID, err.Error())
				log.Printf("上传错误 [%s] [%s]: %s", p.MN, p.UUID, err.Error())
				continue
			}

			p.ProcessRedirection(datagram, i.DataType, &i.DateTime, i.Data)

		case <-p.Ctx.Done():
			sLog.Log(p.MN, "[%s]通讯停止", p.UUID)
			return
		}
	}
}

func (p *Odor) Redirect(redirection, datagram, dataType string, dataTime *time.Time, datas map[string]string) {
	param := strings.Split(redirection, "#")

	addr, err := net.ResolveTCPAddr("tcp", param[0])
	if err != nil {
		sLog.Log(p.MN, "转发失败[%s]: %s", p.MN, addr.String(), err.Error())
		return
	}
	redirectConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		sLog.Log(p.MN, "转发失败[%s]: %s", p.MN, addr.String(), err.Error())
		return
	}

	kill := func() {
		if redirectConn != nil {
			redirectConn.Close()
		}
	}

	sLog.Log(p.MN, "转发报文[%s] %s", param[0], datagram)
	_, err = redirectConn.Write([]byte(datagram))
	if err != nil {
		sLog.Log(p.MN, "转发失败[%s]: %s", param[0], err.Error())
		kill()
		return
	}

	go func() {
		defer kill()
		err := redirectConn.SetReadDeadline(time.Now().Add(time.Second * 10))
		if err != nil {
			sLog.Log(p.MN, "转发回文超时设置失败[%s]: %s", param[0], err.Error())
			return
		}

		reply := make([]byte, 0)

		for {
			buff := make([]byte, 1024)
			readLength, err := redirectConn.Read(buff)
			if err != nil {
				sLog.Log(p.MN, "转发回文读取失败[%s]: %s", param[0], err.Error())
				return
			}
			reply = append(reply, buff[:readLength]...)

			if readLength < 1024 {
				break
			}
		}
		sLog.Log(p.MN, "转发回文[%s]: %s", param[0], string(reply))
	}()
}
