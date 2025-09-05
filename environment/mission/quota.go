package mission

import (
	"database/sql"
	"time"

	"obsessiontech/environment/authority"
)

func init() {
	RegisterMissionPrerequisite("quota", func() IMissionPrerequisite { return new(Quota) })
}

type Quota struct {
	BaseMissionPrerequisite
	Quota         int  `json:"quota"`
	IsGlobalQuota bool `json:"isGlobalQuota,omitempty"`
}

func (p *Quota) UpdateCompleteRequire(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error {

	var uid int
	if !p.IsGlobalQuota {
		uid = actionAuth[0].UID
	} else {
		uid = -1
	}

	self, exists := completeRequests[uid]
	if !exists {
		self = new(MissionCompleteRequest)
		self.Status = make(map[int][]string)
		self.TargetTimes = make(map[int][]*time.Time)
		completeRequests[uid] = self
	}
	self.TargetTimes[mission.ID] = append(self.TargetTimes[mission.ID], &targetTime)
	self.Status[mission.ID] = append(self.Status[mission.ID], COMPLETE_FINAL)

	return nil

}

func (p *Quota) CheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error {

	if mission.Quota < 0 {
		mission.Quota = 0
	}

	checked := false
	for _, a := range actionAuth {
		switch a.Action {
		case ACTION_ADMIN_COMPLETE:
			checked = true
		case ACTION_COMPLETE:
			checked = true
		}
	}

	if !checked {
		return nil
	}

	if mission.CompleteCount < 0 {
		sectionStart, sectionEnd, err := mission.Section.GetInterval(targetTime, true)
		if err != nil {
			return nil
		}

		var uid int
		if !p.IsGlobalQuota {
			uid = actionAuth[0].UID
		} else {
			uid = -1
		}

		mission.CompleteCount = 0

		userCompletes, exists := completeRequests[uid]
		if exists {
			counts, exists := userCompletes.Result[mission.ID]
			if exists {
				if counts[COMPLETE_FINAL] != nil {
					for _, completeTime := range counts[COMPLETE_FINAL] {
						if sectionStart != nil && completeTime.Before(*sectionStart) {
							continue
						}
						if sectionEnd != nil && completeTime.After(*sectionEnd) {
							continue
						}
						mission.CompleteCount++
					}
				}
			}
		}
	}

	if mission.Status == MISSION_INACTIVE {
		return nil
	}

	mission.Quota += p.Quota
	return nil
}
