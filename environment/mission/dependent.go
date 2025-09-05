package mission

import (
	"database/sql"
	"log"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
)

func init() {
	RegisterMissionPrerequisite("completeDependent", func() IMissionPrerequisite { return new(CompleteDependent) })
}

type CompleteDependent struct {
	BaseMissionPrerequisite
	IsSkip                        bool           `json:"isSkip,omitempty"`
	DependentCompleteStatusCount  map[string]int `json:"dependentCompleteStatusCount"`
	IsGlobalDependent             bool           `json:"isGlobalDependent,omitempty"`
	DependentMissionID            []int          `json:"dependentMissionID,omitempty"`
	DependentMissionType          string         `json:"dependentMissionType,omitempty"`
	SequentialOffset              int            `json:"sequentialOffset,omitempty"`
	SequentialDelayActiveInterval *util.Interval `json:"sequentialDelayActiveInterval,omitempty"`
	SequentialDelayActiveDuration *util.Duration `json:"sequentialDelayActiveDuration,omitempty"`
}

func (p *CompleteDependent) UpdateCompleteRequire(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error {

	var uid int
	if !p.IsGlobalDependent {
		uid = actionAuth[0].UID
	} else {
		uid = -1
	}

	if len(p.DependentMissionID) == 0 && p.DependentMissionType == "" {
		log.Println("dependent no mission")
		return nil
	}

	missionIDs := make([]int, 0)
	for mid := range p.DependentMissionID {
		if _, exists := missions[mid]; !exists {
			missionIDs = append(missionIDs, mid)
		}
	}

	missionList, err := getMissionsWithTxn(siteID, txn, false, p.DependentMissionType, missionIDs...)
	if err != nil {
		return err
	}

	log.Println("dependent missionlist: ", len(missionList))

	for _, m := range missionList {

		missions[m.ID] = m

		offsetTargetTime := time.Time(targetTime)
		for i := 0; i < p.SequentialOffset; i++ {
			sectionStart, _, err := m.Section.GetInterval(offsetTargetTime, true)
			if err != nil {
				if err == util.E_not_in_interval {
					return nil
				}
				return err
			}
			if sectionStart != nil {
				offsetTargetTime = sectionStart.Add(-1 * time.Second)
			}
		}

		self, exists := completeRequests[uid]
		if !exists {
			self = new(MissionCompleteRequest)
			self.Status = make(map[int][]string)
			self.TargetTimes = make(map[int][]*time.Time)
			completeRequests[uid] = self
		}

		if _, exists := self.TargetTimes[m.ID]; !exists {
			self.TargetTimes[m.ID] = make([]*time.Time, 0)
		}
		if _, exists := self.Status[m.ID]; !exists {
			self.Status[m.ID] = make([]string, 0)
		}

		existsTime := false
		for _, t := range self.TargetTimes[m.ID] {
			if t.Equal(offsetTargetTime) {
				existsTime = true
				break
			}
		}
		if !existsTime {
			self.TargetTimes[m.ID] = append(self.TargetTimes[m.ID], &offsetTargetTime)
		}

		for status := range p.DependentCompleteStatusCount {
			self.Status[m.ID] = append(self.Status[m.ID], status)
		}
	}

	return nil
}

func (p *CompleteDependent) CheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error {

	if mission.Status == MISSION_INACTIVE {
		return nil
	}

	if len(p.DependentMissionID) == 0 && p.DependentMissionType == "" {
		log.Println("dependent no mission")
		return nil
	}

	var lastCompleteTime time.Time

	dependentMissionIDs := make(map[int]bool)
	for _, mid := range p.DependentMissionID {
		dependentMissionIDs[mid] = true
	}

	for _, m := range missions {
		log.Println("dependent check: ", m.ID)
		if m.Type == p.DependentMissionType || dependentMissionIDs[m.ID] {
			completeTime, err := p.checkDependent(siteID, actionAuth, targetTime, m, completeRequests)
			if err != nil {
				return err
			}

			if completeTime == nil {
				if !p.IsSkip {
					mission.Status = MISSION_INACTIVE
				}
				return E_mission_interrupt
			}

			if lastCompleteTime.IsZero() || completeTime.After(lastCompleteTime) {
				lastCompleteTime = *completeTime
			}
		}
	}

	if p.SequentialDelayActiveDuration != nil {
		if time.Now().Before(lastCompleteTime.Add(p.SequentialDelayActiveDuration.GetDuration())) {
			if !p.IsSkip {
				mission.Status = MISSION_INACTIVE
			}
			return E_mission_interrupt
		}
	}
	if p.SequentialDelayActiveInterval != nil {
		_, completeSectionEnd, err := p.SequentialDelayActiveInterval.GetInterval(lastCompleteTime, true)
		if err != nil && err != util.E_not_in_interval {
			return err
		}

		if completeSectionEnd != nil && !completeSectionEnd.Before(time.Now()) {
			log.Println("sequential delay interval fail 1: ", mission.Name, lastCompleteTime, completeSectionEnd)
			if !p.IsSkip {
				mission.Status = MISSION_INACTIVE
			}
			return E_mission_interrupt
		}
		_, _, err = p.SequentialDelayActiveInterval.GetInterval(time.Now(), true)
		if err == util.E_not_in_interval {
			log.Println("sequential delay interval fail 2: ", mission.Name, lastCompleteTime, completeSectionEnd)
			if !p.IsSkip {
				mission.Status = MISSION_INACTIVE
			}
			return E_mission_interrupt
		}
	}

	return nil
}

func (p *CompleteDependent) checkDependent(siteID string, actionAuth authority.ActionAuthSet, targetTime time.Time, m *Mission, completeRequests map[int]*MissionCompleteRequest) (*time.Time, error) {
	offsetTargetTime := time.Time(targetTime)
	for i := 0; i < p.SequentialOffset; i++ {
		sectionStart, _, err := m.Section.GetInterval(offsetTargetTime, true)
		if err != nil {
			if err == util.E_not_in_interval {
				return nil, nil
			}
			return nil, err
		}
		if sectionStart != nil {
			offsetTargetTime = sectionStart.Add(-1 * time.Second)
		} else {
			break
		}
	}

	targetSectionStart, targetSectionEnd, err := m.Section.GetInterval(offsetTargetTime, true)
	if err != nil {
		if err == util.E_not_in_interval {
			return nil, nil
		}
		return nil, err
	}

	var uid int
	if !p.IsGlobalDependent {
		uid = actionAuth[0].UID
	} else {
		uid = -1
	}

	var lastCompleteTime *time.Time

	userCount := completeRequests[uid]
	if userCount != nil {
		counts := userCount.Result[m.ID]
		if counts != nil {
			for status, require := range p.DependentCompleteStatusCount {

				count := 0

				if counts[status] != nil {
					for _, completeTime := range counts[COMPLETE_FINAL] {
						if targetSectionStart != nil && completeTime.Before(*targetSectionStart) {
							continue
						}
						if targetSectionEnd != nil && completeTime.After(*targetSectionEnd) {
							continue
						}
						if lastCompleteTime == nil || completeTime.After(*lastCompleteTime) {
							lastCompleteTime = completeTime
						}
						count++
					}
				}

				log.Println("check dependent count: ", uid, m.ID, count, require)

				if count < require {
					return nil, nil
				}
			}
		}
	}

	return lastCompleteTime, nil
}
