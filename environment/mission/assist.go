package mission

import (
	"database/sql"
	"errors"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMission("assist", func() IMissionComplete { return new(Assist) })
}

type Assist struct {
	BaseMissionComplete
	Requirement int `json:"requirement"`
}

func (p *Assist) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	result := complete.Result[p.ID]
	if result == nil {
		var assistList []int
		list, exists := complete.Ext[p.ID]
		if exists {
			assistList = list.([]int)
		}

		if len(assistList) >= p.Requirement {
			complete.Status = COMPLETE_FINAL
			return nil
		}

		complete.Status = COMPLETE_PENDING
		return E_mission_interrupt
	}

	assistCompleteID, ok := result.(int)
	if !ok {
		return errors.New("无效记录ID")
	}

	completes, err := getComplete(siteID, txn, true, assistCompleteID)
	if err != nil {
		return err
	}

	if len(completes) == 0 {
		return errors.New("找不到记录")
	}

	assistComplete := completes[0]

	var assistList []int
	list, exists := assistComplete.Ext[p.ID]
	if !exists {
		assistList = make([]int, 0)
	} else {
		assistList = list.([]int)
	}

	assistList = append(assistList, complete.ID)
	assistComplete.Ext[p.ID] = assistList

	for _, mc := range mission.Completes {
		if err := mc.MissionComplete(siteID, txn, authority.ActionAuthSet{{UID: assistComplete.UID}}, mission, missions, assistComplete); err != nil {
			if err == E_mission_interrupt {
				break
			} else {
				return err
			}
		}
	}

	if err := assistComplete.update(siteID, txn); err != nil {
		return err
	}

	complete.Status = COMPLETE_CLOSED
	return E_mission_interrupt
}

func (p *Assist) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	return nil
}
