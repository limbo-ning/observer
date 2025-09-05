package ipcclient

import (
	"log"
	"sync"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/ipc"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/data/recent"
	"obsessiontech/environment/environment/ipcmessage"
)

var Config struct {
	EnvironmentReceiverAddrs               []*ReceiverAddr
	EnvironmentReceiverReconnectTimeOutSec time.Duration
	EnvironmentReceiverRequestTimeOutSec   time.Duration
}

type ReceiverAddr struct {
	SiteID    string
	Addr      string
	ConnType  string
	LogFolder string
}

func init() {
	config.GetConfig("config.yaml", &Config)

	if Config.EnvironmentReceiverRequestTimeOutSec == 0 {
		Config.EnvironmentReceiverRequestTimeOutSec = 3
	}

	for _, addr := range Config.EnvironmentReceiverAddrs {

		switch addr.ConnType {
		case ipc.IPC_TCP:
		case ipc.IPC_UNIX:
		default:
			addr.ConnType = ipc.IPC_UNIX
		}

		if connections == nil {
			connections = make(map[string][]*ipc.Connection)
		}
		if _, exists := connections[addr.SiteID]; !exists {
			connections[addr.SiteID] = make([]*ipc.Connection, 0)
		}
		StartPersistentClient(addr.SiteID, addr.ConnType, addr.Addr)
	}
}

var connections map[string][]*ipc.Connection
var lock sync.RWMutex

func StartPersistentClient(siteID, connType, addr string) error {

	log.Println("persistent environment receiver client connecting: ", siteID, addr)

	conn, err := ipc.StartClient(connType, addr)
	if err != nil {
		log.Println("error start client: ", siteID, addr, err)

		if Config.EnvironmentReceiverReconnectTimeOutSec > 0 {
			time.AfterFunc(time.Second*Config.EnvironmentReceiverReconnectTimeOutSec, func() {
				StartPersistentClient(siteID, connType, addr)
			})
		}

		return err
	}

	listenReq := new(ipcmessage.ListenReq)
	reqData, err := ipcmessage.WrapMessage(listenReq)
	if err != nil {
		log.Println("error wrap ipc listen req: ", listenReq.GetIPCMessageType(), err)
		conn.Cancel()
		return err
	}

	if err := ipc.Write(conn.Conn, reqData); err != nil {
		log.Println("error write ipc listen req: ", listenReq.GetIPCMessageType(), err)
		conn.Cancel()
		return err
	}

	log.Println("persistent environment receiver client established: ", siteID, addr, conn.Conn.RemoteAddr().Network(), conn.Conn.RemoteAddr().String())

	lock.Lock()
	defer lock.Unlock()

	connections[siteID] = append(connections[siteID], conn)

	closeFunc := func() {
		conn.Cancel()

		lock.Lock()
		defer lock.Unlock()

		exists := false

		if list, exists := connections[siteID]; exists {
			newList := make([]*ipc.Connection, 0)
			for _, c := range list {
				if c == conn {
					continue
				}
				if c.Conn.RemoteAddr().String() == addr {
					exists = true
				}
				newList = append(newList, c)
			}

			connections[siteID] = newList
		}

		if !exists {
			if Config.EnvironmentReceiverReconnectTimeOutSec > 0 {
				time.AfterFunc(time.Second*Config.EnvironmentReceiverReconnectTimeOutSec, func() {
					StartPersistentClient(siteID, connType, addr)
				})
			}
		}

	}

	go func() {
		for {
			select {
			case <-conn.Ctx.Done():
				closeFunc()
				return
			default:
			}

			datagrams, _, err := ipc.Receive(conn.Conn)
			if err != nil {
				log.Println("error receive from environment receiver persistent client: ", err)
				closeFunc()
				return
			}

			for _, d := range datagrams {
				message, err := ipcmessage.ParseMessage(d)
				if err != nil {
					log.Println("error ipc message: ", err)
					continue
				}

				var res ipcmessage.IMessage

				switch msg := message.(type) {
				case (*ipcmessage.StationStatusRes):
					res := map[int]bool(*msg)
					for stationID, online := range res {
						go BroadcastStation(siteID, stationID, online)
						go stationHistoryInput(siteID, stationID, online)
						go PushStationStatus(siteID, stationID, online)
					}
				case (*ipcmessage.StationOffline):
					go BroadcastStationOffline(siteID, int(*msg))
					go PushOffline(siteID, int(*msg))
				case (*ipcmessage.ControlRes):
				case (*ipcmessage.RealTime):
					rtd := data.RealTimeData(*msg)
					go data.TriggerRotation(siteID, false)
					if isUpdated, _, _, _ := recent.UpdateRecentData(siteID, &rtd); isUpdated {
						go BroadcastData(siteID, &rtd)
						go PushData(siteID, &rtd)
					}
				case (*ipcmessage.Minutely):
					minutely := data.MinutelyData(*msg)
					go data.TriggerRotation(siteID, false)
					if isUpdated, _, _, _ := recent.UpdateRecentData(siteID, &minutely); isUpdated {
						go BroadcastData(siteID, &minutely)
						go PushData(siteID, &minutely)
					}
				case (*ipcmessage.Hourly):
					hourly := data.HourlyData(*msg)
					go data.TriggerRotation(siteID, false)
					if isUpdated, _, _, _ := recent.UpdateRecentData(siteID, &hourly); isUpdated {
						go BroadcastData(siteID, &hourly)
						go PushData(siteID, &hourly)
					}
				case (*ipcmessage.Daily):
					daily := data.DailyData(*msg)
					go data.TriggerRotation(siteID, false)
					if isUpdated, _, _, _ := recent.UpdateRecentData(siteID, &daily); isUpdated {
						go BroadcastData(siteID, &daily)
						go PushData(siteID, &daily)
					}
				default:
					continue
				}

				if res != nil {
					data, err := ipcmessage.WrapMessage(res)
					if err != nil {
						log.Println("error wrap ipc res: ", res.GetIPCMessageType(), err)
						continue
					}
					if err := ipc.Write(conn.Conn, data); err != nil {
						closeFunc()
						return
					}
				}
			}

		}
	}()

	return nil
}

