package page

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/category"
	"obsessiontech/environment/site"
	"obsessiontech/environment/site/page/template"
)

const (
	MODEL_ROOT   = "ROOT"
	MODEL_NORMAL = "NORMAL"
)

type Model struct {
	template.BaseModel
	ModuleID         string `json:"moduleID"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Description      string `json:"description"`
	HTMLTemplatePath string `json:"HTMLTemplatePath"`
	CSSTemplatePath  string `json:"CSSTemplatePath"`
	JSTemplatePath   string `json:"JSTemplatePath"`
}

const modelColumns = "model.id, model.module_id, model.name, model.type, model.description, model.html_template_path, model.css_template_path, model.js_template_path, model.param"

func modelTable(siteID string) string {
	return siteID + "_pagemodel"
}

var e_model_not_exists = errors.New("模版不存在")
var e_template_file_exists = errors.New("模版文件路径已存在")

func (m *Model) loadTemplate(siteID string) error {

	pathPrefix := Config.PageTemplateFolderPath
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix += "/"
	}
	pathPrefix += siteID

	if m.HTMLTemplatePath != "" && m.HTMLTemplate == "" {
		path := pathPrefix
		if !strings.HasPrefix(m.HTMLTemplatePath, "/") {
			path += "/"
		}
		if d, err := ioutil.ReadFile(path + m.HTMLTemplatePath); err != nil {
			return err
		} else {
			m.HTMLTemplate = string(d)
		}
	}
	if m.CSSTemplatePath != "" && m.CSSTemplate == "" {
		path := pathPrefix
		if !strings.HasPrefix(m.HTMLTemplatePath, "/") {
			path += "/"
		}
		if d, err := ioutil.ReadFile(path + m.CSSTemplatePath); err != nil {
			return err
		} else {
			m.CSSTemplate = string(d)
		}
	}
	if m.JSTemplatePath != "" && m.JSTemplate == "" {
		path := pathPrefix
		if !strings.HasPrefix(m.HTMLTemplatePath, "/") {
			path += "/"
		}
		if d, err := ioutil.ReadFile(path + m.JSTemplatePath); err != nil {
			return err
		} else {
			m.JSTemplate = string(d)
		}
	}
	return nil
}

func (m *Model) scan(rows *sql.Rows, appendix ...interface{}) error {

	var paramStr string
	dest := make([]interface{}, 0)
	dest = append(dest, &m.ID, &m.ModuleID, &m.Name, &m.Type, &m.Description, &m.HTMLTemplatePath, &m.CSSTemplatePath, &m.JSTemplatePath, &paramStr)
	dest = append(dest, appendix...)

	if err := rows.Scan(dest...); err != nil {
		log.Println("error scan module model: ", err)
		return err
	}
	return json.Unmarshal([]byte(paramStr), &m.Param)
}

func (m *Model) CheckEditAuth(siteID string, actionAuth authority.ActionAuthSet) error {
	for _, a := range actionAuth {
		action := strings.TrimPrefix(a.Action, MODULE_MODEL+"#")

		switch action {
		case ACTION_ADMIN_EDIT_MODEL:
			return nil
		default:
			continue
		}
	}

	return errors.New("无权限")
}

func (m *Model) Add(siteID string) error {
	return m.add(siteID, nil)
}

func (m *Model) add(siteID string, txn *sql.Tx) error {

	if m.ID == "" {
		return errors.New("need ID")
	}

	if m.ModuleID == "" {
		return errors.New("need module ID")
	}

	switch m.Type {
	case MODEL_ROOT:
	case MODEL_NORMAL:
	default:
		m.Type = MODEL_NORMAL
	}

	if m.Param == nil {
		m.Param = make(map[string]interface{})
	}
	paramBytes, _ := json.Marshal(m.Param)

	var err error

	SQL := fmt.Sprintf(`
		INSERT INTO %s
			(id, module_id, name, type, description, html_template_path, css_template_path, js_template_path, param)
		VALUES
			(?,?,?,?,?,?,?,?,?)
	`, modelTable(siteID))

	values := []interface{}{m.ID, m.ModuleID, m.Name, m.Type, m.Description, m.HTMLTemplatePath, m.CSSTemplatePath, m.JSTemplatePath, string(paramBytes)}

	if txn == nil {
		_, err = datasource.GetConn().Exec(SQL, values...)
	} else {
		_, err = txn.Exec(SQL, values...)
	}

	if err != nil {
		log.Println("error insert module model: ", err)
		return err
	}

	return nil
}

func (m *Model) AddAssembleModel(siteID string, actionAuth authority.ActionAuthSet, componentList []*template.BaseComponent, paramAlias map[string]map[string]string) error {

	if err := m.CheckEditAuth(siteID, actionAuth); err != nil {
		return err
	}

	folder := m.ID

	toAssemble := make([]template.IComponent, 0)
	for _, comp := range componentList {
		toAssemble = append(toAssemble, comp)
	}

	if err := fillModels(siteID, toAssemble); err != nil {
		return err
	}

	assembled, err := template.AssembleTemplate(toAssemble, paramAlias)
	if err != nil {
		log.Println("error assemble model: ", err)
		return err
	}

	if assembled.GetHTML() != "" {
		m.HTMLTemplatePath = fmt.Sprintf("%s/html.tmpl", folder)
	} else {
		m.HTMLTemplatePath = ""
	}
	if assembled.GetCSS() != "" {
		m.CSSTemplatePath = fmt.Sprintf("%s/css.tmpl", folder)
	} else {
		m.CSSTemplatePath = ""
	}
	if assembled.GetJS() != "" {
		m.JSTemplatePath = fmt.Sprintf("%s/js.tmpl", folder)
	} else {
		m.JSTemplatePath = ""
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := m.add(siteID, txn); err != nil {
			log.Println("error insert module model: ", err)
			panic(err)
		}

		folder := fmt.Sprintf("%s/%s/%s/", Config.PageTemplateFolderPath, siteID, folder)

		log.Println("folder: ", folder)

		if assembled.GetHTML() != "" {
			if err := writeFile([]byte(assembled.GetHTML()), folder, "html.tmpl"); err != nil {
				log.Println("error write html: ", err)
				panic(err)
			}
		}
		if assembled.GetCSS() != "" {
			if err := writeFile([]byte(assembled.GetCSS()), folder, "css.tmpl"); err != nil {
				log.Println("error write css: ", err)
				panic(err)
			}
		}
		if assembled.GetJS() != "" {
			if err := writeFile([]byte(assembled.GetJS()), folder, "js.tmpl"); err != nil {
				log.Println("error write js: ", err)
				panic(err)
			}
		}
	})
}

func (m *Model) Update(siteID string) error {
	return m.update(siteID, nil)
}

func (m *Model) update(siteID string, txn *sql.Tx) error {

	if m.ID == "" {
		return errors.New("need ID")
	}

	if m.ModuleID == "" {
		return errors.New("need module ID")
	}

	switch m.Type {
	case MODEL_ROOT:
	case MODEL_NORMAL:
	default:
		m.Type = MODEL_NORMAL
	}

	if m.Param == nil {
		m.Param = make(map[string]interface{})
	}
	param, _ := json.Marshal(m.Param)

	SQL := fmt.Sprintf(`
		UPDATE 
			%s
		SET
			module_id=?, name=?, type=?, description=?, param=?, html_template_path=?, css_template_path=?, js_template_path=?
		WHERE
			id = ?
	`, modelTable(siteID))

	values := []interface{}{m.ModuleID, m.Name, m.Type, m.Description, string(param), m.HTMLTemplatePath, m.CSSTemplatePath, m.JSTemplatePath, m.ID}

	var err error
	if txn == nil {
		_, err = datasource.GetConn().Exec(SQL, values...)
	} else {
		_, err = txn.Exec(SQL, values...)
	}

	if err != nil {
		log.Println("error update module model: ", err)
		return err
	}

	return nil
}

func (m *Model) UpdateAssembleModel(siteID string, actionAuth authority.ActionAuthSet, componentList []*template.BaseComponent, paramAlias map[string]map[string]string) error {

	if err := m.CheckEditAuth(siteID, actionAuth); err != nil {
		return err
	}

	folder := m.ID
	var assembled template.IComponent

	if componentList != nil && len(componentList) > 0 {
		toAssemble := make([]template.IComponent, 0)
		for _, comp := range componentList {
			toAssemble = append(toAssemble, comp)
		}

		err := fillModels(siteID, toAssemble)
		if err != nil {
			return err
		}

		assembled, err = template.AssembleTemplate(toAssemble, paramAlias)
		if err != nil {
			log.Println("error assemble model: ", err)
			return err
		}
		if assembled.GetHTML() != "" {
			m.HTMLTemplatePath = fmt.Sprintf("%s/html.tmpl", folder)
		} else {
			m.HTMLTemplatePath = ""
		}
		if assembled.GetCSS() != "" {
			m.CSSTemplatePath = fmt.Sprintf("%s/css.tmpl", folder)
		} else {
			m.CSSTemplatePath = ""
		}
		if assembled.GetJS() != "" {
			m.JSTemplatePath = fmt.Sprintf("%s/js.tmpl", folder)
		} else {
			m.JSTemplatePath = ""
		}

	}

	return datasource.Txn(func(txn *sql.Tx) {

		if err := m.update(siteID, txn); err != nil {
			panic(err)
		}

		folder := fmt.Sprintf("%s/%s/%s/", Config.PageTemplateFolderPath, siteID, folder)

		if assembled != nil {
			if assembled.GetHTML() != "" {
				if err := writeFile([]byte(assembled.GetHTML()), folder, "html.tmpl"); err != nil {
					panic(err)
				}
				m.HTMLTemplatePath = fmt.Sprintf("%s/html.tmpl", folder)
			}
			if assembled.GetCSS() != "" {
				if err := writeFile([]byte(assembled.GetCSS()), folder, "css.tmpl"); err != nil {
					panic(err)
				}
			}
			if assembled.GetJS() != "" {
				if err := writeFile([]byte(assembled.GetJS()), folder, "js.tmpl"); err != nil {
					panic(err)
				}
			}
		}

	})
}

func (m *Model) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	return datasource.Txn(func(txn *sql.Tx) {

		models, err := getModelsWithTxn(siteID, txn, true, m.ID)
		if err != nil {
			panic(err)
		}

		var exists bool
		m, exists = models[m.ID]
		if !exists {
			panic(e_model_not_exists)
		}

		if err := m.CheckEditAuth(siteID, actionAuth); err != nil {
			panic(err)
		}

		childs, err := getModelRelations(siteID, txn, true, []string{m.ID}, nil)
		if err != nil {
			panic(err)
		}

		if len(childs) > 0 {
			panic(errors.New("组件仍有关联子组件"))
		}

		if _, err := txn.Exec(fmt.Sprintf(`
			DELETE FROM
				%s
			WHERE
				id = ?
		`, modelTable(siteID)), m.ID); err != nil {
			log.Println("error delete module model: ", err)
			panic(err)
		}

		parents, err := getModelRelations(siteID, txn, true, nil, []string{m.ID})
		if err != nil {
			panic(err)
		}

		for _, p := range parents {
			if err := p.delete(siteID, txn); err != nil {
				panic(err)
			}
		}

	})
}

// func GetModels(siteID string, modelIDs ...string) (map[string]*Model, error) {
// 	return GetSiteModels(siteID, modelIDs...)
// }

func getModelsWithTxn(siteID string, txn *sql.Tx, forUpdate bool, modelIDs ...string) (map[string]*Model, error) {

	models := make(map[string]*Model)
	if len(modelIDs) == 0 {
		return models, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	placeHolders := make([]string, len(modelIDs))
	for i, ID := range modelIDs {
		placeHolders[i] = "?"
		values = append(values, ID)
	}

	whereStmts = append(whereStmts, fmt.Sprintf("model.id IN (%s)", strings.Join(placeHolders, ",")))

	sql := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as model
		WHERE
			%s
	`, modelColumns, modelTable(siteID), strings.Join(whereStmts, " AND "))

	if forUpdate {
		sql += "\nFOR UPDATE"
	}

	rows, err := txn.Query(sql, values...)
	if err != nil {
		log.Println("error get models: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Model
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		if err := m.loadTemplate(siteID); err != nil {
			log.Println("error load template: ", err)
			return nil, err
		}
		m.HTMLTemplatePath = ""
		m.CSSTemplatePath = ""
		m.JSTemplatePath = ""

		models[m.ID] = &m
	}

	return models, nil
}

