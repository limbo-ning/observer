package role

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

type Role struct {
	ID          int                 `json:"ID"`
	Series      string              `json:"series"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Profile     map[string][]string `json:"profile"`
	Sort        int                 `json:"sort"`
}

const roleColumn = "role.id, role.series, role.name, role.description, role.profile, role.sort"

func roleTableName(siteID string, traceParent bool) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_ROLE, traceParent)
	if err != nil {
		return "", err
	}
	return moduleSite + "_role", nil
}

func (m *Role) scan(rows *sql.Rows) error {
	var profile string
	if err := rows.Scan(&m.ID, &m.Series, &m.Name, &m.Description, &profile, &m.Sort); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(profile), &m.Profile); err != nil {
		return err
	}
	return nil
}

func (m *Role) Validate(siteID string) error {
	if m.Profile == nil {
		m.Profile = make(map[string][]string)
	}

	if m.Series != "" {
		vm, err := GetModule(siteID)
		if err != nil {
			return err
		}

		if rs := vm.GetRoleSeries(m.Series); rs == nil {
			return errors.New("invalid series")
		}
	}

	return nil
}

func (m *Role) Add(siteID string) error {

	table, err := roleTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Validate(siteID); err != nil {
		return err
	}
	profile, _ := json.Marshal(m.Profile)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(series, name, description, profile, sort)
		VALUES
			(?,?,?,?,?)
	`, table), m.Series, m.Name, m.Description, string(profile), m.Sort); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *Role) Update(siteID string) error {

	table, err := roleTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Validate(siteID); err != nil {
		return err
	}

	profile, _ := json.Marshal(m.Profile)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			series=?, name=?, description=?, profile=?, sort=?
		WHERE
			id=?
	`, table), m.Series, m.Name, m.Description, string(profile), m.Sort, m.ID); err != nil {
		return err
	}

	return nil
}
func (m *Role) Delete(siteID string) error {
	table, err := roleTableName(siteID, true)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, table), m.ID); err != nil {
		return err
	}

	return nil
}
func (m *Role) SyncAuth(siteID string) error {

	roles, err := GetRoles(siteID, "", m.ID)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return errors.New("角色不存在")
	}

	m = roles[0]

	if m.Series == "" {
		return nil
	}

	vm, err := GetModule(siteID)
	if err != nil {
		return err
	}

	for _, s := range vm.Series {
		if s.Series == m.Series {
			if s.AuthTemplateID <= 0 {
				return nil
			}

			exists, err := GetRoleAuthority(siteID, nil, m.ID)
			if err != nil {
				return err
			}

			for _, r := range exists[m.ID] {
				if err := r.delete(siteID); err != nil {
					return err
				}
			}

			templateAuth, err := GetRoleAuthority(siteID, nil, -1*s.AuthTemplateID)
			if err != nil {
				return err
			}

			list := templateAuth[-1*s.AuthTemplateID]
			if len(list) == 0 {
				return nil
			}

			for _, a := range list {
				a.RoleID = m.ID
				if err := a.add(siteID); err != nil {
					return err
				}
			}

			return nil
		}
	}

	return errors.New("角色类型不存在")
}
func GetRoles(siteID, series string, roleID ...int) ([]*Role, error) {
	table, err := roleTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if series != "" {
		whereStmts = append(whereStmts, "role.series = ?")
		values = append(values, series)
	}

	if len(roleID) > 0 {
		if len(roleID) == 1 {
			whereStmts = append(whereStmts, "role.id=?")
			values = append(values, roleID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range roleID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("role.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s role
	`, roleColumn, table)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY role.sort DESC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	list := make([]*Role, 0)
	for rows.Next() {
		var m Role
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}

	return list, nil
}

func GetRoleWithTxn(siteID string, txn *sql.Tx, roleID int, forUpdate bool) (*Role, error) {
	table, err := roleTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s role
		WHERE
			id = ?
	`, roleColumn, table)

	if forUpdate {
		SQL += "FOR UPDATE"
	}

	rows, err := txn.Query(SQL, roleID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var m Role
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		return &m, nil
	}

	return nil, fmt.Errorf("找不到角色[%d]", roleID)
}

func GetRoleSeriesWithTxn(siteID string, txn *sql.Tx, series string, forUpdate bool) ([]*Role, error) {
	table, err := roleTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s role
		WHERE
			series = ?
	`, roleColumn, table)

	if forUpdate {
		SQL += "FOR UPDATE"
	}

	rows, err := txn.Query(SQL, series)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]*Role, 0)
	for rows.Next() {
		var m Role
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &m)
	}

	return result, nil
}
