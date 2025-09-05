package mission

import (
	"database/sql"
	"errors"
	"log"
	"math"
	"time"

	"obsessiontech/common/random"
	"obsessiontech/environment/authority"
)

const RANDOM_MISSION = "randomMission"

func init() {
	RegisterMission(RANDOM_MISSION, func() IMissionComplete { return new(RandomMission) })
}

type RandomMission struct {
	BaseMissionComplete
	Possibility map[int]int `json:"possibility"`
}

func (m *RandomMission) AppendMissionChecking(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, missions map[int]*Mission) ([]*Mission, error) {

	fetchIDs := make([]int, 0)

	for mid := range m.Possibility {
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

func (m *RandomMission) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	mid := make([]int, 0)
	for id := range m.Possibility {
		mid = append(mid, id)
	}

	offsetPossibily := make(map[int]int)
	for i, id := range mid {
		sub, exists := missions[id]
		if !exists {
			log.Println("random mission not found: ", id)
			offsetPossibily[i] = 0
			continue
		}

		if sub.Status == MISSION_INACTIVE {
			log.Println("random mission not active: ", id, sub.Name, sub.Quota)
			offsetPossibily[i] = 0
			continue
		}

		if sub.Quota > 0 && sub.CompleteCount > 0 {
			log.Println("random mission need offset: ", id, sub.Name, sub.Status, sub.Quota, sub.CompleteCount)
			remain := sub.Quota - sub.CompleteCount
			if remain < 0 {
				remain = 0
			}
			offsetPossibily[i] = int(math.Ceil(float64(m.Possibility[id]) * float64(remain) / float64(sub.Quota)))
		} else {
			offsetPossibily[i] = m.Possibility[id]
		}
	}

	var total int
	listing := make([]int, 0)
	for i := range mid {
		total += offsetPossibily[i]
		listing = append(listing, total)
	}

	var result *Mission

	picked := random.GetRandomNumber(total + 1) //random return [0, n)

	for i, limit := range listing {
		if i > 0 && picked < limit {
			break
		}
		if offsetPossibily[i] > 0 {
			result = missions[mid[i]]
		}
	}

	if result == nil {
		log.Println("random: no mission", m.Possibility, offsetPossibily, total, listing, picked)
		return errors.New("没有可用的随机任务")
	}

	log.Println("random: ", m.Possibility, offsetPossibily, total, listing, picked, result.ID)

	success, err := completeMission(siteID, txn, actionAuth, result, missions, complete.Result, time.Time(complete.CompleteTime))
	if err != nil {
		return err
	}

	complete.Ext[m.ID] = map[string]interface{}{
		"missionID":  result.ID,
		"completeID": success.ID,
	}

	return nil
}

func (m *RandomMission) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	success := complete.Ext[m.ID]
	if success == nil {
		log.Printf("fatal error: can not find random mission resulting complete: mission[%d] complete[%d] random[%s]", mission.ID, complete.ID, m.ID)
		return nil
	}

	completeID := success.(map[string]interface{})["completeID"]
	if completeID == nil {
		log.Printf("fatal error: can not find random mission resulting complete: mission[%d] complete[%d] random[%s]", mission.ID, complete.ID, m.ID)
		return nil
	}

	return RevertMissionCompleteWithTxn(siteID, txn, actionAuth, completeID.(int))
}
