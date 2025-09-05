package authority

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site/initialization"
)

var empowerRegistry = make(map[string]func() IEmpower)

var e_empower_not_support = errors.New("不支持的授权")

func RegisterEmpower(empowerType string, fac func() IEmpower) {
	if _, exists := empowerRegistry[empowerType]; exists {
		log.Panic("duplicate empower")
	}
	empowerRegistry[empowerType] = fac
}

func GetEmpower(empowerType string) (IEmpower, error) {
	if fac, exists := empowerRegistry[empowerType]; exists {
		return fac(), nil
	}

	log.Println(empowerType, e_empower_not_support)
	return nil, e_empower_not_support
}

func Empowers() map[string]IEmpower {
	result := make(map[string]IEmpower)

	for k, fac := range empowerRegistry {
		result[k] = fac()
	}

	return result
}

type IEmpower interface {
	EmpowerID(siteID string, actionAuth ActionAuthSet) ([]string, error)
}

var E_no_empower = errors.New("没有授权")
var E_empower_not_restricted = errors.New("仅关联")

type Empower struct {
	ID        string `json:"ID"`
	Empower   string `json:"empower"`
	EmpowerID string `json:"empowerID"`
	Action    string `json:"action"`
}

func empowerTable(siteID, source string) string {
	return fmt.Sprintf("%s_%sempower", siteID, source)
}

func (ae *Empower) scan(rows *sql.Rows) error {
	if err := rows.Scan(&ae.ID, &ae.Empower, &ae.EmpowerID, &ae.Action); err != nil {
		return err
	}
	return nil
}

const empowerColumns = "empower.id, empower.empower, empower.empower_id, empower.action"

func (ae *Empower) Add(siteID, source string, txn *sql.Tx) error {

	if err := initialization.CreateTable(siteID, txn, fmt.Sprintf("%sempower", source), "empower"); err != nil {
		return err
	}

	if _, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(id,empower,empower_id,action)
		VALUES
			(?,?,?,?)
		ON DUPLICATE KEY UPDATE
			id=VALUES(id), empower=VALUES(empower), empower_id=VALUES(empower_id), action=VALUES(action)
	`, empowerTable(siteID, source)), ae.ID, ae.Empower, ae.EmpowerID, ae.Action); err != nil {
		log.Println("error insert empower: ", err)
		return err
	}

	return nil
}

func (ae *Empower) Delete(siteID, source string, txn *sql.Tx) error {
	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ? AND empower = ? AND empower_id = ? AND action = ?
	`, empowerTable(siteID, source)), ae.ID, ae.Empower, ae.EmpowerID, ae.Action); err != nil {
		log.Println("error delete empower: ", err)
		return err
	}

	return nil
}

