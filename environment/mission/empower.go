package mission

import (
	"database/sql"
	"fmt"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

func CheckAuth(siteID string, actionAuth authority.ActionAuthSet, missionID int, authType ...string) error {

	actions := make(map[string]string)
	for _, a := range authType {
		adminAction, exists := AdminActions[a]
		if !exists {
			return authority.E_no_empower
		}

		if actionAuth.CheckAction(adminAction) {
			return nil
		}

		if !actionAuth.CheckAction(a) {
			return authority.E_no_empower
		}

		actions[a] = adminAction
	}

	authToCheck := make(map[string]byte)

	for _, a := range authType {
		authToCheck[a] = 1
	}

	empowers, err := authority.GetEmpowers(siteID, "mission", actionAuth, actions, fmt.Sprintf("%d", missionID))
	if err != nil {
		return err
	}

	if empowers == nil {
		return nil
	}

	missionEmpowers, exists := empowers[fmt.Sprintf("%d", missionID)]
	if !exists {
		return authority.E_no_empower
	}

	for _, a := range authType {
		if _, exists := missionEmpowers[a]; !exists {
			return authority.E_no_empower
		}
	}

	return nil
}

func AddMissionEmpower(siteID string, missionID int, empower string, empowerID []string, authType []string) error {

	return datasource.Txn(func(txn *sql.Tx) {

		missions, err := getMissionsWithTxn(siteID, txn, true, "", missionID)
		if err != nil {
			panic(err)
		}

		if len(missions) == 0 {
			panic(E_mission_not_found)
		}

		if err := authority.AddEmpower(siteID, "mission", fmt.Sprintf("%d", missionID), txn, empower, empowerID, authType); err != nil {
			panic(err)
		}
	})
}

func DeleteMissionEmpower(siteID string, missionID int, empower string, empowerID []string, authType ...string) error {

	if len(authType) == 0 {
		return nil
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := authority.DeleteEmpower(siteID, "mission", fmt.Sprintf("%d", missionID), txn, empower, empowerID, authType...); err != nil {
			panic(err)
		}
	})
}
