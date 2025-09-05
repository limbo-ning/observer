package category

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/category/module"
	"obsessiontech/environment/relation"
)

type Category struct {
	ID               int                 `json:"ID"`
	Source           string              `json:"source"`
	Type             string              `json:"type"`
	ParentCategoryID int                 `json:"parentID"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Profile          map[string][]string `json:"profile"`
	Sort             int                 `json:"sort"`
	Total            int                 `json:"total"`
}

const categogyColumn = "category.id, category.source, category.parent_id, category.type, category.name, category.description, category.profile,category.sort, category.total"

func (c *Category) scan(rows *sql.Rows, appendix ...interface{}) error {
	var profile string

	dest := make([]interface{}, 0)
	dest = append(dest, &c.ID, &c.Source, &c.ParentCategoryID, &c.Type, &c.Name, &c.Description, &profile, &c.Sort, &c.Total)
	dest = append(dest, appendix...)

	if err := rows.Scan(dest...); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(profile), &c.Profile); err != nil {
		return err
	}

	return nil
}

func categoryTableName(siteID string) string {
	return siteID + "_category"
}

func GetCategories(siteID, source, clientAgent string, actionAuth authority.ActionAuthSet, categoryType ...string) (map[string]map[string][]*Category, error) {

	categories := make(map[string]map[string][]*Category)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if source != "" {
		whereStmts = append(whereStmts, "category.source = ?")
		values = append(values, source)
	}

	if len(categoryType) == 1 {
		whereStmts = append(whereStmts, "category.type = ?")
		values = append(values, categoryType[0])
	} else if len(categoryType) > 1 {
		placeholder := make([]string, 0)
		for _, t := range categoryType {
			placeholder = append(placeholder, "?")
			values = append(values, t)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("category.type IN (%s)", strings.Join(placeholder, ",")))
	}

	sql := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as category
	`, categogyColumn, categoryTableName(siteID))

	if len(whereStmts) > 0 {
		sql += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	sql += "\n ORDER BY category.sort DESC, category.id DESC"

	rows, err := datasource.GetConn().Query(sql, values...)
	if err != nil {
		log.Println("error get category: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Category
		if err := c.scan(rows); err != nil {
			log.Println("error scan category: ", err)
		}
		if _, exists := categories[c.Source]; !exists {
			categories[c.Source] = make(map[string][]*Category, 0)
		}
		if _, exists := categories[c.Source][c.Type]; !exists {
			categories[c.Source][c.Type] = make([]*Category, 0)
		}
		categories[c.Source][c.Type] = append(categories[c.Source][c.Type], &c)
	}

	return categories, nil
}

func GetObjectCategories(siteID, source string, objectIDs []string, actionAuth authority.ActionAuthSet) (map[string][]*Category, error) {

	result := make(map[string][]*Category)

	if source == "" {
		return result, nil
	}

	if len(objectIDs) == 0 {
		return result, nil
	}

	pat, err := regexp.Compile("^[0-9a-zA-Z]+$")
	if err != nil {
		return nil, err
	}
	for _, id := range objectIDs {
		if !pat.Match([]byte(id)) {
			return result, nil
		}
	}

	joinSQL, joinTable, whereStmts, values, err := relation.JoinSQL(siteID, source, "category", "", "category", objectIDs...)
	if err != nil {
		return nil, err
	}

	if joinSQL == "" {
		return result, nil
	}

	var where string
	if len(whereStmts) > 0 {
		where = "WHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s, %s.a
		FROM
			%s category
		%s
		%s
		ORDER BY %s.a DESC, category.sort DESC, category.id DESC
	`, categogyColumn, joinTable, categoryTableName(siteID), joinSQL, where, joinTable)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get object categories: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	list := make([]*Category, 0)
	fetched := make(map[*Category]string)

	for rows.Next() {
		var c Category
		var ID string

		if err := c.scan(rows, &ID); err != nil {
			log.Println("error scan object categories: ", err)
		}

		fetched[&c] = ID
		list = append(list, &c)
	}

	for _, c := range list {
		ID := fetched[c]

		//todo check auth

		if _, exists := result[ID]; !exists {
			result[ID] = make([]*Category, 0)
		}

		result[ID] = append(result[ID], c)
	}

	return result, nil
}

func GetCategoryObjectIDs[SourceType relation.RelationID](siteID, source string, categoryID ...int) (map[int][]SourceType, error) {
	result := make(map[int][]SourceType)

	relations, err := relation.ExistRelations[SourceType, int](siteID, source, "category", "", nil, categoryID)
	if err != nil {
		return nil, err
	}

	for _, r := range relations {
		result[*r.BID] = append(result[*r.BID], *r.AID)
	}

	return result, nil
}

func (c *Category) Validate(siteID string) error {

	sm, err := module.GetModule(siteID)
	if err != nil {
		return err
	}

	var categoryType *module.CategoryType
	for _, ct := range sm.CategoryTypes {
		if ct.Source == c.Source && ct.Type == c.Type {
			categoryType = ct
			break
		}
	}

	if categoryType == nil {
		return errors.New("invalid type")
	}

	if c.Profile == nil {
		c.Profile = make(map[string][]string)
	}

	return nil
}

func (c *Category) Add(siteID string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		if err := c.AddWithTxn(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func (c *Category) AddWithTxn(siteID string, txn *sql.Tx) error {

	if c.Profile == nil {
		c.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(c.Profile)

	if ret, err := txn.Exec(`
		INSERT INTO `+categoryTableName(siteID)+`
			(type, source, parent_id, name, description, profile, sort)
		VALUES
			(?,?,?,?,?,?,?)
	`, c.Type, c.Source, c.ParentCategoryID, c.Name, c.Description, string(profile), c.Sort); err != nil {
		log.Println("error insert category: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		c.ID = int(id)
	}
	return nil
}

func (c *Category) Update(siteID string) error {

	if c.Profile == nil {
		c.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(c.Profile)

	setStmts := make([]string, 0)
	values := make([]interface{}, 0)

	setStmts = append(setStmts, "parent_id = ?", "name = ?", "description = ?", "profile=?", "sort = ?")
	values = append(values, c.ParentCategoryID, c.Name, c.Description, string(profile), c.Sort)

	if c.Total >= 0 {
		setStmts = append(setStmts, "total = ?")
		values = append(values, c.Total)
	}

	values = append(values, c.ID)

	if _, err := datasource.GetConn().Exec(`
		UPDATE 
			`+categoryTableName(siteID)+`
		SET
			`+strings.Join(setStmts, ",")+`
		WHERE
			id = ?
	`, values...); err != nil {
		log.Println("error update category: ", err)
		return err
	}
	return nil
}

func (c *Category) Delete(siteID string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		if err := c.DeleteWithTxn(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func (c *Category) DeleteWithTxn(siteID string, txn *sql.Tx) error {

	sm, err := module.GetModule(siteID)
	if err != nil {
		return err
	}

	c, err = GetCategoryWithTxn(siteID, txn, c.ID, true)
	if err != nil {
		return err
	}

	if _, err := txn.Exec(`
		DELETE FROM 
			`+categoryTableName(siteID)+`
		WHERE
			id = ?
	`, c.ID); err != nil {
		log.Println("error remove category: ", err)
		return err
	}

	var categoryType *module.CategoryType
	for _, ct := range sm.CategoryTypes {
		if ct.Source == c.Source && ct.Type == c.Type {
			categoryType = ct
			break
		}
	}

	if categoryType != nil {
		exists, err := relation.ExistRelationsWithTxn[string, int](siteID, txn, categoryType.Source, "category", "", nil, []int{c.ID})
		if err != nil {
			return err
		}

		for _, r := range exists {
			if err := r.Delete(siteID, txn); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetCategoryWithTxn(siteID string, txn *sql.Tx, categoryID int, forUpdate bool) (*Category, error) {

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s category
		WHERE
			category.id = ?
	`, categogyColumn, categoryTableName(siteID))

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, categoryID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var c Category
		if err := c.scan(rows); err != nil {
			return nil, err
		}

		return &c, nil
	}
	return nil, errors.New("category not exists")
}

