package role

import (
	"database/sql"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/mission"
)

type RoleMission struct {
	mission.BaseMissionComplete
	OriRoleID  int `json:"oriRoleID,omitempty"`
	DestRoleID int `json:"destRoleID,omitempty"`
	Expires    int `json:"expires,omitempty"`
}

func init() {
	mission.RegisterMission("userrole", func() mission.IMissionComplete { return new(RoleMission) })
}

func (m *RoleMission) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, done *mission.Mission, missions map[int]*mission.Mission, complete *mission.Complete) error {

	uid := actionAuth[0].UID

	if m.OriRoleID > 0 {
		if err := unbindUserRole(siteID, txn, uid, m.OriRoleID); err != nil {
			return err
		}
	}

	if m.DestRoleID > 0 {
		if _, err := bindUserRole(siteID, txn, uid, m.DestRoleID, m.Expires); err != nil {
			return err
		}
	}
	return nil
}

func (m *RoleMission) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, done *mission.Mission, complete *mission.Complete) error {

	uid := actionAuth[0].UID

	if m.DestRoleID > 0 {
		if err := unbindUserRole(siteID, txn, uid, m.DestRoleID); err != nil {
			return err
		}
	}

	if m.OriRoleID > 0 {
		if _, err := bindUserRole(siteID, txn, uid, m.OriRoleID, m.Expires); err != nil {
			return err
		}
	}
	return nil
}
