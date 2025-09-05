package mission

import (
	"database/sql"
	"log"
	"time"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMission("clear", func() IMissionComplete { return new(Clear) })
}

type Clear struct {
	BaseMissionComplete
	CompleteStatus       string `json:"completeStatus"`
	MissionType          string `json:"missionType,omitempty"`
	MissionIDs           []int  `json:"missionIDs,omitempty"`
	IncludeBeforeSection bool   `json:"includeBeforeSection,omitempty"`
}

func (p *Clear) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	missionList, err := getMissionsWithTxn(siteID, txn, false, p.MissionType, p.MissionIDs...)
	if err != nil {
		return err
	}
	log.Println("get missions to clear: ", len(missionList))

	if len(missionList) == 0 {
		return nil
	}

	now := time.Now()

	toClearMissions := make(map[int]*Mission)
	missionTargetTimes := make(map[int][]*time.Time)
	for _, m := range missionList {
		toClearMissions[m.ID] = m
		if !p.IncludeBeforeSection {
			missionTargetTimes[m.ID] = []*time.Time{&now}
		}
	}

	completes, err := getCompletes(siteID, txn, actionAuth[0].UID, []string{p.CompleteStatus}, toClearMissions, missionTargetTimes, true)
	if err != nil {
		return err
	}

	completeIDs := make([]int, 0)

	for _, list := range completes {
		for _, complete := range list {
			completeIDs = append(completeIDs, complete.ID)
			complete.Status = COMPLETE_CLOSED
			if err := complete.update(siteID, txn); err != nil {
				return err
			}
		}
	}

	complete.Ext[p.ID] = completeIDs

	return nil
}

func (p *Clear) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	recordIDs, exists := complete.Ext[p.ID]
	if !exists {
		return nil
	}

	completes, err := getComplete(siteID, txn, true, recordIDs.([]int)...)
	if err != nil {
		return err
	}

	for _, complete := range completes {
		complete.Status = p.CompleteStatus
		if err := complete.update(siteID, txn); err != nil {
			return err
		}
	}

	return nil
}
