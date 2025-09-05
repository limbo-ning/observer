package role

import (
	"fmt"

	"obsessiontech/environment/authority"
)

func init() {
	authority.RegisterEmpower("role", func() authority.IEmpower {
		return new(RoleEmpower)
	})
	authority.RegisterEmpower("roleSeries", func() authority.IEmpower {
		return new(RoleSeriesEmpower)
	})
}

type RoleEmpower struct{}

func (e *RoleEmpower) EmpowerID(siteID string, actionAuth authority.ActionAuthSet) ([]string, error) {

	roleIDs := make(map[int]byte)
	result := make([]string, 0)
	for _, a := range actionAuth {
		if _, exists := roleIDs[a.RoleID]; !exists {
			roleIDs[a.RoleID] = 1
			result = append(result, fmt.Sprintf("%d", a.RoleID))
		}
	}

	return result, nil
}

type RoleSeriesEmpower struct{}

func (e *RoleSeriesEmpower) EmpowerID(siteID string, actionAuth authority.ActionAuthSet) ([]string, error) {

	roleSeries := make(map[string]byte)
	result := make([]string, 0)
	for _, a := range actionAuth {
		if _, exists := roleSeries[a.RoleSeries]; !exists {
			roleSeries[a.RoleSeries] = 1
			result = append(result, a.RoleSeries)
		}
	}

	return result, nil
}
