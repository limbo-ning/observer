package hjt212

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"obsessiontech/environment/environment/protocol"
	sLog "obsessiontech/environment/environment/receiver/log"
)

type HJT212 struct {
	protocol.BaseProtocol
	version string

	lineSwitcher LineSwitch
}

type LineSwitch struct {
	Lock  sync.RWMutex
	Lines map[string]*Line
}

type Line struct {
	InputCh chan *Instruction
}

var DEVICE_NOT_CONNECTED = errors.New("当前设备连接中断")

func init() {
	protocol.Register("HJT212-2017", func() protocol.IProtocol {
		return &HJT212{
			version: "2017",
			lineSwitcher: LineSwitch{
				Lines: make(map[string]*Line),
			},
		}
	})

	protocol.Register("HJT212-2005", func() protocol.IProtocol {
		return &HJT212{
			version: "2005",
			lineSwitcher: LineSwitch{
				Lines: make(map[string]*Line),
			},
		}
	})
}

func (p *HJT212) Run() {
	sLog.Log(p.MN, "[%s]通讯启动", p.UUID)

	var previous string

	for {
		select {
		case incoming := <-p.InputChan:
			datagrams := strings.Split(incoming, "\r\n")
			for i, datagram := range datagrams {

				if i == 0 {
					datagram = previous + datagram
					previous = ""
				}

				if datagram == "" {
					continue
				}
				data, err := ValidateDatagram(datagram)
				if err != nil {
					if err == e_incomplete {
						log.Println("datagram incomplete: ", datagram)
						previous = datagram
						continue
					}

					sLog.Log(p.MN, "[%s]校验失败:%s", p.UUID, err.Error())
					p.Cancel()
					continue
				}
				sLog.Log(p.MN, "[%s]解析报文:%s", p.UUID, datagram)
				instruction, err := DecomposeInstruction(data)
				if err != nil {
					sLog.Log(p.MN, "[%s]解析失败:%s", p.UUID, err.Error())
					p.Cancel()
					continue
				}
				instruction.version = p.version

				if err := p.demuxConversation(data, instruction); err != nil {
					log.Printf("[%s]会话路由失败: %s", p.MN, err.Error())
					sLog.Log(p.MN, "[%s]会话路由失败: %s", p.UUID, err.Error())
					continue
				}
			}
		case <-p.Ctx.Done():
			sLog.Log(p.MN, "通讯停止[%s]", p.UUID)
			return
		}
	}
}

func (p *HJT212) demuxConversation(datagram string, instruction *Instruction) error {
	var count int
perTry:
	count++

	log.Printf("路由会话: MN[%s] QN[%s] (第%d次)", instruction.MN, instruction.QN, count)

	if instruction.QN == "" {
		instruction.QN = GenerateQN()
	}

	p.lineSwitcher.Lock.RLock()
	line, exist := p.lineSwitcher.Lines[instruction.QN]
	p.lineSwitcher.Lock.RUnlock()

	if !exist {
		exe, err := p.InvokeExecutor(instruction)
		if err != nil {
			return err
		}
		newLine, input, process, output, close, err := p.initializeConversation(instruction.QN)
		if err != nil {
			return err
		}
		line = newLine
		go exe.Execute(p.SiteID, instruction.QN, input, process, output, close)
	}

	select {
	case line.InputCh <- instruction:
	case <-time.After(5 * time.Second):
		if count < 5 {
			goto perTry
		} else {
			log.Printf("[%s]会话[%s]繁忙 丢弃", instruction.MN, instruction.QN)
			return nil
		}
	}

	return nil
}

func (p *HJT212) initializeConversation(QN string) (*Line, func() (*Instruction, error), func(*Instruction), func(*Instruction) error, func(err error), error) {
	log.Printf("初始化会话 MN[%s] UUID[%s] QN[%s]", p.MN, p.UUID, QN)

	inputCh := make(chan *Instruction)

	p.lineSwitcher.Lock.Lock()
	line := &Line{
		InputCh: inputCh,
	}
	p.lineSwitcher.Lines[QN] = line
	p.lineSwitcher.Lock.Unlock()

	ctx, cancel := context.WithCancel(p.Ctx)

	input := func() (*Instruction, error) {
		select {
		case instruction := <-inputCh:
			return instruction, nil
		case <-ctx.Done():
			return nil, DEVICE_NOT_CONNECTED
		}
	}

	process := func(i *Instruction) {
		p.ProcessRedirection(PackDatagram(ComposeInstruction(i)), i.dataType, i.dataTime, i.data)
	}

	output := func(o *Instruction) error {
		select {
		case <-ctx.Done():
			return DEVICE_NOT_CONNECTED
		case p.OutputChan <- PackDatagram(ComposeInstruction(o)):
			return nil
		}
	}

	close := func(err error) {
		p.lineSwitcher.Lock.Lock()
		if err != nil {
			log.Printf("关闭会话 [%s] [%s] %s", p.MN, QN, err.Error())
			sLog.Log(p.MN, "关闭会话 [%s] [%s] %s", p.UUID, QN, err.Error())
		} else {
			sLog.Log(p.MN, "关闭会话 [%s] [%s]", p.UUID, QN)
		}
		current := p.lineSwitcher.Lines[QN]
		if current == line {
			delete(p.lineSwitcher.Lines, QN)
		}
		p.lineSwitcher.Lock.Unlock()

		cancel()
	}

	time.AfterFunc(10*time.Minute, func() {
		p.lineSwitcher.Lock.RLock()
		current := p.lineSwitcher.Lines[QN]
		p.lineSwitcher.Lock.RUnlock()

		if line == current {
			close(errors.New("执行超时"))
		}
	})

	return line, input, process, output, close, nil
}

func (p *HJT212) Redirect(redirection, datagram, dataType string, dataTime *time.Time, datas map[string]string) {
	param := strings.Split(redirection, "#")

	redirectInstruction := new(Instruction)
	redirectInstruction.data = datas

	originDatagram, err := ValidateDatagram(datagram)
	if err != nil {
		sLog.Log(p.MN, "转发失败[%s]: %s 无法按协议解析: %s", p.MN, datagram, err.Error())
		return
	}

	originInstruction, err := DecomposeInstruction(originDatagram)
	if err != nil {
		sLog.Log(p.MN, "转发失败[%s]: %s 无法按协议解析: %s", p.MN, datagram, err.Error())
		return
	}

	redirectInstruction.CN = originInstruction.CN
	redirectInstruction.Flag = originInstruction.Flag
	redirectInstruction.MN = originInstruction.MN
	redirectInstruction.QN = originInstruction.QN
	redirectInstruction.ST = originInstruction.ST
	redirectInstruction.PNO = originInstruction.PNO
	redirectInstruction.PNUM = originInstruction.PNUM
	redirectInstruction.PW = originInstruction.PW

	redirectInstruction.CP = composeCPGroup(dataType, dataTime, datas)

	if set, exists := p.GetStation().Ext["MN"]; exists {
		if mn, ok := set.(string); ok {
			redirectInstruction.MN = mn
		}
	}
	if set, exists := p.GetStation().Ext["ST"]; exists {
		if st, ok := set.(string); ok {
			redirectInstruction.ST = st
		}
	}
	if set, exists := p.GetStation().Ext["PW"]; exists {
		if pw, ok := set.(string); ok {
			redirectInstruction.ST = pw
		}
	}

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

	toSend := PackDatagram(ComposeInstruction(redirectInstruction))

	sLog.Log(p.MN, "转发报文[%s] %s", param[0], toSend)

	_, err = redirectConn.Write([]byte(toSend))
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
