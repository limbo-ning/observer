package operation

import (
	"errors"
	"log"
	"time"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
)

func Massage(siteID string, actionAuth authority.ActionAuthSet, dataType string, stationIDs, monitorIDs, monitorCodeIDs []int, beginTime, endTime time.Time, flag string, restoreBeforeProcess, skipNoOrigins bool, processor dataprocess.DataProcessors) (int, error) {

	filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, stationIDs, entity.ACTION_ENTITY_EDIT)
	if err != nil {
		return 0, err
	}

	for _, sid := range stationIDs {
		if !filtered[sid] {
			return 0, errors.New("无权限")
		}
	}

	dataList, err := data.GetData(siteID, dataType, stationIDs, monitorIDs, monitorCodeIDs, nil, beginTime, endTime, nil, data.ORIGIN_DATA)
	if err != nil {
		return 0, err
	}

	if err := monitor.LoadMonitor(siteID); err != nil {
		return 0, err
	}
	if err := monitor.LoadMonitorCode(siteID); err != nil {
		return 0, err
	}
	if err := monitor.LoadFlagLimit(siteID); err != nil {
		return 0, err
	}

	count := 0

	uper := new(dataprocess.Uploader)

	up := new(Upload)
	up.UploaderUID = actionAuth[0].UID
	up.Processors = processor

	for _, d := range dataList {
		if restoreBeforeProcess {
			if restored := data.RestoreValue(d); !restored && skipNoOrigins {
				continue
			}
		}

		if flag != "" {
			if err := monitor.ChangeFlag(siteID, d, flag, actionAuth.GetUID()); err != nil {
				return 0, err
			}
		}

		if len(processor) > 0 {
			if err := processor.Process(siteID, uper, up, d); err != nil {
				return count, err
			}
		} else {
			monitorCode := monitor.GetMonitorCodeByID(siteID, d.GetMonitorCodeID())
			if monitorCode == nil {
				monitorCode = monitor.GetMonitorCodeByCode(siteID, d.GetStationID(), d.GetCode())
			}
			if monitorCode == nil {
				monitorCode = monitor.GetMonitorCodeByStationMonitor(siteID, d.GetStationID(), d.GetMonitorID())
			}

			if monitorCode != nil {
				if err := monitorCode.Processors.Process(siteID, uper, up, d); err != nil {
					return count, err
				}
			} else {
				log.Println("monitor code not found ", d.GetStationID(), d.GetMonitorCodeID())
			}
		}

		count++
	}

	if err := uper.UploadUnuploaded(siteID, up); err != nil {
		return count, err
	}

	return count, nil
}
