package extension

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/mission"
	"obsessiontech/environment/role"
)

type PointMission struct {
	mission.BaseMissionComplete
	PointID     int           `json:"pointID"`
	Point       int           `json:"point,omitempty"`
	PointScales []*PointScale `json:"pointScales,omitempty"`
}

type PointScale struct {
	Require int `json:"require"`
	Base    int `json:"base"`
	Rate    int `json:"rate"`
}

func init() {
	mission.RegisterMission("userpoint", func() mission.IMissionComplete {
		return new(PointMission)
	})
}

func (m *PointMission) getPoint(score int) int {

	if m.Point > 0 {
		return m.Point
	}

	var scale *PointScale
	for _, s := range m.PointScales {
		if score < s.Require {
			break
		}
		scale = s
	}

	if scale == nil {
		return 0
	}

	log.Println("scale match: ", score, scale.Require, scale.Base, scale.Rate)

	result := scale.Base
	if scale.Rate > 0 {
		overflow := score - scale.Require
		result += scale.Rate * overflow
		log.Println("scale step: ", overflow, scale.Rate*overflow, result)
	}

	return result
}

func (m *PointMission) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, done *mission.Mission, missions map[int]*mission.Mission, complete *mission.Complete) error {

	var score int
	result, ok := complete.Result[m.ID]
	if ok {
		score, ok = result.(int)
		if !ok {
			return errors.New("invalid score")
		}
	}
	point := m.getPoint(score)
	complete.Ext[m.ID] = point
	return role.ChangeUserPoint(siteID, txn, complete.UID, m.PointID, point, "MISSION", fmt.Sprintf("完成[%s]", done.Name), map[string]interface{}{"completeID": complete.ID}, false)
}

func (m *PointMission) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, done *mission.Mission, complete *mission.Complete) error {
	var sum int
	if pre, exists := complete.Ext[m.ID]; exists {
		if preInt, isInt := pre.(int); isInt {
			sum = preInt
		} else if preFloat, isFloat := pre.(float64); isFloat {
			sum = int(preFloat)
		}
		if sum > 0 {
			complete.Ext[m.ID] = 0
			return role.ChangeUserPoint(siteID, txn, complete.UID, m.PointID, -1*sum, "MISSION_REVERT", fmt.Sprintf("撤销[%s]", done.Name), map[string]interface{}{"completeID": complete.ID}, true)
		}
	}

	return nil
}
