package ipcclient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"obsessiontech/environment/environment/entity"
	"os/exec"
)

func GetStationLog(siteID string, stationID, lines int) (*string, error) {

	var folder string

	for _, addr := range Config.EnvironmentReceiverAddrs {
		if addr.SiteID == siteID {
			folder = addr.LogFolder
			break
		}
	}

	if folder == "" {
		return nil, errors.New("未配置日志位置")
	}

	stations, err := entity.GetStation(siteID, stationID)
	if err != nil {
		return nil, err
	}

	if len(stations) != 1 {
		return nil, errors.New("找不到站点")
	}

	station := stations[0]

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf(`tail %s -n %d`, fmt.Sprintf("%s/%s.log", folder, station.MN), lines))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	result := string(bytes)

	return &result, nil
}
