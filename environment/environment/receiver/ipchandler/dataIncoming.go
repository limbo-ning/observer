package ipchandler

import (
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/ipcmessage"
)

func ReportData(incoming data.IData) {
	switch incoming.(type) {
	case (*data.RealTimeData):
		convert := ipcmessage.RealTime(*incoming.(*data.RealTimeData))
		broadcast(&convert)
	case (*data.MinutelyData):
		convert := ipcmessage.Minutely(*incoming.(*data.MinutelyData))
		broadcast(&convert)
	case (*data.HourlyData):
		convert := ipcmessage.Hourly(*incoming.(*data.HourlyData))
		broadcast(&convert)
	case (*data.DailyData):
		convert := ipcmessage.Daily(*incoming.(*data.DailyData))
		broadcast(&convert)
	}
}
