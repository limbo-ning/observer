package operation

import (
	"errors"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/monitor"
	"time"
)

func Modify(siteID string, d data.IData, fields []string, uid int) (data.IData, error) {

	if len(fields) == 0 {
		if d.GetDataType() == data.REAL_TIME {
			fields = []string{data.RTD, data.FLAG}
		} else {
			fields = []string{data.AVG, data.MIN, data.MAX, data.COU, data.FLAG}
		}
	}

	list, err := data.GetData(siteID, d.GetDataType(), []int{d.GetStationID()}, []int{d.GetMonitorID()}, []int{d.GetMonitorCodeID()}, nil, time.Time(d.GetDataTime()), time.Time(d.GetDataTime()), nil, data.ORIGIN_DATA)
	if err != nil {
		return nil, err
	}

	var origin data.IData
	if len(list) == 0 {
		origin = d
	} else {
		origin = list[0]
	}

	for _, f := range fields {
		switch f {
		case data.RTD:
			if rtd, ok := d.(data.IRealTime); ok {
				if err := data.ModifyValue(origin, data.RTD, rtd.GetRtd(), uid); err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("数据字段不符")
			}
		case data.AVG:
			if interval, ok := d.(data.IInterval); ok {
				if err := data.ModifyValue(origin, data.AVG, interval.GetAvg(), uid); err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("数据字段不符")
			}
		case data.MIN:
			if interval, ok := d.(data.IInterval); ok {
				if err := data.ModifyValue(origin, data.MIN, interval.GetMin(), uid); err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("数据字段不符")
			}
		case data.MAX:
			if interval, ok := d.(data.IInterval); ok {
				if err := data.ModifyValue(origin, data.MAX, interval.GetMax(), uid); err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("数据字段不符")
			}
		case data.COU:
			if interval, ok := d.(data.IInterval); ok {
				if err := data.ModifyValue(origin, data.COU, interval.GetCou(), uid); err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("数据字段不符")
			}
		case data.FLAG:
			if err := monitor.ChangeFlag(siteID, origin, d.GetFlag(), uid); err != nil {
				return nil, err
			}
			fields = append(fields, data.FLAG_BIT)
		}
	}

	if len(list) == 0 {
		return origin, data.Add(siteID, origin)
	}

	return origin, data.Update(siteID, origin, fields...)
}
