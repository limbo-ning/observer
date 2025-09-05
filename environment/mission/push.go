package mission

import (
	"errors"
	"log"

	"obsessiontech/environment/push"
)

const (
	PUSH_MISSION = "mission"
)

func init() {
	push.Register(PUSH_MISSION, new(MissionPusher))
}

var e_invalid_mission_push_config = errors.New("未实任务推送接口")

type IMissionPush interface {
	GetMission(string) (*Mission, error)
	GetMissionEmpowers(string) (map[string]map[string][]string, error)
}

type MissionPusher struct{}

func (p *MissionPusher) Validate(siteID string, ipush push.IPush) error {

	i, ok := ipush.(IMissionPush)
	if !ok {
		log.Println("error validate not implement IMissionPush")
		return e_invalid_mission_push_config
	}

	_, err := i.GetMissionEmpowers(siteID)
	if err != nil {
		return err
	}

	return nil
}
func (p *MissionPusher) Push(siteID string, ipush push.IPush) error {
	i, ok := ipush.(IMissionPush)
	if !ok {
		log.Println("error push not implement IMissionPush")
		return e_invalid_mission_push_config
	}

	m, err := i.GetMission(siteID)
	if err != nil {
		return err
	}

	empowers, err := i.GetMissionEmpowers(siteID)
	if err != nil {
		return err
	}

	if err := m.Add(siteID); err != nil {
		return err
	}

	for action, empowers := range empowers {
		for empower, empowerIDs := range empowers {
			if err := AddMissionEmpower(siteID, m.ID, empower, empowerIDs, []string{action}); err != nil {
				return err
			}
		}
	}

	return nil
}
