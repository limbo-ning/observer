package ipc

import (
	"context"
	"log"
	"net"
	"os"

	myContext "obsessiontech/common/context"
)

const (
	IPC_TCP  = "tcp"
	IPC_UNIX = "unix"
)

func StartHost(connType, local string) (<-chan *Connection, error) {

	log.Println("starting host: ", connType, local)

	if connType == IPC_UNIX {
		if err := os.Remove(local); err != nil {
			if !os.IsNotExist(err) {
				log.Println("fail to clean unix local file: ", err)
			}
		}
	}

	listener, err := net.Listen(connType, local)
	if err != nil {
		log.Println("error listen: ", connType, local, err)
		return nil, err
	}

	ctx, cancel := myContext.GetContext()
	closeListener := func() {
		listener.Close()
		log.Println("ipc socket closed: ", connType, local)

		cancel()
	}

	listenerChan := make(chan net.Conn)
	connChan := make(chan *Connection)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("ipc host listener close: ", connType, local)
				return
			default:
			}
			conn, err := listener.Accept()
			if err != nil {
				log.Println("error accept: ", err)
				continue
			}

			listenerChan <- conn
		}
	}()

	go func() {
		for {
			select {
			case conn := <-listenerChan:
				out := new(Connection)
				out.Conn = conn

				childCtx, childCancel := myContext.GetContext()
				out.Ctx, out.Cancel = context.WithCancel(childCtx)

				go func() {
					<-out.Ctx.Done()
					log.Println("connection close: ", connType, local, conn.RemoteAddr())
					conn.Close()
					childCancel()
				}()

				connChan <- out
			case <-ctx.Done():
				log.Println("host ctx done: ", connType, local)
				closeListener()
				close(connChan)
				return
			}
		}
	}()

	return connChan, nil
}