func GetEmpowerDetails(siteID, empower string, empowerID []string, source string, sourceID []string, groupBy string) (map[string][]*Empower, error) {
	result := make(map[string][]*Empower)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(sourceID) == 1 {
		whereStmts = append(whereStmts, "empower.id = ?")
		values = append(values, sourceID[0])
	} else if len(sourceID) > 1 {
		placeholder := make([]string, 0)
		for _, id := range sourceID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("empower.id IN (%s)", strings.Join(placeholder, ",")))
	} else if empower == "" || len(empowerID) == 0 {
		return result, nil
	}

	if empower != "" {
		whereStmts = append(whereStmts, "empower.empower = ?")
		values = append(values, empower)

		if len(empowerID) > 0 {
			if len(empowerID) == 1 {
				whereStmts = append(whereStmts, "empower.empower_id = ?")
				values = append(values, empowerID[0])
			} else if len(empowerID) > 1 {
				placeholder := make([]string, 0)
				for _, id := range empowerID {
					placeholder = append(placeholder, "?")
					values = append(values, id)
				}
				whereStmts = append(whereStmts, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
			}
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s empower
	`, empowerColumns, empowerTable(siteID, source))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var ae Empower
		if err := ae.scan(rows); err != nil {
			return nil, err
		}

		var key string

		switch groupBy {
		case "empower":
			key = ae.EmpowerID
		case "source":
			key = ae.ID
		default:
			key = ae.ID
		}

		log.Println("group by: ", groupBy, key, ae)

		if _, exists := result[key]; !exists {
			result[key] = make([]*Empower, 0)
		}
		result[key] = append(result[key], &ae)
	}

	return result, nil
}

func GetEmpowers(siteID, source string, actionAuth ActionAuthSet, actions map[string]string, sourceID ...string) (map[string]map[string]bool, error) {
	result := make(map[string]map[string]bool)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(sourceID) == 1 {
		whereStmts = append(whereStmts, "empower.id = ?")
		values = append(values, sourceID[0])
	} else if len(sourceID) > 1 {
		placeholder := make([]string, 0)
		for _, id := range sourceID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("empower.id IN (%s)", strings.Join(placeholder, ",")))
	} else {
		return result, nil
	}

	if len(actions) == 0 {
		return result, nil
	} else {
		toCheck := make([]string, 0)
		for a, admin := range actions {

			if actionAuth.CheckAction(admin) {
				for _, sid := range sourceID {
					if _, exists := result[sid]; !exists {
						result[sid] = make(map[string]bool)
					}
					result[sid][a] = true
				}
			} else {
				toCheck = append(toCheck, a)
			}
		}

		if len(toCheck) == 0 {
			return result, nil
		} else if len(toCheck) == 1 {
			whereStmts = append(whereStmts, "empower.action = ?")
			values = append(values, toCheck[0])
		} else if len(toCheck) > 1 {
			placeholder := make([]string, 0)
			for _, id := range toCheck {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("empower.action IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	orWhere := make([]string, 0)
	for empowerT, empower := range Empowers() {

		perWhere := make([]string, 0)

		ids, err := empower.EmpowerID(siteID, actionAuth)
		if err != nil {
			if err == E_empower_not_restricted {
				continue
			}
			return nil, err
		}

		if len(ids) == 0 {
			continue
		}

		perWhere = append(perWhere, "empower.empower = ?")
		values = append(values, empowerT)

		if len(ids) == 1 {
			perWhere = append(perWhere, "empower.empower_id = ?")
			values = append(values, ids[0])
		} else if len(ids) > 1 {
			placeholder := make([]string, 0)
			for _, id := range ids {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			perWhere = append(perWhere, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
		}

		orWhere = append(orWhere, fmt.Sprintf("(%s)", strings.Join(perWhere, " AND ")))
	}

	if len(orWhere) > 0 {
		whereStmts = append(whereStmts, fmt.Sprintf("(%s)", strings.Join(orWhere, " OR ")))
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s empower
	`, empowerColumns, empowerTable(siteID, source))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var ae Empower
		if err := ae.scan(rows); err != nil {
			return nil, err
		}

		if _, exists := result[ae.ID]; !exists {
			result[ae.ID] = make(map[string]bool)
		}
		result[ae.ID][ae.Action] = true
	}

	return result, nil
}

func ClearEmpowers(siteID, source, sourceID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, empowerTable(siteID, source)), sourceID); err != nil {
		log.Println("error clear empower: ", sourceID, err)
		return err
	}

	return nil
}

func AddEmpower(siteID, source, sourceID string, txn *sql.Tx, empower string, empowerID []string, authType []string) error {

	_, err := GetEmpower(empower)
	if err != nil {
		return err
	}

	for _, id := range empowerID {
		for _, a := range authType {
			r := new(Empower)
			r.Action = a
			r.ID = sourceID
			r.Empower = empower
			r.EmpowerID = id

			if err := r.Add(siteID, source, txn); err != nil {
				return err
			}
		}
	}

	return nil
}

