package page

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
)

type ModelRelation struct {
	ID            int               `json:"ID"`
	Description   string            `json:"description"`
	ParentModelID string            `json:"parentModelID"`
	ChildModelID  string            `json:"childModelID"`
	Min           int               `json:"min"`
	Max           int               `json:"max"`
	SampleCount   int               `json:"sampleCount"`
	ParamSample   map[string]string `json:"paramSample"`
	Sort          int               `json:"sort"`
}

var e_need_param = errors.New("需要设定默认参数")

const modelRelationColumns = "modelrelation.id,modelrelation.description, modelrelation.parent_model_id, modelrelation.child_model_id,modelrelation.min,modelrelation.max,modelrelation.sample_count,modelrelation.param_sample,modelrelation.sort"

func modelRelationTable(siteID string) string { return siteID + "_pagemodelrelation" }

func (m *ModelRelation) scan(rows *sql.Rows) error {
	var param string
	if err := rows.Scan(&m.ID, &m.Description, &m.ParentModelID, &m.ChildModelID, &m.Min, &m.Max, &m.SampleCount, &param, &m.Sort); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(param), &m.ParamSample); err != nil {
		return err
	}
	return nil
}

func (m *ModelRelation) PreScan() ([]interface{}, []interface{}) {
	var param string
	return []interface{}{&m.ID, &m.Description, &m.ParentModelID, &m.ChildModelID, &m.Min, &m.Max, &m.SampleCount, &param, &m.Sort}, []interface{}{&param}
}

func (m *ModelRelation) PostScan(tmps []interface{}) error {
	param := tmps[0].(*string)
	if err := json.Unmarshal([]byte(*param), &m.ParamSample); err != nil {
		return err
	}
	return nil
}

func (m *ModelRelation) Add(siteID string, actionAuth authority.ActionAuthSet) error {
	if m.ParamSample == nil {
		return e_need_param
	}

	return datasource.Txn(func(txn *sql.Tx) {

		models, err := getModelsWithTxn(siteID, txn, true, m.ParentModelID, m.ChildModelID)
		if err != nil {
			panic(err)
		}

		if parent, exists := models[m.ParentModelID]; !exists {
			panic(fmt.Errorf("组件不存在【%s】", m.ParentModelID))
		} else if err := parent.CheckEditAuth(siteID, actionAuth); err != nil {
			panic(err)
		}

		if _, exists := models[m.ChildModelID]; !exists {
			panic(fmt.Errorf("组件不存在【%s】", m.ChildModelID))
		}

		param, _ := json.Marshal(m.ParamSample)

		if ret, err := txn.Exec(fmt.Sprintf(`
			INSERT INTO %s
				(description,parent_model_id,child_model_id,min,max,sample_count,param_sample,sort)
			VALUES
				(?,?,?,?,?,?,?,?)
		`, modelRelationTable(siteID)), m.Description, m.ParentModelID, m.ChildModelID, m.Min, m.Max, m.SampleCount, string(param), m.Sort); err != nil {
			log.Println("error insert model relation: ", err)
			panic(err)
		} else if id, err := ret.LastInsertId(); err != nil {
			log.Println("error insert model relation: ", err)
			panic(err)
		} else {
			m.ID = int(id)
		}
	})
}

func (m *ModelRelation) Update(siteID string, actionAuth authority.ActionAuthSet) error {

	if m.ParamSample == nil {
		return e_need_param
	}

	return datasource.Txn(func(txn *sql.Tx) {

		models, err := getModelsWithTxn(siteID, txn, true, m.ParentModelID, m.ChildModelID)
		if err != nil {
			panic(err)
		}

		if parent, exists := models[m.ParentModelID]; !exists {
			panic(fmt.Errorf("组件不存在【%s】", m.ParentModelID))
		} else if err := parent.CheckEditAuth(siteID, actionAuth); err != nil {
			panic(err)
		}

		if _, exists := models[m.ChildModelID]; !exists {
			panic(fmt.Errorf("组件不存在【%s】", m.ChildModelID))
		}

		param, _ := json.Marshal(m.ParamSample)

		if _, err := txn.Exec(fmt.Sprintf(`
			UPDATE
				%s
			SET
				description=?, min=?, max=?,sample_count=?,param_sample=?,sort=?
			WHERE
				id=?
		`, modelRelationTable(siteID)), m.Description, m.Min, m.Max, m.SampleCount, string(param), m.Sort, m.ID); err != nil {
			log.Println("error update model relation: ", err)
			panic(err)
		}
	})
}

