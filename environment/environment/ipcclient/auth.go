package ipcclient

// type _stationCheck int

// func (s _stationCheck) GetStationID() int { return int(s) }

// func filterStationID(siteID string, actionAuth authority.ActionAuthSet, stationID []int) ([]int, error) {

// 	result := make([]int, 0)

// 	filtered := make([]entity.IEntityStationAuth, 0)
// 	for _, id := range stationID {
// 		filtered = append(filtered, _stationCheck(id))
// 	}

// 	filtered, err := entity.FilterEntityStationAuth(siteID, filtered, actionAuth)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, stationID := range filtered {
// 		id := int(stationID.(_stationCheck))
// 		result = append(result, id)
// 	}
// 	return result, nil
// }
