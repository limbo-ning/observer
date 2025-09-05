package page

import (
	"database/sql"
	"errors"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site/page/template"
)

func SavePageComponents(siteID, pageID string, components []*PageComponent) (e error) {

	if pageID == "" {
		return errors.New("pageID不能为空")
	}

	iComps := make([]template.IComponent, 0)
	for _, c := range components {
		iComps = append(iComps, c)
	}
	if err := fillModels(siteID, iComps); err != nil {
		return err
	}
	_, err := template.ParseTemplate(iComps)
	if err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {

		existsComps, err := getPageComponents(siteID, txn, true, pageID, STATUS_EDIT)
		if err != nil {
			panic(err)
		}

		if len(existsComps) == 0 && len(components) == 0 {
			return
		}

		if len(existsComps) != 0 && len(components) == 0 {
			deleted := new(PageComponent)
			deleted.ComponentID = "deleted"
			deleted.PageID = pageID
			components = append(components, deleted)
		}

		toRemove := make(map[string]*PageComponent)
		toUpdate := make(map[string]*PageComponent)
		toInsert := make(map[string]*PageComponent)

		for _, comp := range existsComps {
			toRemove[comp.GetID()] = comp.(*PageComponent)
		}

		for _, toSave := range components {
			toSave.PageID = pageID
			toSave.Status = STATUS_EDIT

			if toSave.Param == nil {
				toSave.Param = make(map[string]string)
			}
			toSave.Param["ID"] = toSave.GetID()
			toSave.Param["parentID"] = toSave.GetParentID()

			if origin, exists := toRemove[toSave.GetID()]; exists {
				toSave.ID = origin.ID
				toUpdate[toSave.GetID()] = toSave
				delete(toRemove, toSave.GetID())
			} else {
				toInsert[toSave.GetID()] = toSave
			}
		}

		for _, comp := range toRemove {
			if err := comp.delete(siteID, txn); err != nil {
				panic(err)
			}
		}

		for _, comp := range toUpdate {
			if err := comp.update(siteID, txn); err != nil {
				panic(err)
			}
		}

		for _, comp := range toInsert {
			if err := comp.insert(siteID, txn); err != nil {
				panic(err)
			}
		}
	})
}
