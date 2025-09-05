package mission

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMission("highscore", func() IMissionComplete { return new(HighScore) })
}

type HighScore struct {
	BaseMissionComplete
	IsRevertPrevious bool `json:"isRevertPrevious,omitempty"`
}

func (p *HighScore) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	completeTime := time.Time(complete.CompleteTime)
	completes, err := getCompletes(siteID, txn, actionAuth[0].UID, []string{COMPLETE_FINAL}, missions, map[int][]*time.Time{mission.ID: {&completeTime}}, false)
	if err != nil {
		return err
	}

	var score int
	result, ok := complete.Result[p.ID]
	if !ok {
		return errors.New("need highscore")
	}
	score, ok = result.(int)
	if !ok {
		return errors.New("need highscore")
	}

	list, exists := completes[mission.ID]
	var previous *Complete
	var highest int
	if exists {
		for _, c := range list {
			if c.ID == complete.ID {
				continue
			}
			highscore, exists := c.Result[p.ID]
			if !exists {
				continue
			}
			if hs, ok := highscore.(int); !ok {
				continue
			} else {
				if hs >= score {
					return E_mission_interrupt
				}
				if hs > highest {
					previous = c
					highest = hs
				}
			}
		}
	}

	if p.IsRevertPrevious && previous != nil {
		log.Println("to revert: ", previous.ID, complete.ID)
		if err := revertMissionComplete(siteID, txn, actionAuth, mission, previous); err != nil {
			return err
		}
	}

	complete.Ext[p.ID] = true

	return nil
}

func (p *HighScore) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	if flag, exist := complete.Ext[p.ID]; exist {
		if flag.(bool) {
			return nil
		}
		return E_mission_interrupt
	}

	return nil
}
