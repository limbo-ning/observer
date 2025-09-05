package ipc

import (
	"context"
	"net"
)

type Connection struct {
	Conn   net.Conn
	Ctx    context.Context
	Cancel func()
}
