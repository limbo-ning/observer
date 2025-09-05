package connection

import (
	"context"
	"net"
	"sync"

	"obsessiontech/environment/environment/protocol"
	sLog "obsessiontech/environment/environment/receiver/log"
)

type Connection struct {
	Conn   *net.TCPConn
	Ctx    context.Context
	Cancel func()
}

var connections = make(map[string]protocol.IProtocol)
var connLock sync.RWMutex

func GetRunningProtocol(mn string) (protocol.IProtocol, bool) {
	connLock.RLock()
	defer connLock.RUnlock()

	protocol, exists := connections[mn]

	return protocol, exists
}

func AddConnection(mn, proto string, p protocol.IProtocol) {
	connLock.Lock()
	sLog.Log(mn, "设备接入[%s]", p.GetUUID())
	if old := connections[mn]; old != nil && old.GetUUID() != p.GetUUID() {
		sLog.Log(mn, "设备关闭旧连接[%s]", old.GetUUID())
		old.GetCancel()()
	}
	connections[mn] = p

	connLock.Unlock()
}

func RemoveConnection(mn, proto string, p protocol.IProtocol) {
	connLock.Lock()
	if current := connections[mn]; current != nil && current.GetUUID() == p.GetUUID() {
		sLog.Log(mn, "设备断开[%s]", p.GetUUID())
		current.GetCancel()()
		delete(connections, mn)
	}
	connLock.Unlock()
}
