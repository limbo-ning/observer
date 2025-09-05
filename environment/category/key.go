package category

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"log"
// 	"strconv"
// 	"strings"

// 	"obsessiontech/common/datasource"
// 	"obsessiontech/environment/authority"
// 	"obsessiontech/environment/category/module"
// 	"obsessiontech/environment/relation"
// )

// type CategoryKey struct {
// 	ID          int                 `json:"ID"`
// 	Name        string              `json:"name"`
// 	Source      string              `json:"source"`
// 	Type        string              `json:"type"`
// 	Description string              `json:"description"`
// 	Profile     map[string][]string `json:"profile"`
// 	Sort        int                 `json:"sort"`
// }

// const categoryKeyColumn = "categoryKey.id, categoryKey.name, categoryKey.source, categoryKey.type, categoryKey.description, categoryKey.profile, categoryKey.sort"

// func (c *CategoryKey) scan(rows *sql.Rows, appendix ...interface{}) error {
// 	var profile string

// 	dest := make([]interface{}, 0)
// 	dest = append(dest, &c.ID, &c.Name, &c.Source, &c.Type, &c.Description, &profile, &c.Sort)
// 	dest = append(dest, appendix...)

// 	if err := rows.Scan(dest...); err != nil {
// 		return err
// 	}

// 	if err := json.Unmarshal([]byte(profile), &c.Profile); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func categoryKeyTableName(siteID string) string {
// 	return siteID + "_categorykey"
// }

// func (c *CategoryKey) validate(siteID string) error {
// 	if c.Profile == nil {
// 		c.Profile = make(map[string][]string)
// 	}

// 	if c.Source == "" {
// 		return errors.New("invalid source")
// 	}

// 	if c.Type != "" {
// 		categoryModule, err := module.GetModule(siteID)
// 		if err != nil {
// 			return err
// 		}

// 		for _, s := range categoryModule.CategoryTypes {
// 			if s.Type == c.Type && s.Source == c.Source {
// 				return nil
// 			}
// 		}
// 		return errors.New("invalid type")
// 	}

// 	return nil
// }

// func (c *CategoryKey) Add(siteID string) error {

// 	if err := c.validate(siteID); err != nil {
// 		return err
// 	}

// 	profile, _ := json.Marshal(c.Profile)

// 	if ret, err := datasource.GetConn().Exec(`
// 		INSERT INTO `+categoryKeyTableName(siteID)+`
// 			(name, source, type, description, profile, sort)
// 		VALUES
// 			(?,?,?,?,?,?)
// 	`, c.Name, c.Source, c.Type, c.Description, string(profile), c.Sort); err != nil {
// 		log.Println("error insert categoryKey: ", err)
// 		return err
// 	} else if id, err := ret.LastInsertId(); err != nil {
// 		return err
// 	} else {
// 		c.ID = int(id)
// 	}
// 	return nil
// }

// func (c *CategoryKey) Update(siteID string) error {

// 	if err := c.validate(siteID); err != nil {
// 		return err
// 	}

// 	profile, _ := json.Marshal(c.Profile)

// 	setStmts := make([]string, 0)
// 	values := make([]interface{}, 0)

// 	setStmts = append(setStmts, "name = ?", "type = ?", "description = ?", "profile=?", "sort = ?")
// 	values = append(values, c.Name, c.Type, c.Description, string(profile), c.Sort)

// 	values = append(values, c.ID)

// 	if _, err := datasource.GetConn().Exec(`
// 		UPDATE
// 			`+categoryKeyTableName(siteID)+`
// 		SET
// 			`+strings.Join(setStmts, ",")+`
// 		WHERE
// 			id = ?
// 	`, values...); err != nil {
// 		log.Println("error update categoryKey: ", err)
// 		return err
// 	}
// 	return nil
// }

// func (c *CategoryKey) Delete(siteID string) error {
// 	return datasource.Txn(func(txn *sql.Tx) {
// 		if err := c.DeleteWithTxn(siteID, txn); err != nil {
// 			panic(err)
// 		}
// 	})
// }

// func (c *CategoryKey) DeleteWithTxn(siteID string, txn *sql.Tx) error {

