package site

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site/initialization"
)

const (
	MODULE_SITE            = "site"
	ACTION_C_EDIT_SITE     = "c_edit"
	ACTION_ADMIN_EDIT_SITE = "admin_edit"
)

const (
	MODULE_MODULE = "module"

	ACTION_ADMIN_VIEW_MODULE = "admin_view"
	ACTION_ADMIN_EDIT_MODULE = "admin_edit"

	ACTION_C_VIEW_MODULE = "c_view"
	ACTION_C_EDIT_MODULE = "c_edit"
)

func init() {
	initialization.Register("module", []string{"module", "model"})
}

type Module struct {
	ModuleID    string                 `json:"ID"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Prerequsite map[string]interface{} `json:"prerequisite"`
	Action      []*ModuleAction        `json:"action"`
}

type ModuleAction struct {
	Action      string `json:"action"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const ModuleColumns = "module.id, module.name, module.description, module.prerequisite, module.action"
const ModuleTableName = "c_module"

func (m *Module) Scan(rows *sql.Rows, appendix ...interface{}) error {
	var pre, action string

	dest := make([]interface{}, 0)
	dest = append(dest, &m.ModuleID, &m.Name, &m.Description, &pre, &action)
	dest = append(dest, appendix...)

	if err := rows.Scan(dest...); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(pre), &m.Prerequsite); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(action), &m.Action); err != nil {
		return err
	}
	return nil
}

func (m *Module) Add() error {
	if m.Prerequsite == nil {
		m.Prerequsite = make(map[string]interface{})
	}
	if m.Action == nil {
		m.Action = make([]*ModuleAction, 0)
	}
	pre, _ := json.Marshal(m.Prerequsite)
	action, _ := json.Marshal(m.Action)
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(id, name, description, prerequisite, action)
		VALUES
			(?,?,?,?,?)
	`, ModuleTableName), m.ModuleID, m.Name, m.Description, []byte(pre), []byte(action)); err != nil {
		return err
	}
	return nil
}

func (m *Module) Update() error {
	if m.Prerequsite == nil {
		m.Prerequsite = make(map[string]interface{})
	}
	if m.Action == nil {
		m.Action = make([]*ModuleAction, 0)
	}
	pre, _ := json.Marshal(m.Prerequsite)
	action, _ := json.Marshal(m.Action)
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE 
			%s
		SET
			name=?, description=?, prerequisite=?, action=?
		WHERE
			id=?
	`, ModuleTableName), m.Name, m.Description, []byte(pre), []byte(action), m.ModuleID); err != nil {
		return err
	}
	return nil
}

func (m *Module) Delete() error {
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM 
			%s
		WHERE
			id=?
	`, ModuleTableName), m.ModuleID); err != nil {
		return err
	}
	return nil
}

func GetModule(moduleID string) (*Module, error) {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s module
		WHERE
			id = ?
	`, ModuleColumns, ModuleTableName), moduleID)
	if err != nil {
		log.Println("error get modules: ", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var m Module
		if err := m.Scan(rows); err != nil {
			return nil, err
		}

		return &m, nil
	}

	return nil, errors.New("模块不存在")
}

func GetSiteModuleList(siteID string, moduleID ...string) ([]*Module, error) {
	result := make([]*Module, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as module
	`, ModuleColumns, ModuleTableName)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	joinAuth, joinWhere, joinValues := JoinSiteModuleAuth(siteID, "module", "id")
	SQL += joinAuth

	whereStmts = append(whereStmts, joinWhere...)
	values = append(values, joinValues...)

	if len(moduleID) > 0 {
		placeHolder := make([]string, 0)
		for _, id := range moduleID {
			if id != "" {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
		}
		if len(placeHolder) > 0 {
			whereStmts = append(whereStmts, fmt.Sprintf("module.id IN (%s)", strings.Join(placeHolder, ",")))
		}
	}

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get modules: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Module
		if err := m.Scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &m)
	}

	return result, nil
}
