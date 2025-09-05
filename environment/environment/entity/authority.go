package entity

import "obsessiontech/environment/authority"

func FilterEntityStationAuth(siteID string, actionAuth authority.ActionAuthSet, stationID []int, authType ...string) (map[int]bool, error) {
	stations, err := GetStation(siteID, stationID...)
	if err != nil {
		return nil, err
	}

	entityStationMapping := make(map[int][]int)

	for _, s := range stations {
		mapping, exists := entityStationMapping[s.EntityID]
		if !exists {
			mapping = make([]int, 0)
		}
		mapping = append(mapping, s.ID)
		entityStationMapping[s.EntityID] = mapping
	}

	result := make(map[int]bool)
	for entityID, list := range entityStationMapping {
		if CheckAuth(siteID, actionAuth, entityID, authType...) != nil {
			for _, id := range list {
				result[id] = false
			}
		} else {
			for _, id := range list {
				result[id] = true
			}
		}
	}

	for _, id := range stationID {
		if id == 0 {
			result[id] = true
		}
		if id < 0 {
			if actionAuth.CheckAction(ACTION_ADMIN_VIEW, ACTION_ADMIN_EDIT) {
				result[id] = true
			}
		}
	}

	return result, nil
}

type IEntityAuth interface {
	GetEntityID() int
}

func FilterEntityAuthInterface(siteID string, authList []IEntityAuth, actionAuth authority.ActionAuthSet, authType ...string) ([]IEntityAuth, error) {

	filtered := make(map[int]bool)

	for _, a := range authList {
		filtered[a.GetEntityID()] = false
	}

	for entityID := range filtered {

		if entityID <= 0 {
			filtered[entityID] = true
		} else if CheckAuth(siteID, actionAuth, entityID, authType...) != nil {
			filtered[entityID] = false
		} else {
			filtered[entityID] = true
		}
	}

	result := make([]IEntityAuth, 0)
	for _, a := range authList {
		if filtered[a.GetEntityID()] {
			result = append(result, a)
		}
	}

	return result, nil
}

type IEntityStationAuth interface {
	GetStationID() int
}

func FilterEntityStationAuthInterface(siteID string, authList []IEntityStationAuth, actionAuth authority.ActionAuthSet, authType ...string) ([]IEntityStationAuth, error) {

	stationIDs := make([]int, 0)
	unique := make(map[int]byte)
	for _, a := range authList {
		if _, exists := unique[a.GetStationID()]; !exists {
			unique[a.GetStationID()] = 1
			stationIDs = append(stationIDs, a.GetStationID())
		}
	}

	filtered, err := FilterEntityStationAuth(siteID, actionAuth, stationIDs, authType...)
	if err != nil {
		return nil, err
	}

	result := make([]IEntityStationAuth, 0)
	for _, a := range authList {
		if filtered[a.GetStationID()] {
			result = append(result, a)
		}
	}

	return result, nil
}
