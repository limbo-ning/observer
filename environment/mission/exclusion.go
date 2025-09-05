package mission

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMissionPrerequisite("exclusion", func() IMissionPrerequisite { return new(Exclusion) })
}

type Exclusion struct {
	BaseMissionPrerequisite
	Exclusions map[int]int `json:"exclusions"`
}

func (p *Exclusion) AppendMissionChecking(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, missions map[int]*Mission) ([]*Mission, error) {

	fetchIDs := make([]int, 0)

	for mid := range p.Exclusions {
		if _, exists := missions[mid]; !exists {
			fetchIDs = append(fetchIDs, mid)
		}
	}

	toAppends, err := getMissionsWithTxn(siteID, txn, false, "", fetchIDs...)
	if err != nil {
		return nil, err
	}

	for _, m := range toAppends {
		missions[m.ID] = m
	}

	return toAppends, nil
}

func (p *Exclusion) CheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error {
	return nil
}

func (p *Exclusion) PostCheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission) error {

	for mid, value := range p.Exclusions {
		check, exists := missions[mid]
		if !exists {
			log.Println("post check exclusion: 校验任务不存在", mission.ID, mid, value)
			return fmt.Errorf("校验任务不存在[%d]", mid)
		}

		if check.Status == MISSION_ACTIVE && check.Quota >= value {
			mission.Status = MISSION_INACTIVE
		}
	}

	return nil
}