func StartRequestClient(siteID, connType, addr string, output <-chan ipcmessage.IMessage) (<-chan ipcmessage.IMessage, error) {

	log.Println("environment receiver request client connecting: ", siteID, addr)

	conn, err := ipc.StartClient(connType, addr)
	if err != nil {
		log.Println("error start request client: ", siteID, addr, err)
		select {
		case <-output:
		default:
		}
		return nil, err
	}

	input := make(chan ipcmessage.IMessage)

	go func() {
		for {
			select {
			case <-conn.Ctx.Done():
				log.Println("request client ctx done")
				conn.Cancel()
				return
			case toSend, ok := <-output:
				if !ok {
					log.Println("request client output closed")
					conn.Cancel()
					return
				}
				data, err := ipcmessage.WrapMessage(toSend)
				if err != nil {
					log.Println("error wrap ipc data: ", toSend.GetIPCMessageType(), err)
					continue
				}
				if err := ipc.Write(conn.Conn, data); err != nil {
					conn.Cancel()
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-conn.Ctx.Done():
				log.Println("request client ctx done")
				conn.Cancel()
				close(input)
				return
			default:
			}

			datagrams, _, err := ipc.Receive(conn.Conn)
			if err != nil {
				log.Println("request client receive err: ", err)
				return
			}

			for _, d := range datagrams {
				message, err := ipcmessage.ParseMessage(d)
				if err != nil {
					log.Println("error ipc message: ", err)
					continue
				}

				select {
				case input <- message:
				case <-time.After(time.Second * 10):
					log.Println("request client input channel receive timeout")
				}
			}

		}
	}()

	log.Println("environment receiver request client established: ", siteID, addr)

	return input, nil
}
