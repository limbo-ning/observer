package mission

import (
	"database/sql"
	"errors"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMission("review", func() IMissionComplete { return new(Review) })
}

type Review struct {
	BaseMissionComplete
}

func (p *Review) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	if !actionAuth.CheckAction(ACTION_REVIEW, ACTION_ADMIN_REVIEW) {
		return nil
	}

	result := complete.Result[p.ID]
	if result == nil {
		return nil
	}

	status, ok := result.(string)
	if !ok {
		return errors.New("需要任务终结状态")
	}

	switch status {
	case MISSION_CLOSED:
	case MISSION_SUCCESS:
	case MISSION_FAILED:
	default:
		return errors.New("需要任务终结状态")
	}

	mission.Status = status

	return E_mission_interrupt
}

func (p *Review) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {
	return nil
}