// 	var err error

// 	categories, err := GetCategoriesByKeyWithTxn(siteID, txn, c.ID, true)
// 	if err != nil {
// 		return err
// 	}

// 	for _, cat := range categories {
// 		if err := cat.DeleteWithTxn(siteID, txn); err != nil {
// 			return err
// 		}
// 	}

// 	if _, err := txn.Exec(`
// 		DELETE FROM
// 			`+categoryKeyTableName(siteID)+`
// 		WHERE
// 			id = ?
// 	`, c.ID); err != nil {
// 		log.Println("error remove categoryKey: ", err)
// 		return err
// 	}

// 	return nil
// }

// func (c *CategoryKey) CheckAuth(siteID string, actionAuth authority.ActionAuthSet, args ...interface{}) error {
// 	for _, a := range actionAuth {
// 		action := strings.TrimPrefix(a.Action, module.MODULE_CATEGORY+"#")

// 		switch action {
// 		case module.ACTION_ADMIN_VIEW:
// 		case module.ACTION_ADMIN_EDIT:
// 		case module.ACTION_VIEW:
// 			id, err := strconv.Atoi(a.RoleType)
// 			if err != nil {
// 				log.Println("error atoi role type: ", err)
// 				continue
// 			}
// 			if id != c.ID {
// 				continue
// 			}
// 		case module.ACTION_VIEW_TYPE:
// 			if c.Type != a.RoleType {
// 				continue
// 			}
// 		default:
// 			continue
// 		}

// 		return nil
// 	}

// 	return errors.New("无权限")
// }

// func getCategoryKeys(siteID string, keyID ...int) (map[int]*CategoryKey, error) {

// 	result := make(map[int]*CategoryKey)

// 	if len(keyID) == 0 {
// 		return result, nil
// 	}

// 	whereStmts := make([]string, 0)
// 	values := make([]interface{}, 0)

// 	if len(keyID) == 1 {
// 		whereStmts = append(whereStmts, "id = ?")
// 		values = append(values, keyID[0])
// 	} else {
// 		placeholder := make([]string, 0)
// 		for _, id := range keyID {
// 			placeholder = append(placeholder, "?")
// 			values = append(values, id)
// 		}
// 		whereStmts = append(whereStmts, fmt.Sprintf("id IN (%s)", strings.Join(placeholder, ",")))
// 	}

// 	SQL := fmt.Sprintf(`
// 		SELECT
// 			%s
// 		FROM
// 			%s categoryKey
// 		WHERE
// 			%s
// 	`, categoryKeyColumn, categoryKeyTableName(siteID), whereStmts[0])

// 	rows, err := datasource.GetConn().Query(SQL, values...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer rows.Close()

// 	for rows.Next() {
// 		var key CategoryKey
// 		if err := key.scan(rows); err != nil {
// 			return nil, err
// 		}
// 		result[key.ID] = &key
// 	}
// 	return result, nil
// }

// func GetCategoryKeyWithTxn(siteID string, txn *sql.Tx, keyID int, forUpdate bool) (*CategoryKey, error) {
// 	SQL := fmt.Sprintf(`
// 		SELECT
// 			%s
// 		FROM
// 			%s categoryKey
// 		WHERE
// 			id = ?
// 	`, categoryKeyColumn, categoryKeyTableName(siteID))

// 	if forUpdate {
// 		SQL += "\nFOR UPDATE"
// 	}

// 	rows, err := txn.Query(SQL, keyID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer rows.Close()

// 	if rows.Next() {
// 		var key CategoryKey
// 		if err := key.scan(rows); err != nil {
// 			return nil, err
// 		}
// 		return &key, nil
// 	}
// 	return nil, errors.New("category key not exists")
// }

// func GetCategoryKeyList(siteID, categoryType, source, clientAgent string, actionAuth authority.ActionAuthSet, keyID ...int) ([]*CategoryKey, error) {
// 	whereStmts := make([]string, 0)
// 	values := make([]interface{}, 0)

// 	var joinSQL string

