package entity

import (
	"database/sql"
	"fmt"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

func init() {
	authority.RegisterEmpower(MODULE_ENTITY+"#entity", func() authority.IEmpower {
		return new(EntityEmpower)
	})
	authority.RegisterEmpower(MODULE_ENTITY+"#station", func() authority.IEmpower {
		return new(StationEmpower)
	})
}

type EntityEmpower struct{}

func (e *EntityEmpower) EmpowerID(siteID string, actionAuth authority.ActionAuthSet) ([]string, error) {
	return nil, authority.E_empower_not_restricted
}

type StationEmpower struct{}

func (e *StationEmpower) EmpowerID(siteID string, actionAuth authority.ActionAuthSet) ([]string, error) {
	return nil, authority.E_empower_not_restricted
}

func CheckAuth(siteID string, actionAuth authority.ActionAuthSet, entityID int, authType ...string) error {

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

	empowers, err := authority.GetEmpowers(siteID, "entity", actionAuth, actions, fmt.Sprintf("%d", entityID))
	if err != nil {
		return err
	}

	if empowers == nil {
		return nil
	}

	entityEmpowers, exists := empowers[fmt.Sprintf("%d", entityID)]
	if !exists {
		return authority.E_no_empower
	}

	for _, a := range authType {
		if _, exists := entityEmpowers[a]; !exists {
			return authority.E_no_empower
		}
	}

	return nil
}

func AddEntityEmpower(siteID string, actionAuth authority.ActionAuthSet, entityID int, empower string, empowerID []string, authType []string) error {

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
		if err := CheckAuth(siteID, actionAuth, entityID, append(authType, ACTION_ENTITY_EDIT)...); err != nil {
			return err
		}
	}

	entities, err := GetEntities(siteID, entityID)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		return e_need_entity
	}

	return datasource.Txn(func(txn *sql.Tx) {

		if err := authority.AddEmpower(siteID, "entity", fmt.Sprintf("%d", entityID), txn, empower, empowerID, authType); err != nil {
			panic(err)
		}
	})
}

func DeleteEntityEmpower(siteID string, actionAuth authority.ActionAuthSet, entityID int, empower string, empowerID []string, authType ...string) error {

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
		if err := CheckAuth(siteID, actionAuth, entityID, append(authType, ACTION_ENTITY_EDIT)...); err != nil {
			return err
		}
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := authority.DeleteEmpower(siteID, "entity", fmt.Sprintf("%d", entityID), txn, empower, empowerID, authType...); err != nil {
			panic(err)
		}
	})
}
