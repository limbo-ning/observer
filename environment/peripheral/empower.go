package peripheral

import (
	"database/sql"
	"fmt"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

func CheckAuth(siteID string, actionAuth authority.ActionAuthSet, deviceID int, authType ...string) error {

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

	empowers, err := authority.GetEmpowers(siteID, "device", actionAuth, actions, fmt.Sprintf("%d", deviceID))
	if err != nil {
		return err
	}

	if empowers == nil {
		return nil
	}

	deviceEmpowers, exists := empowers[fmt.Sprintf("%d", deviceID)]
	if !exists {
		return authority.E_no_empower
	}

	for _, a := range authType {
		if _, exists := deviceEmpowers[a]; !exists {
			return authority.E_no_empower
		}
	}

	return nil
}

func AddDeviceEmpower(siteID string, actionAuth authority.ActionAuthSet, deviceID int, empower string, empowerID []string, authType []string) error {

	skipAuth := false
	for _, a := range actionAuth {
		switch a.Action {
		case ACTION_ADMIN_EDIT:
			skipAuth = true
		}
		if skipAuth {
			break
		}
	}

	if !skipAuth {
		if err := CheckAuth(siteID, actionAuth, deviceID, append(authType, ACTION_EDIT)...); err != nil {
			return err
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {

		devices, err := getDevice(siteID, txn, true, deviceID)
		if err != nil {
			panic(err)
		}

		if len(devices) == 0 {
			panic(e_device_not_exist)
		}

		if err := authority.AddEmpower(siteID, "device", fmt.Sprintf("%d", deviceID), txn, empower, empowerID, authType); err != nil {
			panic(err)
		}
	})
}

func DeleteDeviceEmpower(siteID string, actionAuth authority.ActionAuthSet, deviceID int, empower string, empowerID []string, authType ...string) error {

	if len(authType) == 0 {
		return nil
	}

	skipAuth := false
	for _, a := range actionAuth {
		switch a.Action {
		case ACTION_ADMIN_EDIT:
			skipAuth = true
		}
		if skipAuth {
			break
		}
	}

	if !skipAuth {
		if err := CheckAuth(siteID, actionAuth, deviceID, append(authType, ACTION_EDIT)...); err != nil {
			return err
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := authority.DeleteEmpower(siteID, "device", fmt.Sprintf("%d", deviceID), txn, empower, empowerID, authType...); err != nil {
			panic(err)
		}
	})
}
