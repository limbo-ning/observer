package user

import (
	"fmt"

	"obsessiontech/environment/authority"
)

func init() {
	authority.RegisterEmpower("user", func() authority.IEmpower {
		return new(UserEmpower)
	})
}

type UserEmpower struct{}

func (e *UserEmpower) EmpowerID(siteID string, actionAuth authority.ActionAuthSet) ([]string, error) {
	return []string{fmt.Sprintf("%d", actionAuth[0].UID)}, nil
}
