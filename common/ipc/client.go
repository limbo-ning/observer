package ipc

import (
	"context"
	"log"
	"net"

	myContext "obsessiontech/common/context"
)

func StartClient(connType, remote string) (*Connection, error) {
	conn, err := net.Dial(connType, remote)
	if err != nil {
		log.Println("error dial unix: ", err)
		return nil, err
	}

	out := new(Connection)
	out.Conn = conn

	childCtx, childCancel := myContext.GetContext()
	out.Ctx, out.Cancel = context.WithCancel(childCtx)

	go func() {
		<-out.Ctx.Done()
		log.Println("client connection close")
		conn.Close()
		childCancel()
	}()

	return out, nil
}
