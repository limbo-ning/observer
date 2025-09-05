package main

import (
	"log"

	"obsessiontech/common/ipc"
)

func main() {
	connChan, err := ipc.StartHost("unix", "/tmp/testHost.unix")
	if err != nil {
		log.Println("error: ", err)
		return
	}

	log.Println("started")

	for {
		select {
		case conn, ok := <-connChan:
			if !ok {
				log.Println("host closed")
				return
			}

			go listen(conn)
		}
	}
}

func listen(conn *ipc.Connection) {

	for {
		data, client, err := ipc.Receive(conn.Conn)
		if err != nil {
			log.Println("error: ", err)
			if client != nil {
				ipc.Write(conn.Conn, []byte("bad data"))
			}
			conn.Cancel()
			return
		}

		log.Println("receive: ", data)

		for _, d := range data {
			if err := ipc.Write(conn.Conn, append([]byte("echo:"), d...)); err != nil {
				log.Println("error: ", err)
				conn.Cancel()
				return
			}
		}
	}
}