func GetCategoriesByTypeWithTxn(siteID string, txn *sql.Tx, categoryType string, forUpdate bool) ([]*Category, error) {

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s category
		WHERE
			category.type = ?
	`, categogyColumn, categoryTableName(siteID))

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, categoryType)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]*Category, 0)

	for rows.Next() {
		var c Category
		if err := c.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, nil
}

func AddCategoryClientAgentMapping(siteID string, categoryID int, clientAgent string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		if err := AddCategoryClientAgentMappingWithTxn(siteID, txn, categoryID, clientAgent); err != nil {
			panic(err)
		}
	})
}

func AddCategoryClientAgentMappingWithTxn(siteID string, txn *sql.Tx, categoryID int, clientAgent string) error {

	r := new(relation.Relation[int, string])
	r.A = "category"
	r.B = "clientagent"
	r.AID = &categoryID
	r.BID = &clientAgent

	return r.Add(siteID, txn)
}

func DeleteCategoryClientAgentMapping(siteID string, categoryID int, clientAgent string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		if err := DeleteCategoryClientAgentMappingWithTxn(siteID, txn, categoryID, clientAgent); err != nil {
			panic(err)
		}
	})
}

func DeleteCategoryClientAgentMappingWithTxn(siteID string, txn *sql.Tx, categoryID int, clientAgent string) error {
	r := new(relation.Relation[int, string])
	r.A = "category"
	r.B = "clientagent"
	r.AID = &categoryID
	r.BID = &clientAgent

	return r.Delete(siteID, txn)
}

func GetCategoryClientAgents(siteID string, categoryID ...int) (map[int][]string, error) {

	result := make(map[int][]string)
	relationList, err := relation.ExistRelations[int, string](siteID, "category", "clientagent", "", categoryID, nil)
	if err != nil {
		return nil, err
	}

	for _, r := range relationList {
		if _, exists := result[*r.AID]; !exists {
			result[*r.AID] = make([]string, 0)
		}
		result[*r.AID] = append(result[*r.AID], *r.BID)
	}

	return result, nil
}

func AddCategoryMapping[SourceType relation.RelationID](siteID, source string, sourceID SourceType, cid int) error {

	return datasource.Txn(func(txn *sql.Tx) {
		if err := AddCategoryMappingWithTxn(siteID, txn, source, sourceID, cid); err != nil {
			panic(err)
		}
	})
}

func AddCategoryMappingWithTxn[SourceType relation.RelationID](siteID string, txn *sql.Tx, source string, sourceID SourceType, cid int) error {

	c, err := GetCategoryWithTxn(siteID, txn, cid, true)
	if err != nil {
		return err
	}

	sm, err := module.GetModule(siteID)
	if err != nil {
		return err
	}

	var categoryType *module.CategoryType
	for _, ct := range sm.CategoryTypes {
		if ct.Source == c.Source && ct.Type == c.Type {
			categoryType = ct
			break
		}
	}

	if categoryType.Source != source {
		return errors.New("类别不符")
	}

	r := &relation.Relation[SourceType, int]{
		A:   source,
		B:   "category",
		AID: &sourceID,
		BID: &cid,
	}

	if err := r.Add(siteID, txn); err != nil {
		return err
	}

	return nil
}

func DeleteCategoryMapping[SourceType relation.RelationID](siteID, source string, sourceID SourceType, cid int) error {
	return datasource.Txn(func(txn *sql.Tx) {
		if err := DeleteCategoryMappingWithTxn(siteID, txn, source, sourceID, cid); err != nil {
			panic(err)
		}
	})
}

func DeleteCategoryMappingWithTxn[SourceType relation.RelationID](siteID string, txn *sql.Tx, source string, sourceID SourceType, cid int) error {

	c, err := GetCategoryWithTxn(siteID, txn, cid, true)
	if err != nil {
		return err
	}

	sm, err := module.GetModule(siteID)
	if err != nil {
		return err
	}

	var categoryType *module.CategoryType
	for _, ct := range sm.CategoryTypes {
		if ct.Source == c.Source && ct.Type == c.Type {
			categoryType = ct
			break
		}
	}

	if categoryType.Source != source {
		return errors.New("类别不符")
	}

	r := &relation.Relation[SourceType, int]{
		A:   source,
		B:   "category",
		AID: &sourceID,
		BID: &cid,
	}

	if err := r.Delete(siteID, txn); err != nil {
		return err
	}
	return nil
}

func JoinCategoryMapping(siteID, source string, categoryType string, cids ...int) (string, []string, []interface{}, error) {
	var SQL string

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if categoryType != "" {
		cJoin, cTable, cJoinWhere, cJoinValues, err := relation.JoinSQL[int](siteID, source, "category", "", source)
		if err != nil {
			return "", nil, nil, err
		}

		SQL += "\n" + cJoin
		whereStmts = append(whereStmts, cJoinWhere...)
		values = append(values, cJoinValues...)

		SQL += fmt.Sprintf(`
			JOIN %s as category
				ON category.id = %s.b
		`, categoryTableName(siteID), cTable)

		whereStmts = append(whereStmts, "category.type = ?")
		values = append(values, categoryType)
	}

	for _, cid := range cids {
		perJoin, _, joinWhere, joinValues, err := relation.JoinSQL(siteID, source, "category", "", source, cid)
		if err != nil {
			return "", nil, nil, err
		}
		if perJoin != "" {
			SQL += "\n" + perJoin
			whereStmts = append(whereStmts, joinWhere...)
			values = append(values, joinValues...)
		}
	}

	return SQL, whereStmts, values, nil
}
