package mission

import (
	"database/sql"
	"errors"
	"time"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMission("stage", func() IMissionComplete { return new(Stage) })
}

type Stage struct {
	BaseMissionComplete
	StageCount int `json:"stageCount"`
}

func (p *Stage) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	completeTime := time.Time(complete.CompleteTime)
	completes, err := getCompletes(siteID, txn, actionAuth.GetUID(), []string{COMPLETE_PENDING, COMPLETE_FINAL}, missions, map[int][]*time.Time{mission.ID: {&completeTime}}, true)
	if err != nil {
		return err
	}

	var stage int
	result, ok := complete.Result[p.ID]
	if ok {
		stage, ok = result.(int)
		if !ok {
			return errors.New("invalid stage")
		}
	}

	list, exists := completes[mission.ID]
	var currentStage int
	if exists {
		for _, c := range list {
			if c.ID == complete.ID {
				continue
			}
			var stage int
			result, ok := complete.Result[p.ID]
			if !ok {
				continue
			}
			stage, ok = result.(int)
			if !ok {
				continue
			}

			if stage > currentStage {
				currentStage = stage
			}
		}
	}

	if stage == 0 {
		stage = currentStage + 1
	}

	if stage > currentStage+1 {
		return errors.New("不可跳过关卡")
	}

	if stage >= p.StageCount {
		stage = p.StageCount
		complete.Status = COMPLETE_FINAL
	} else {
		complete.Status = COMPLETE_PENDING
		return E_mission_interrupt
	}

	complete.Result[p.ID] = stage

	return nil
}

func (p *Stage) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {
	return nil
}
