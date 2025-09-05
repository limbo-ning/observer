package page

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/site/initialization"
	"obsessiontech/environment/site/page/template"
)

func init() {
	initialization.Register("site_page", []string{"pagecomponent"})
}

const STATUS_ARCHIVED = "ARCHIVED"
const STATUS_PUBLISHED = "PUBLISHED"
const STATUS_EDIT = "EDIT"

type PageComponent struct {
	template.BaseComponent
	ID         int    `json:"-"`
	PageID     string `json:"pageID"`
	Status     string `json:"status"`
	CreateTime *util.Time
	UpdateTime *util.Time
}

const pageComponentColumn = "pagecomponent.id, pagecomponent.page_id, pagecomponent.component_id, pagecomponent.model_id, pagecomponent.model_relation_id, pagecomponent.parent_component_id, pagecomponent.no, pagecomponent.param, pagecomponent.status"

func pageComponentTable(siteID string) string {
	return siteID + "_pagecomponent"
}

func (c *PageComponent) scan(rows *sql.Rows) error {
	var paramStr string
	if err := rows.Scan(&c.ID, &c.PageID, &c.ComponentID, &c.ModelID, &c.ModelRelationID, &c.ParentComponentID, &c.No, &paramStr, &c.Status); err != nil {
		log.Println("error scan page componet: ", err)
		return err
	}

	if err := json.Unmarshal([]byte(paramStr), &c.Param); err != nil {
		log.Println("error unmarshal param: ", err)
	}
	return nil
}

func GetPageComponents(siteID, pageID, status string) ([]template.IComponent, map[string]*Model, error) {
	list, err := getPageComponents(siteID, nil, false, pageID, status)
	if err != nil {
		return nil, nil, err
	}

	modelIDs := make([]string, len(list))
	for i, pc := range list {
		modelIDs[i] = pc.(*PageComponent).ModelID
	}

	models, err := GetSiteModels(siteID, modelIDs...)
	if err != nil {
		return nil, nil, err
	}

	return list, models, nil
}

func GetPages(siteID, status string) ([]template.IComponent, error) {
	return getPages(siteID, nil, false, status)
}

func getPages(siteID string, txn *sql.Tx, forUpdate bool, status string) ([]template.IComponent, error) {
	result := make([]template.IComponent, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if status != "" {
		whereStmts = append(whereStmts, "pagecomponent.status = ?")
		values = append(values, status)
	} else {
		return nil, errors.New("需要指定状态")
	}

	whereStmts = append(whereStmts, "pagecomponent.parent_component_id = ?")
	values = append(values, "")

	var rows *sql.Rows
	var err error

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s pagecomponent
	`, pageComponentColumn, pageComponentTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	}
	if err != nil {
		log.Println("error get page: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c PageComponent
		if err := c.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &c)
	}

	return result, nil
}

func getPageComponents(siteID string, txn *sql.Tx, forUpdate bool, pageID, status string) ([]template.IComponent, error) {
	result := make([]template.IComponent, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if pageID != "" {
		whereStmts = append(whereStmts, "pagecomponent.page_id = ?")
		values = append(values, pageID)
	}
	if status != "" {
		whereStmts = append(whereStmts, "pagecomponent.status = ?")
		values = append(values, status)
	} else {
		return nil, errors.New("需要指定状态")
	}

	var rows *sql.Rows
	var err error

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s pagecomponent
	`, pageComponentColumn, pageComponentTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	}

	if err != nil {
		log.Println("error get page component: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c PageComponent
		if err := c.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &c)
	}

	return result, nil
}

func (comp *PageComponent) insert(siteID string, txn *sql.Tx) error {
	if comp.Param == nil {
		comp.Param = make(map[string]string)
	}
	paramBytes, _ := json.Marshal(comp.Param)

	if _, err := txn.Exec(`
		INSERT INTO `+pageComponentTable(siteID)+`
			(page_id, component_id, model_id, model_relation_id, parent_component_id, no, param, status)
		VALUES
			(?,?,?,?,?,?,?,?)
	`, comp.PageID, comp.ComponentID, comp.ModelID, comp.ModelRelationID, comp.ParentComponentID, comp.No, string(paramBytes), comp.Status); err != nil {
		log.Println("error insert page component: ", err)
		return err
	}
	return nil
}

func (comp *PageComponent) update(siteID string, txn *sql.Tx) error {
	if comp.Param == nil {
		comp.Param = make(map[string]string)
	}
	paramBytes, _ := json.Marshal(comp.Param)

	if _, err := txn.Exec(`
		UPDATE 
			`+pageComponentTable(siteID)+`
		SET
			page_id = ?, component_id = ?, model_id = ?, model_relation_id = ?, parent_component_id = ?, no = ?, param = ?, status = ?
		WHERE
			id = ?
	`, comp.PageID, comp.ComponentID, comp.ModelID, comp.ModelRelationID, comp.ParentComponentID, comp.No, string(paramBytes), comp.Status, comp.ID); err != nil {
		log.Println("error update page component: ", err)
		return err
	}
	return nil
}

func (comp *PageComponent) delete(siteID string, txn *sql.Tx) error {
	if _, err := txn.Exec(`
		DELETE FROM 
			`+pageComponentTable(siteID)+`
		WHERE
			id = ?
	`, comp.ID); err != nil {
		log.Println("error remove page component: ", err)
		return err
	}
	return nil
}
