package ipchandler

import (
	"log"
	"sync"

	"obsessiontech/common/ipc"
	"obsessiontech/environment/environment/ipcmessage"
)

var connections = make(map[*ipc.Connection]*listenerConnection)
var lock sync.RWMutex

type listenerConnection struct {
	IsPersistentListener bool
	CloseFunc            func()
}

func StartIPC(connType, addr string) error {
	connChan, err := ipc.StartHost(connType, addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case conn, ok := <-connChan:
				if !ok {
					log.Printf("%s closing", addr)
					return
				}

				lock.Lock()
				connections[conn] = &listenerConnection{
					IsPersistentListener: false,
					CloseFunc: func() {
						conn.Cancel()

						lock.Lock()
						defer lock.Unlock()
						if _, exist := connections[conn]; exist {
							delete(connections, conn)
						}
					},
				}
				lock.Unlock()

				go listen(conn)
			}
		}
	}()

	return nil
}

func listen(conn *ipc.Connection) {

	for {
		select {
		case <-conn.Ctx.Done():
			lock.RLock()
			listenerConn := connections[conn]
			lock.RUnlock()
			if listenerConn != nil {
				listenerConn.CloseFunc()
			}
			return
		default:
		}

		datagrams, _, err := ipc.Receive(conn.Conn)
		if err != nil {
			lock.RLock()
			listenerConn := connections[conn]
			lock.RUnlock()
			if listenerConn != nil {
				listenerConn.CloseFunc()
			}
			return
		}

		for _, d := range datagrams {
			message, err := ipcmessage.ParseMessage(d)
			if err != nil {
				log.Println("error ipc message: ", err)
				continue
			}

			log.Println("receive ipc message:", message.GetIPCMessageType())

			var res ipcmessage.IMessage

			switch message.(type) {
			case (*ipcmessage.ListenReq):
				log.Println("receiver listen req")
				lock.RLock()
				listenerConn := connections[conn]
				lock.RUnlock()
				if listenerConn != nil {
					listenerConn.IsPersistentListener = true
				}
				log.Println("listen req processed")
				var ack ipcmessage.Ack
				ack = 1
				res = &ack
			case (*ipcmessage.StationStatusReq):
				req := message.(*ipcmessage.StationStatusReq)
				res = ReportRequestedStation([]int(*req))
			case (*ipcmessage.ModuleReloadReq):
				res = ReloadModule()
			case (*ipcmessage.StationReloadReq):
				stationID := message.(*ipcmessage.StationReloadReq)
				res = ReloadStation(int(*stationID))
			case (*ipcmessage.MonitorReloadReq):
				res = ReloadMonitor()
			case (*ipcmessage.MonitorCodeReloadReq):
				res = ReloadMonitorCode()
			case (*ipcmessage.FlagLimitReloadReq):
				res = ReloadFlagLimit()
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
					lock.RLock()
					listenerConn := connections[conn]
					lock.RUnlock()
					if listenerConn != nil {
						listenerConn.CloseFunc()
					}
					return
				}
			}
		}

	}
}

func broadcast(message ipcmessage.IMessage) {

	data, err := ipcmessage.WrapMessage(message)
	if err != nil {
		log.Println("error wrap ipc message: ", message.GetIPCMessageType(), err)
		return
	}

	connList := make([]*ipc.Connection, 0)

	lock.RLock()
	defer lock.RUnlock()
	for conn, listenerConn := range connections {
		if listenerConn.IsPersistentListener {
			connList = append(connList, conn)
		}
	}

	for _, conn := range connList {
		if err := ipc.Write(conn.Conn, data); err != nil {
			log.Println("error broadcasting: ", err)
			listenerConn := connections[conn]
			if listenerConn != nil {
				listenerConn.CloseFunc()
			}
		}
	}

	return
}
