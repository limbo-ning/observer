package main

import (
	"fmt"
	"log"
	"os"

	"obsessiontech/common/ipc"
)

func main() {

	for i := 0; i < 10; i++ {
		conn, err := ipc.StartClient("unix", "/tmp/testHost.unix")
		if err != nil {
			panic(err)
		}
		client(i, conn)
	}
}

func client(i int, conn *ipc.Connection) {

	if err := ipc.Write(conn.Conn, []byte(fmt.Sprintf("[%d]first message", i))); err != nil {
		log.Println("error send: ", err)
		conn.Cancel()
		if err := conn.Conn.Close(); err != nil {
			log.Println("error close: ", err)
		}
		os.Exit(1)
	}
	log.Printf("%d: sent first", i)

	if data, _, err := ipc.Receive(conn.Conn); err != nil {
		log.Println("error receive: ", err)
		conn.Cancel()
		if err := conn.Conn.Close(); err != nil {
			log.Println("error close: ", err)
		}
		os.Exit(1)
	} else {
		for _, d := range data {
			log.Printf("%d: receive %s", i, string(d))
		}
	}

	if err := ipc.Write(conn.Conn, []byte(fmt.Sprintf("[%d]second message", i))); err != nil {
		log.Println("error: ", err)
		conn.Cancel()
		if err := conn.Conn.Close(); err != nil {
			log.Println("error close: ", err)
		}
		os.Exit(1)
	}
	log.Printf("%d: sent second", i)

	if data, _, err := ipc.Receive(conn.Conn); err != nil {
		log.Println("error: ", err)
		if err := conn.Conn.Close(); err != nil {
			log.Println("error close: ", err)
		}
		os.Exit(1)
	} else {
		for _, d := range data {
			log.Printf("%d: receive %s", i, string(d))
		}
	}

	if _, _, err := ipc.Receive(conn.Conn); err != nil {
		log.Println("error: ", err)
		if err := conn.Conn.Close(); err != nil {
			log.Println("error close: ", err)
		}
		os.Exit(1)
	}

}
