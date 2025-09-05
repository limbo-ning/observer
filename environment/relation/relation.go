package relation

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site/initialization"
)

const RELATION_A = "A"
const RELATION_B = "B"
const Default_relation_type = "-"

type RelationID interface {
	~int | ~string
}

// type T interface {
// 	comparable
// }

type Relation[AType RelationID, BType RelationID] struct {
	A      string `json:"a"`
	AID    *AType `json:"aID"`
	B      string `json:"b"`
	BID    *BType `json:"bID"`
	Type   string `json:"type"`
	Expire int    `json:"expire"`
}

func RelationTableName(siteID, A, B string) (string, error) {
	if siteID == "" || A == "" || B == "" {
		log.Printf("error wrong relation: %s_%s_%s", siteID, A, B)
		return "", errors.New("无效的关联")
	}
	return fmt.Sprintf("%s_%s_%s", siteID, A, B), nil
}

func analyzeTarget(A, B, target string) (string, string, string, error) {
	var targetName, targetTable, targetColumn string

	if strings.Contains(target, " alias ") {
		parts := strings.Split(target, "alias")
		target = strings.TrimSpace(parts[0])
		targetName = strings.TrimSpace(parts[1])
	}

	if strings.Contains(target, ".") {
		parts := strings.Split(target, ".")
		if targetName == "" {
			targetName = parts[0]
		}
		targetTable = parts[0]
		targetColumn = parts[1]
	} else {
		if targetName == "" {
			targetName = target
		}
		targetTable = target
		targetColumn = "id"
	}

	if A == B {
		switch targetName {
		case RELATION_A:
		case RELATION_B:
		default:
			return "", "", "", fmt.Errorf("when A B is symetric target should use A B instead of target name")
		}
		if targetTable == "" || targetTable == RELATION_A || targetTable == RELATION_B {
			targetTable = A
		}
	}

	return targetName, targetTable, targetColumn, nil
}