func CountEmpowersWithTxn(siteID, source, sourceID string, txn *sql.Tx) (int, error) {
	rows, err := txn.Query(fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s
		WHERE
			id = ?
		FOR UPDATE
	`, empowerTable(siteID, source)), sourceID)
	if err != nil {
		log.Println("error count empower: ", err)
		return 0, err
	}

	defer rows.Close()

	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err != nil {
			log.Println("error count empower: ", err)
		}
		return count, nil
	}

	return 0, errors.New("error count empower, no result")
}

func DeleteEmpower(siteID, source, sourceID string, txn *sql.Tx, empower string, empowerID []string, authType ...string) error {

	for _, id := range empowerID {
		for _, a := range authType {
			r := new(Empower)
			r.Action = a
			r.ID = sourceID
			r.Empower = empower
			r.EmpowerID = id

			if err := r.Delete(siteID, source, txn); err != nil {
				return err
			}
		}
	}

	return nil
}

func JoinEmpower(siteID, source string, actionAuth ActionAuthSet, adminActions map[string]string, authAction, joinTable string, joinColumn string, empowerType string, empowerID ...string) (string, []string, []interface{}, error) {

	log.Println("create join: ", source, authAction, joinTable, joinColumn, empowerType, empowerID)

	var joinSQL string
	joinWhere := make([]string, 0)
	joinValues := make([]interface{}, 0)

	orWhere := make([]string, 0)
	if empowerType == "" || len(empowerID) == 0 {

		if actionAuth.CheckAction(adminActions[authAction]) {
			return "", nil, nil, nil
		}

		for empowerT, empower := range Empowers() {
			perWhere := make([]string, 0)

			ids, err := empower.EmpowerID(siteID, actionAuth)
			if err != nil {
				if err == E_empower_not_restricted {
					continue
				}
				return "", nil, nil, err
			}

			perWhere = append(perWhere, "empower.empower = ?")
			joinValues = append(joinValues, empowerT)

			//no empowered ids means join condition should always be false
			if len(ids) == 0 {
				perWhere = append(perWhere, "empower.empower IS NULL")
			} else {
				if len(ids) == 1 {
					perWhere = append(perWhere, "empower.empower_id = ?")
					joinValues = append(joinValues, ids[0])
				} else if len(ids) > 1 {
					placeholder := make([]string, 0)
					for _, id := range ids {
						placeholder = append(placeholder, "?")
						joinValues = append(joinValues, id)
					}
					perWhere = append(perWhere, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
				}
			}

			orWhere = append(orWhere, fmt.Sprintf("(%s)", strings.Join(perWhere, " AND ")))
		}
	} else {
		perWhere := make([]string, 0)
		perWhere = append(perWhere, "empower.empower = ?")
		joinValues = append(joinValues, empowerType)

		if actionAuth.CheckAction(adminActions[authAction]) {
			placeholder := make([]string, 0)
			for _, id := range empowerID {
				joinValues = append(joinValues, id)
				placeholder = append(placeholder, "?")
			}
			perWhere = append(perWhere, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
		} else {
			empower, err := GetEmpower(empowerType)
			if err != nil {
				return "", nil, nil, err
			}
			ids, err := empower.EmpowerID(siteID, actionAuth)
			if err == E_empower_not_restricted {
				placeholder := make([]string, 0)
				for _, id := range empowerID {
					joinValues = append(joinValues, id)
					placeholder = append(placeholder, "?")
				}
				perWhere = append(perWhere, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
			} else {
				if err != nil {
					return "", nil, nil, err
				}

				idmap := make(map[string]byte)
				if len(ids) > 0 {
					for _, id := range ids {
						idmap[id] = 1
					}
				}

				placeholder := make([]string, 0)
				for _, id := range empowerID {
					if _, exists := idmap[id]; exists {
						joinValues = append(joinValues, id)
						placeholder = append(placeholder, "?")
					}
				}

				//empty empowered ids means join condition should always be false
				if len(placeholder) > 0 {
					perWhere = append(perWhere, fmt.Sprintf("empower.empower_id IN (%s)", strings.Join(placeholder, ",")))
				} else {
					perWhere = append(perWhere, "empower.empower IS NULL")
				}
			}
		}

		orWhere = append(orWhere, fmt.Sprintf("(%s)", strings.Join(perWhere, " AND ")))
	}

	if len(orWhere) == 0 {
		return "", nil, nil, nil
	}

	onColumns := make([]string, 0)
	for _, c := range strings.Split(joinColumn, ",") {
		if strings.TrimSpace(c) != "" {
			onColumns = append(onColumns, fmt.Sprintf("empower.id = %s.%s", joinTable, strings.TrimSpace(c)))
		}
	}

	joinSQL = fmt.Sprintf(`
			JOIN
				%s empower
			ON
				%s
		`, empowerTable(siteID, source), strings.Join(onColumns, " OR "))
	joinWhere = append(joinWhere, fmt.Sprintf("(%s)", strings.Join(orWhere, " OR ")), "empower.action = ?")
	joinValues = append(joinValues, authAction)

	return joinSQL, joinWhere, joinValues, nil
}
