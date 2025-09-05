package clientAgent

import (
	"obsessiontech/environment/site"
)

type ClientAgent struct {
	ID          string `json:"ID"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func GetClientAgentList(siteID string) ([]*ClientAgent, error) {
	smList, err := site.GetSiteModules(siteID, "", "clientagent_")
	if err != nil {
		return nil, err
	}

	result := make([]*ClientAgent, 0)

	for _, sm := range smList {

		var c ClientAgent
		c.ID = sm.ModuleID
		if name, exists := sm.Param["name"]; exists {
			c.Name = name.(string)
		}
		if description, exists := sm.Param["description"]; exists {
			c.Description = description.(string)
		}

		result = append(result, &c)
	}

	return result, nil
}
