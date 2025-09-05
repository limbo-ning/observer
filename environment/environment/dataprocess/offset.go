package dataprocess

import (
	"database/sql"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
)

func init() {
	Register("offset", func() IDataProcessor { return new(offsetProcessor) })
}

type offsetProcessor struct {
	BaseDataProcessor
	Method string   `json:"method"`
	Offset float64  `json:"offset"`
	Fields []string `json:"fields"`
}

func (p *offsetProcessor) ProcessData(siteID string, txn *sql.Tx, entry data.IData, uploader *Uploader, upload IDataUpload) (bool, error) {

	if realTime, ok := entry.(data.IRealTime); ok {
		for _, field := range p.Fields {
			if field == data.RTD {
				data.ModifyValue(entry, data.RTD, p.calucate(realTime.GetRtd()), -1)
				return false, nil
			}
		}
	} else if interval, ok := entry.(data.IInterval); ok {
		for _, field := range p.Fields {
			if field == data.AVG {
				data.ModifyValue(entry, data.AVG, p.calucate(interval.GetAvg()), -1)
			} else if field == data.MIN {
				data.ModifyValue(entry, data.MIN, p.calucate(interval.GetMin()), -1)
			} else if field == data.MAX {
				data.ModifyValue(entry, data.MAX, p.calucate(interval.GetMax()), -1)
			} else if field == data.COU {
				data.ModifyValue(entry, data.COU, p.calucate(interval.GetCou()), -1)
			}
		}
	}

	return false, nil
}

func (p *offsetProcessor) calucate(input float64) float64 {
	switch p.Method {
	case "add":
		return input + p.Offset
	case "multiply":
		accuracy := util.GetAccuracy(input) + util.GetAccuracy(p.Offset)
		return util.ApplyAccuracy(input*p.Offset, accuracy)
	default:
		return input
	}
}
