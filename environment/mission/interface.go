package mission

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"obsessiontech/environment/authority"
)

var missions = make(map[string]func() IMissionComplete)

func RegisterMission(missionType string, fac func() IMissionComplete) {
	if _, exists := missions[missionType]; exists {
		panic("duplicate mission type: " + missionType)
	}
	missions[missionType] = fac
}

var E_mission_interrupt = errors.New("interrupt")

type IMissionComplete interface {
	GetID() string
	SetID(string)
	GetType() string
	Validate(siteID string) error
	MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error
	MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error
}

type BaseMissionComplete struct {
	ID   string `json:"ID"`
	Type string `json:"type"`
}

func (m *BaseMissionComplete) GetID() string   { return m.ID }
func (m *BaseMissionComplete) SetID(ID string) { m.ID = ID }
func (m *BaseMissionComplete) GetType() string { return m.Type }
func (m *BaseMissionComplete) Validate(siteID string) error {
	return nil
}

func GetMissionComplete(missionType string) IMissionComplete {
	fac, exists := missions[missionType]
	if !exists {
		return nil
	}
	return fac()
}

type MissionCompletes []IMissionComplete

func (missions *MissionCompletes) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	list := make([]IMissionComplete, 0)

	IDs := make(map[string]byte)

	for _, r := range raw {
		var t BaseMissionComplete
		if err := json.Unmarshal(r, &t); err != nil {
			return err
		}

		instance := GetMissionComplete(t.Type)
		if instance == nil {
			log.Printf("error mission type not exists:%s", t.Type)
			return errors.New("未知的任务完成")
		}

		if err := json.Unmarshal([]byte(r), instance); err != nil {
			return err
		}

		if instance.GetID() != "" {
			IDs[instance.GetID()] = 1
		}

		list = append(list, instance)
	}

	for i, m := range list {
		if m.GetID() == "" {
			j := i
			id := fmt.Sprintf("%s_%d", m.GetType(), j)
			for {
				if _, exists := IDs[id]; !exists {
					m.SetID(id)
					IDs[id] = 1
					break
				}
				j++
			}
		}
	}

	*missions = list

	return nil
}

var prerequisites = make(map[string]func() IMissionPrerequisite)

func RegisterMissionPrerequisite(prerequisiteType string, fac func() IMissionPrerequisite) {
	if _, exists := missions[prerequisiteType]; exists {
		panic("duplicate mission prerequisite type: " + prerequisiteType)
	}
	prerequisites[prerequisiteType] = fac
}

type IMissionPrerequisite interface {
	GetType() string
	Validate(siteID string) error
	CheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error
}

type BaseMissionPrerequisite struct {
	Type string `json:"type"`
}

func (m *BaseMissionPrerequisite) GetType() string              { return m.Type }
func (m *BaseMissionPrerequisite) Validate(siteID string) error { return nil }

func GetMissionPrerequisite(prerequisiteType string) IMissionPrerequisite {
	fac, exists := prerequisites[prerequisiteType]
	if !exists {
		return nil
	}
	return fac()
}

type MissionPrerequisites []IMissionPrerequisite

func (missions *MissionPrerequisites) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	list := make([]IMissionPrerequisite, 0)

	for _, r := range raw {
		var t BaseMissionPrerequisite
		if err := json.Unmarshal(r, &t); err != nil {
			return err
		}

		instance := GetMissionPrerequisite(t.Type)
		if instance == nil {
			log.Printf("error mission prerequisite type not exists:%s", t.Type)
			continue
		}

		if err := json.Unmarshal([]byte(r), instance); err != nil {
			return err
		}

		list = append(list, instance)
	}

	*missions = list

	return nil
}

type MissionCompleteRequest struct {
	Status      map[int][]string
	TargetTimes map[int][]*time.Time
	Result      map[int]map[string][]*time.Time
}
type IMissionRequireCompletePrerequisite interface {
	UpdateCompleteRequire(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest) error
}

type IMissionRequireAppendMissionCheckingPrerequisite interface {
	AppendMissionChecking(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, missions map[int]*Mission) ([]*Mission, error)
}

type IMissionPostCheckPrerequisite interface {
	PostCheckMission(siteID string, txn *sql.Tx, mission *Mission, actionAuth authority.ActionAuthSet, targetTime time.Time, relavents map[int]*Mission) error
}