// 	if clientAgent != "" {
// 		join, _, joinWhere, joinValues, err := relation.JoinSQL(siteID, "categoryKey", "clientagent", "", "categoryKey", clientAgent)
// 		if err != nil {
// 			return nil, err
// 		}

// 		joinSQL += join
// 		whereStmts = append(whereStmts, joinWhere...)
// 		values = append(values, joinValues...)
// 	}

// 	if categoryType != "" {
// 		whereStmts = append(whereStmts, "categoryKey.type = ?")
// 		values = append(values, categoryType)
// 	}

// 	if source != "" {
// 		whereStmts = append(whereStmts, "categoryKey.source = ?")
// 		values = append(values, source)
// 	}

// 	if len(keyID) > 0 {
// 		if len(keyID) == 0 {
// 			whereStmts = append(whereStmts, "categoryKey.id = ?")
// 			values = append(values, keyID[0])
// 		} else {
// 			placeholder := make([]string, 0)
// 			for _, id := range keyID {
// 				placeholder = append(placeholder, "?")
// 				values = append(values, id)
// 			}
// 			whereStmts = append(whereStmts, fmt.Sprintf("categoryKey.id IN (%s)", strings.Join(placeholder, ",")))
// 		}
// 	}

// 	SQL := fmt.Sprintf(`
// 		SELECT
// 			%s
// 		FROM
// 			%s categoryKey
// 	`, categoryKeyColumn, categoryKeyTableName(siteID))

// 	if len(whereStmts) > 0 {
// 		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
// 	}

// 	SQL += joinSQL

// 	SQL += "\nORDER BY categoryKey.sort DESC"

// 	rows, err := datasource.GetConn().Query(SQL, values...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	result := make([]*CategoryKey, 0)
// 	for rows.Next() {
// 		var categoryKey CategoryKey
// 		if err := categoryKey.scan(rows); err != nil {
// 			return nil, err
// 		}
// 		if categoryKey.CheckAuth(siteID, actionAuth) == nil {
// 			result = append(result, &categoryKey)
// 		}
// 	}

// 	return result, nil
// }

// func AddCategoryKeyClientAgentMapping(siteID string, categoryKeyID int, clientAgent string) error {

// 	return datasource.Txn(func(txn *sql.Tx) {
// 		if err := AddCategoryKeyClientAgentMappingWithTxn(siteID, txn, categoryKeyID, clientAgent); err != nil {
// 			panic(err)
// 		}
// 	})
// }

// func AddCategoryKeyClientAgentMappingWithTxn(siteID string, txn *sql.Tx, categoryKeyID int, clientAgent string) error {

// 	r := new(relation.Relation[int, string])
// 	r.A = "categorykey"
// 	r.B = "clientagent"
// 	r.AID = &categoryKeyID
// 	r.BID = &clientAgent

// 	return r.Add(siteID, txn)
// }

// func DeleteCategoryKeyClientAgentMapping(siteID string, categoryKeyID int, clientAgent string) error {

// 	return datasource.Txn(func(txn *sql.Tx) {
// 		if err := DeleteCategoryKeyClientAgentMappingWithTxn(siteID, txn, categoryKeyID, clientAgent); err != nil {
// 			panic(err)
// 		}
// 	})
// }

// func DeleteCategoryKeyClientAgentMappingWithTxn(siteID string, txn *sql.Tx, categoryKeyID int, clientAgent string) error {
// 	r := new(relation.Relation[int, string])
// 	r.A = "categorykey"
// 	r.B = "clientagent"
// 	r.AID = &categoryKeyID
// 	r.BID = &clientAgent

// 	return r.Delete(siteID, txn)
// }

// func GetCategoryKeyClientAgents(siteID string, categoryKeyID ...int) (map[int][]string, error) {

// 	result := make(map[int][]string)
// 	relationList, err := relation.ExistRelations[int, string](siteID, "categorykey", "clientagent", "", categoryKeyID, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, r := range relationList {
// 		if _, exists := result[*r.AID]; !exists {
// 			result[*r.AID] = make([]string, 0)
// 		}
// 		result[*r.AID] = append(result[*r.AID], *r.BID)
// 	}

// 	return result, nil
// }
