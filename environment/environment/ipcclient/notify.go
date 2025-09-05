package ipcclient

import (
	"log"
	"time"

	"obsessiontech/environment/environment/ipcmessage"
)

func NotifyModuleChange(siteID string) (done, fail, timeout int) {
	return notify(siteID, new(ipcmessage.ModuleReloadReq))
}
func NotifyStationChange(siteID string, stationID int) (done, fail, timeout int) {
	req := ipcmessage.StationReloadReq(stationID)
	return notify(siteID, &req)
}
func NotifyMonitorChange(siteID string) (done, fail, timeout int) {
	return notify(siteID, new(ipcmessage.MonitorReloadReq))
}
func NotifyMonitorCodeChange(siteID string) (done, fail, timeout int) {
	return notify(siteID, new(ipcmessage.MonitorCodeReloadReq))
}
func NotifyFlagLimitChange(siteID string) (done, fail, timeout int) {
	return notify(siteID, new(ipcmessage.FlagLimitReloadReq))
}

func notify(siteID string, req ipcmessage.IMessage) (done, fail, timeout int) {

	log.Println("notify: ", siteID, req.GetIPCMessageType())

	for _, hostAddr := range Config.EnvironmentReceiverAddrs {
		if hostAddr.SiteID == siteID {

			log.Println("notifying ", hostAddr.SiteID, hostAddr.Addr, req.GetIPCMessageType())

			send := make(chan ipcmessage.IMessage)
			receive, err := StartRequestClient(siteID, hostAddr.ConnType, hostAddr.Addr, send)
			if err != nil {
				log.Println("notify err establishing: ", err)
				fail++
				continue
			}

			log.Println("notify channel established")
			send <- req
			log.Println("nofity req sent: ", req.GetIPCMessageType())

			select {
			case res, ok := <-receive:
				log.Println("notify received: ", res, ok)
				if !ok {
					fail++
				} else {
					if ack, ok := res.(*ipcmessage.Ack); ok && *ack == 0 {
						done++
					} else {
						fail++
					}
				}
			case <-time.After(Config.EnvironmentReceiverRequestTimeOutSec * time.Second):
				log.Println("notify timeout")
				timeout++
			}

			close(send)
		}
	}

	return
}