func GetSiteModels(siteID string, modelIDs ...string) (map[string]*Model, error) {

	models := make(map[string]*Model)
	if len(modelIDs) == 0 {
		return models, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	var join string

	if siteID != "" {
		joinAuth, joinWhere, joinValues := site.JoinSiteModuleAuth(siteID, "model", "module_id")
		join += joinAuth

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	placeHolders := make([]string, len(modelIDs))
	for i, ID := range modelIDs {
		placeHolders[i] = "?"
		values = append(values, ID)
	}

	whereStmts = append(whereStmts, fmt.Sprintf("model.id IN (%s)", strings.Join(placeHolders, ",")))

	sql := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as model
		%s
		WHERE
			%s
	`, modelColumns, modelTable(siteID), join, strings.Join(whereStmts, " AND "))

	rows, err := datasource.GetConn().Query(sql, values...)
	if err != nil {
		log.Println("error get models: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Model
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		if err := m.loadTemplate(siteID); err != nil {
			log.Println("error load template: ", err)
			return nil, err
		}
		m.HTMLTemplatePath = ""
		m.CSSTemplatePath = ""
		m.JSTemplatePath = ""

		models[m.ID] = &m
	}

	return models, nil
}

func JoinSiteModel(siteID, table, column string, cids []int, modelType, moduleID, q string) (string, string, []string, []interface{}, error) {
	var join string

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if siteID != "" {
		joinAuth, joinWhere, joinValues := site.JoinSiteModuleAuth(siteID, "model", "module_id")
		join += joinAuth

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if cids != nil && len(cids) > 0 {
		for _, cid := range cids {
			joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "model", "", cid)
			if err != nil {
				return "", "", nil, nil, err
			}
			if joinSQL == "" {
				return "", "", whereStmts, values, nil
			}
			join += joinSQL

			whereStmts = append(whereStmts, joinWhere...)
			values = append(values, joinValues...)
		}
	}

	if moduleID != "" {
		whereStmts = append(whereStmts, "model.module_id = ?")
		values = append(values, moduleID)
	}
	if modelType != "" {
		whereStmts = append(whereStmts, "model.type = ?")
		values = append(values, modelType)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(model.id LIKE ? OR model.name LIKE ? OR model.description LIKE ?)")
		values = append(values, qq, qq, qq)
	}

	SQL := fmt.Sprintf(`
		JOIN
			%s model
		ON
			model.id = %s.%s
		%s
	`, modelTable(siteID), table, column, join)

	return SQL, "model", whereStmts, values, nil
}

func GetSiteModelList(siteID string, cids []int, modelType, moduleID, q string, pageNo, pageSize int, modelID ...string) ([]*Model, int, error) {
	models := make([]*Model, 0)
	var total int

	var join string
	column := modelColumns

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if siteID != "" {
		joinAuth, joinWhere, joinValues := site.JoinSiteModuleAuth(siteID, "model", "module_id")
		join += joinAuth

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if len(cids) > 0 {
		for _, cid := range cids {
			joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "model", "", cid)
			if err != nil {
				return nil, 0, err
			}
			if joinSQL == "" {
				return models, total, nil
			}
			join += joinSQL

			whereStmts = append(whereStmts, joinWhere...)
			values = append(values, joinValues...)
		}
	}

	if len(modelID) > 0 {
		if len(modelID) == 1 {
			whereStmts = append(whereStmts, "model.id = ?")
			values = append(values, modelID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range modelID {
				placeholder = append(placeholder, ",")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("model.id IN (%s)", strings.Join(placeholder, ",")))
		}
		pageSize = -1
	}

	if moduleID != "" {
		whereStmts = append(whereStmts, "model.module_id IN (?,?)")
		values = append(values, "*", moduleID)
	}
	if modelType != "" {
		whereStmts = append(whereStmts, "model.type = ?")
		values = append(values, modelType)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(model.id LIKE ? OR model.name LIKE ? OR model.description LIKE ?)")
		values = append(values, qq, qq, qq)
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as model
		%s
	`, column, modelTable(siteID), join)

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s as model
		%s
	`, modelTable(siteID), join)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count models: ", countSQL, values, err)
		return nil, 0, err
	}

	SQL += "\nORDER BY model.id DESC"
	if pageSize != -1 {
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageNo <= 0 {
			pageNo = 0
		} else {
			pageNo = (pageNo - 1) * pageSize
		}
		values = append(values, pageNo, pageSize)

		SQL += "\nLIMIT ?,?"
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get models: ", SQL, values, err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Model
		if err := m.scan(rows); err != nil {
			return nil, 0, err
		}
		models = append(models, &m)
	}

	return models, total, nil
}

func AddModelCategory(modelID string, cid int) error {
	return category.AddCategoryMapping("c", "model", modelID, cid)
}

func DeleteModelCategory(modelID string, cid int) error {
	return category.DeleteCategoryMapping("c", "model", modelID, cid)
}
