package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"

	// "net/http"
	// _ "net/http/pprof"

	"obsessiontech/common/config"
	myContext "obsessiontech/common/context"
	"obsessiontech/common/ipc"
	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/receiver/connection"
	"obsessiontech/environment/environment/receiver/engine"
	"obsessiontech/environment/environment/receiver/ipchandler"

	_ "obsessiontech/environment/environment/data/operation"

	_ "obsessiontech/environment/environment/receiver/HJ/hjt212"
	_ "obsessiontech/environment/environment/receiver/fume"
	_ "obsessiontech/environment/environment/receiver/noise"
	_ "obsessiontech/environment/environment/receiver/odor"
	_ "obsessiontech/environment/environment/receiver/thwater"
)

var Config struct {
	SiteID       string
	ReceiverCode string
	TCPPort      string
	IPCConnType  string
	IPCTCPHost   string
}

func init() {
	config.GetConfig("config.yaml", &Config)

	switch Config.IPCConnType {
	case ipc.IPC_TCP:
		if Config.IPCTCPHost == "" {
			panic("ipc tcp host empty")
		}
	case ipc.IPC_UNIX:
	default:
		Config.IPCConnType = ipc.IPC_UNIX
	}
}

func main() {

	// defer func() {
	// 	fmt.Println(http.ListenAndServe("0.0.0.0:8005", nil))
	// }()

	if _, err := environment.GetModule(Config.SiteID); err != nil {
		panic(err)
	}

	if err := entity.LoadStation(Config.SiteID); err != nil {
		panic(err)
	}

	if err := monitor.LoadMonitor(Config.SiteID); err != nil {
		panic(err)
	}

	if err := monitor.LoadMonitorCode(Config.SiteID); err != nil {
		panic(err)
	}

	if err := monitor.LoadFlagLimit(Config.SiteID); err != nil {
		panic(err)
	}

	tcpAddress, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%s", Config.TCPPort))
	if err != nil {
		log.Panic(err)
	}

	tcp, err := net.ListenTCP("tcp4", tcpAddress)
	if err != nil {
		log.Panic(err)
	}

	log.Println("tcp listener started")

	ctx, cancel := myContext.GetContext()
	closeListener := func() {
		tcp.Close()
		cancel()
	}

	listener := make(chan *net.TCPConn)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("receiver listener stop")
				return
			default:
			}
			conn, err := tcp.AcceptTCP()
			if err != nil {
				log.Println("error accept", err)
				continue
			}

			listener <- conn
		}
	}()

	if Config.IPCConnType == ipc.IPC_UNIX {
		if err := ipchandler.StartIPC(ipc.IPC_UNIX, fmt.Sprintf("/tmp/envrecv_%s_%s.sock", Config.SiteID, Config.ReceiverCode)); err != nil {
			log.Println("error start ipc: ", err)
			closeListener()
			return
		}
	} else if Config.IPCConnType == ipc.IPC_TCP {
		if err := ipchandler.StartIPC(ipc.IPC_TCP, Config.IPCTCPHost); err != nil {
			log.Println("error start ipc: ", err)
			closeListener()
			return
		}
	}

	log.Println("ipc socket host started")

	for {
		select {
		case conn := <-listener:

			out := new(connection.Connection)
			out.Conn = conn

			childCtx, childCancel := myContext.GetContext()
			out.Ctx, out.Cancel = context.WithCancel(childCtx)

			go func() {
				<-out.Ctx.Done()
				log.Println("connection close")
				conn.Close()
				childCancel()
			}()

			go engine.EstablishConnection(out)

			log.Println("debug goroutines: ", runtime.NumGoroutine())

		case <-ctx.Done():
			log.Println("receiver listener closing")
			closeListener()
			//这里不要return中断主程序 用select阻塞 让context来执行gracefully exit 详见common/context
			select {}
		}
	}
}