func (m *ModelRelation) Delete(siteID string, actionAuth authority.ActionAuthSet) error {
	return datasource.Txn(func(txn *sql.Tx) {
		if err := m.delete(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func (m *ModelRelation) delete(siteID string, txn *sql.Tx) error {

	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, modelRelationTable(siteID)), m.ID); err != nil {
		log.Println("error delete model relation: ", err)
		return err
	}

	return nil
}

func GetModelRelations(siteID string, parentModelID, childModelID []string, modelRelationID ...int) ([]*ModelRelation, error) {
	return getModelRelations(siteID, nil, false, parentModelID, childModelID, modelRelationID...)
}

func getModelRelations(siteID string, txn *sql.Tx, forUpdate bool, parentModelID, childModelID []string, modelRelationID ...int) ([]*ModelRelation, error) {

	result := make([]*ModelRelation, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(parentModelID) > 0 {
		if len(parentModelID) == 0 {
			whereStmts = append(whereStmts, "modelrelation.parent_model_id = ?")
			values = append(values, parentModelID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range parentModelID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("modelrelation.parent_model_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(childModelID) > 0 {
		if len(childModelID) == 0 {
			whereStmts = append(whereStmts, "modelrelation.child_model_id = ?")
			values = append(values, childModelID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range childModelID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("modelrelation.child_model_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(modelRelationID) > 0 {
		if len(modelRelationID) == 0 {
			whereStmts = append(whereStmts, "modelrelation.id ")
			values = append(values, modelRelationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range modelRelationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("modelrelation.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(whereStmts) == 0 {
		return result, nil
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s modelrelation
		WHERE
			%s
	`, modelRelationColumns, modelRelationTable(siteID), strings.Join(whereStmts, " AND "))

	log.Println("SQL: ", SQL, values)

	var rows *sql.Rows
	var err error

	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	}
	if err != nil {
		log.Println("error get model relations: ", SQL, values, err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var mr ModelRelation
		if err := mr.scan(rows); err != nil {
			log.Println("error scan model relations: ", err)
			return nil, err
		}
		result = append(result, &mr)
	}

	return result, nil
}

func GetChildModels(siteID string, parentModelID string, cids []int, modelType, moduleID, q string, pageNo, pageSize int) ([]*ModelRelation, map[string]*Model, int, error) {

	modelRelationList := make([]*ModelRelation, 0)
	models := make(map[string]*Model)
	total := 0

	if parentModelID == "" {
		return modelRelationList, models, total, nil
	}

	joinModel, _, joinWhere, joinValues, err := JoinSiteModel(siteID, "modelrelation", "child_model_id", cids, modelType, moduleID, q)
	if err != nil {
		return nil, nil, 0, err
	}
	if joinModel == "" {
		return modelRelationList, models, total, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	whereStmts = append(whereStmts, joinWhere...)
	values = append(values, joinValues...)

	whereStmts = append(whereStmts, "modelrelation.parent_model_id = ?")
	values = append(values, parentModelID)

	SQL := fmt.Sprintf(`
		SELECT
			%s, %s
		FROM
			%s modelrelation
		%s
	`, modelColumns, modelRelationColumns, modelRelationTable(siteID), joinModel)

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s modelrelation
		%s
	`, modelRelationTable(siteID), joinModel)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count child models: ", err)
		return nil, nil, 0, err
	}

	SQL += "\nORDER BY modelrelation.sort DESC"

	if pageSize != -1 {
		if pageNo <= 0 {
			pageNo = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		SQL += "\nLIMIT ?,?"
		values = append(values, (pageNo-1)*pageSize, pageSize)
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get child models: ", err)
		return nil, nil, 0, err
	}

	defer rows.Close()

	for rows.Next() {
		var model Model
		var mr ModelRelation

		prescan, postscan := mr.PreScan()
		if err := model.scan(rows, prescan...); err != nil {
			log.Println("error scan child models: ", err)
			return nil, nil, 0, err
		}

		if err := mr.PostScan(postscan); err != nil {
			log.Println("error post scan: ", err)
			return nil, nil, 0, err
		}

		modelRelationList = append(modelRelationList, &mr)

		if _, exists := models[model.ID]; !exists {
			models[model.ID] = &model
		}
	}

	return modelRelationList, models, total, nil
}