func JoinSQL[TargetType RelationID](siteID, A, B, relationType, target string, IDs ...TargetType) (string, string, []string, []interface{}, error) {
	tableName, err := RelationTableName(siteID, A, B)
	if err != nil {
		return "", "", nil, nil, err
	}

	alias := tableName
	if len(IDs) > 0 {
		for _, id := range IDs {
			alias += "_" + fmt.Sprint(id)
		}
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	var idField string
	if len(IDs) == 1 {
		idField = " = ?"
		values = append(values, IDs[0])
	} else if len(IDs) > 1 {
		placeholder := make([]string, 0)
		for _, id := range IDs {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		idField = fmt.Sprintf(" IN (%s)", strings.Join(placeholder, ","))
	}

	var sql string

	targetName, targetTable, targetColumn, err := analyzeTarget(A, B, target)
	if err != nil {
		return "", "", nil, nil, err
	}

	log.Printf("relation[%s] targetName[%s] targetTable[%s] targetColumn[%s]", tableName, targetName, targetTable, targetColumn)

	if A == targetName || targetName == RELATION_A {
		sql = fmt.Sprintf(`
			JOIN %s as %s
				ON %s.%s = %s.a
		`, tableName, alias, targetTable, targetColumn, alias)

		if idField != "" {
			whereStmts = append(whereStmts, fmt.Sprintf("%s.b %s", alias, idField))
		}
	} else if B == targetName || targetName == RELATION_B {
		sql = fmt.Sprintf(`
			JOIN %s as %s
				ON %s.%s = %s.b
		`, tableName, alias, targetTable, targetColumn, alias)

		if idField != "" {
			whereStmts = append(whereStmts, fmt.Sprintf("%s.a %s", alias, idField))
		}
	} else {
		return "", "", nil, nil, fmt.Errorf("error relation target not match: %s in %s", target, tableName)
	}

	if relationType != "" {
		whereStmts = append(whereStmts, fmt.Sprintf("%s.type = ?", alias))
		values = append(values, relationType)
	}

	whereStmts = append(whereStmts, fmt.Sprintf("(%s.expire = 0 || %s.expire > UNIX_TIMESTAMP())", alias, alias))

	return sql, alias, whereStmts, values, nil
}

func JoinNotExistSQL[TargetType RelationID](siteID, A, B, relationType, target string, IDs ...TargetType) (string, string, []string, []interface{}, error) {
	join, table, whereStmts, values, err := JoinSQL(siteID, A, B, relationType, target, IDs...)
	if err != nil {
		return "", "", nil, nil, err
	}

	if join != "" {
		join = "LEFT " + join
		whereStmts = append(whereStmts, fmt.Sprintf("%s.b IS NULL", table))
	}

	return join, table, whereStmts, values, nil

}

func ExistRelations[AType RelationID, BType RelationID](siteID, A, B, relationType string, AIDs []AType, BIDs []BType) ([]*Relation[AType, BType], error) {
	return getRelations(siteID, A, B, relationType, AIDs, BIDs)
}

func ExistRelationsWithTxn[AType RelationID, BType RelationID](siteID string, txn *sql.Tx, A, B, relationType string, AIDs []AType, BIDs []BType) ([]*Relation[AType, BType], error) {
	return getRelationsWithTxn(siteID, txn, A, B, relationType, AIDs, BIDs, true)
}

func getRelations[AType RelationID, BType RelationID](siteID, A, B, relationType string, AIDs []AType, BIDs []BType) ([]*Relation[AType, BType], error) {
	return getRelationsWithTxn(siteID, nil, A, B, relationType, AIDs, BIDs, false)
}

func getRelationsWithTxn[AType RelationID, BType RelationID](siteID string, txn *sql.Tx, A, B, relationType string, AIDs []AType, BIDs []BType, forUpdate bool) ([]*Relation[AType, BType], error) {

	result := make([]*Relation[AType, BType], 0)

	tableName, err := RelationTableName(siteID, A, B)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(AIDs) > 0 {
		if len(AIDs) == 1 {
			whereStmts = append(whereStmts, "a = ?")
			values = append(values, AIDs[0])
		} else {
			placeHolder := make([]string, 0)
			for _, id := range AIDs {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("a IN (%s)", strings.Join(placeHolder, ",")))
		}
	}
	if len(BIDs) > 0 {
		if len(BIDs) == 1 {
			whereStmts = append(whereStmts, "b = ?")
			values = append(values, BIDs[0])
		} else {
			placeHolder := make([]string, 0)
			for _, id := range BIDs {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("b IN (%s)", strings.Join(placeHolder, ",")))
		}
	}
	if relationType != "" {
		whereStmts = append(whereStmts, "type = ?")
		values = append(values, relationType)
	}

	SQL := `
		SELECT
			a, b, type, expire
		FROM
			` + tableName + `
	`

	if len(whereStmts) > 0 {
		SQL += "WHERE\n" + strings.Join(whereStmts, " AND ")
	}

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	log.Println("get relation: ", SQL, values)

	var rows *sql.Rows

	if txn != nil {
		rows, err = txn.Query(SQL, values...)
	} else {
		rows, err = datasource.GetConn().Query(SQL, values...)
	}
	if err != nil {
		log.Println("error check relation: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r Relation[AType, BType]
		r.A = A
		r.B = B
		rows.Scan(&r.AID, &r.BID, &r.Type, &r.Expire)

		result = append(result, &r)
	}

	return result, nil
}

func Count(siteID, A, B, relationType, target string, IDs []string) (int, error) {
	tableName, err := RelationTableName(siteID, A, B)
	if err != nil {
		return 0, err
	}

	if len(IDs) == 0 {
		return 0, nil
	}
	for i, id := range IDs {
		IDs[i] = "'" + id + "'"
	}
	ids := strings.Join(IDs, ",")

	var count int
	if A == target {
		if err := datasource.GetConn().QueryRow(fmt.Sprintf(`
			SELECT COUNT(DISTINCT a) FROM
				%s
			WHERE b IN (%s) AND type = ? AND (expire = 0 || expire > UNIX_TIMESTAMP())
		`, tableName, ids), relationType).Scan(&count); err != nil {
			return 0, err
		}
	} else if B == target {
		if err := datasource.GetConn().QueryRow(fmt.Sprintf(`
			SELECT COUNT(DISTINCT b) FROM
				%s
			WHERE a IN (%s) AND type = ? AND (expire = 0 || expire > UNIX_TIMESTAMP())
		`, tableName, ids), relationType).Scan(&count); err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("error relation target not match: %s in %s", target, tableName)
	}

	return count, nil
}

func (r *Relation[AType, BType]) Add(siteID string, txn *sql.Tx) error {

	log.Println("add relation called: ", r)

	if r.A == "" || r.B == "" || r.AID == nil || r.BID == nil {
		log.Printf("error add relation: %+v", *r.AID, *r.BID, r.Type, r.Expire)
		return errors.New("关联关系不允许空值")
	}

	if r.Type == "" {
		r.Type = Default_relation_type
	}

	tableName, err := RelationTableName(siteID, r.A, r.B)
	if err != nil {
		return err
	}

	if err := initialization.CreateTable(siteID, txn, fmt.Sprintf("%s_%s", r.A, r.B), "relation"); err != nil {
		return err
	}

	if _, err := txn.Exec(`
		INSERT INTO `+tableName+`
			(a, b, type, expire)
		VALUES
			(?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			type = VALUES(type), expire = VALUES(expire)
	`, *r.AID, *r.BID, r.Type, r.Expire); err != nil {
		return err
	}
	return nil
}

func (r *Relation[AType, BType]) Delete(siteID string, txn *sql.Tx) error {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	whereStmts = append(whereStmts, "a=?", "b=?")
	values = append(values, *r.AID, *r.BID)

	if r.Type != "" {
		whereStmts = append(whereStmts, "type=?")
		values = append(values, r.Type)
	}

	tableName, err := RelationTableName(siteID, r.A, r.B)
	if err != nil {
		return err
	}

	if _, err := txn.Exec(`
		DELETE FROM 
			`+tableName+`
		WHERE
	`+strings.Join(whereStmts, " AND "), values...); err != nil {
		return err
	}
	return nil
}
