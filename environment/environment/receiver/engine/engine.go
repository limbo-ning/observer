package engine

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"regexp"
	"time"

	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/protocol"
	"obsessiontech/environment/environment/receiver/connection"
	"obsessiontech/environment/environment/receiver/ipchandler"

	sLog "obsessiontech/environment/environment/receiver/log"
)

var mnRegexp *regexp.Regexp

func EstablishConnection(conn *connection.Connection) {

	datagram, err := receive(conn.Conn)
	if err != nil || len(datagram) == 0 {
		conn.Cancel()
		return
	}

	MN, err := extractMN(datagram)
	if err != nil {
		log.Println("error 报文提取MN失败:", datagram, err)
		conn.Cancel()
		return
	}

	station := entity.GetCacheStationByMN(Config.SiteID, MN)
	if station == nil {
		log.Printf("error MN[%s] 不存在", MN)
		conn.Cancel()
		return
	}

	if station.Status == entity.INACTIVE {
		log.Printf("error MN[%s] 已关停", MN)
		conn.Cancel()
		return
	}

	var protocolInstance protocol.IProtocol

	module, err := environment.GetModule(Config.SiteID)
	if err != nil {
		log.Printf("error MN[%s]  获取环保模块设定失败: %s", MN, err.Error())
		sLog.Log(MN, "获取环保模块设定失败: %s", err.Error())
		conn.Cancel()
		return
	}
	for _, m := range module.Protocols {
		if m.Protocol == station.Protocol {
			protocolInstance = protocol.GetProtocol(station.Protocol)
			break
		}
	}

	if protocolInstance == nil {
		log.Printf("error [%s] 协议[%s]不支持", MN, station.Protocol)
		conn.Cancel()
		return
	}

	uuid := fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Int())

	readCh := make(chan string)
	outputCh := make(chan string)

	timerResetCh := make(chan byte)

	protocolInstance.SetSiteID(Config.SiteID)
	protocolInstance.SetUUID(uuid)
	protocolInstance.SetMN(MN)
	protocolInstance.SetInputChan(readCh)
	protocolInstance.SetOutputChan(outputCh)
	protocolInstance.SetCtx(conn.Ctx)
	protocolInstance.SetCancel(conn.Cancel)
	protocolInstance.SetStation(station)

	go func() {
		t := time.NewTimer(time.Minute * 15)
		for {
			select {
			case <-timerResetCh:
				if !t.Stop() {
					select {
					case <-t.C:
					default:
					}
				}
				t.Reset(time.Minute * 15)
			case <-t.C:
				sLog.Log(MN, "连接闲置超时 [%s]", uuid)
				conn.Cancel()
			case <-conn.Ctx.Done():
				sLog.Log(MN, "停止连接")
				connection.RemoveConnection(MN, station.Protocol, protocolInstance)
				ipchandler.Offline(MN, station.Protocol)
				return
			}
		}
	}()

	connection.AddConnection(MN, station.Protocol, protocolInstance)
	ipchandler.Online(MN, station.Protocol)

	go protocolInstance.Run()
	go ipchandler.ReportStation(MN, true)

	go func() {
		for {
			count := 0
		perTry:
			count++
			log.Printf("%s 处理命令[%s](第%d次): %s", MN, uuid, count, datagram)
			select {
			case readCh <- datagram:
				sLog.Log(MN, "接收报文:%s", datagram)
				ipchandler.Online(MN, station.Protocol)
				timerResetCh <- 1
			case <-time.After(5 * time.Second):
				if count < 5 {
					goto perTry
				} else {
					sLog.Log(MN, "系统繁忙[%s] 重启", uuid)
					conn.Cancel()
					return
				}
			case <-conn.Ctx.Done():
				sLog.Log(MN, "停止连接接收端")
				return
			}
			datagram, err = receive(conn.Conn)
			if err != nil {
				sLog.Log(MN, "连接读取数据错误[%s] %s", uuid, err.Error())
				conn.Cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-conn.Ctx.Done():
			sLog.Log(MN, "停止连接发送端")
			return
		case datagram := <-outputCh:
			timerResetCh <- 1
			sLog.Log(MN, "发送报文:%s", datagram)
			err := write(conn.Conn, datagram)
			if err != nil {
				sLog.Log(MN, "发送失败[%s] %s", uuid, err.Error())
				conn.Cancel()
				return
			}
		}
	}
}

func extractMN(datagram string) (string, error) {

	matches := mnRegexp.FindAllStringSubmatch(datagram, 1)
	log.Println("extract mn from datagrom: ", datagram, matches)
	if len(matches) == 0 {
		return "", errors.New("报文没有包含有效的MN号")
	}

	for i, m := range matches[0] {
		if i > 0 && m != "" {
			return m, nil
		}
	}

	if len(matches[0]) > 1 {
		return matches[0][1], nil
	}

	return "", errors.New("报文没有包含有效的MN号")
}

const readBuffSize = 1024

func receive(conn *net.TCPConn) (string, error) {
	var datagram []byte

	buf := make([]byte, readBuffSize)
	for {
		length, err := conn.Read(buf)
		if err == io.EOF {
			log.Println("read eof", conn.RemoteAddr().String())
			break
		} else if err != nil {
			log.Println("read error:", err)
			return "", err
		} else if length == 0 {
			log.Println("read length 0", conn.RemoteAddr().String())
		} else {
			if length < len(buf) {
				buf[length] = 0
				datagram = append(datagram, buf[0:length]...)
				break
			} else {
				datagram = append(datagram, buf...)
				buf = make([]byte, len(buf)+readBuffSize)
				continue
			}
		}
	}

	if len(datagram) == 0 {
		log.Println("read error empty data:", io.ErrUnexpectedEOF, conn.RemoteAddr().String())
		return "", io.ErrUnexpectedEOF
	}

	str := string(datagram)
	log.Println("接收报文: ", str)

	return str, nil
}

func write(conn *net.TCPConn, datagram string) error {
	log.Println("发送报文: ", datagram)
	send, err := conn.Write([]byte(datagram))
	log.Println("sent", send)

	if err != nil {
		log.Println("write error:", err)
		return err
	}

	return nil
}
