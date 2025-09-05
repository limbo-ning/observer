package role

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site"
)

// func init() {
// 	Register("upgrade", func() IMilestone { return new(UpgradeMilestone) })
// 	Register("hint", func() IMilestone { return new(HintMilestone) })
// }

// type IMilestone interface {
// 	GetType() string
// 	Exec(string, *sql.Tx, int, bool) error
// }

type Milestone struct {
	Description string `json:"description"`
	MissionIDs  []int  `json:"missionIDs"`
}

// func (m *BaseMilestone) GetType() string {
// 	return m.Type
// }

// var milestoneRegistry = make(map[string]func() IMilestone)

// func Register(milestoneType string, fac func() IMilestone) {
// 	if _, exists := milestoneRegistry[milestoneType]; exists {
// 		panic("duplicate point milestone: " + milestoneType)
// 	}
// 	milestoneRegistry[milestoneType] = fac
// }

// type Milestones []IMilestone

// func (milestones *Milestones) UnmarshalJSON(data []byte) error {
// 	var raw []json.RawMessage

// 	if err := json.Unmarshal(data, &raw); err != nil {
// 		return err
// 	}

// 	list := make([]IMilestone, 0)

// 	for _, r := range raw {
// 		var a BaseMilestone
// 		json.Unmarshal(r, &a)

// 		fac := milestoneRegistry[a.Type]
// 		if fac == nil {
// 			return fmt.Errorf("milestone type not exists:%s", a.Type)
// 		}
// 		instance := fac()
// 		if err := json.Unmarshal([]byte(r), instance); err != nil {
// 			log.Println("error unmarsahl milestone: ", a.Type, instance, err)
// 			return err
// 		}

// 		list = append(list, instance)
// 	}

// 	*milestones = list

// 	return nil
// }

type PointMilestone struct {
	ID         int          `json:"ID"`
	PointID    int          `json:"pointID"`
	Point      int          `json:"point"`
	Milestones []*Milestone `json:"milestones"`
}

const pointMilestoneColumn = "milestone.id, milestone.point_id, milestone.point, milestone.milestones"

func pointMilestoneTableName(siteID string, traceParent bool) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_ROLE, traceParent)
	if err != nil {
		return "", err
	}
	return moduleSite + "_pointmilestone", nil
}

func (m *PointMilestone) scan(rows *sql.Rows) error {
	var milestones string
	if err := rows.Scan(&m.ID, &m.PointID, &m.Point, &milestones); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(milestones), &m.Milestones); err != nil {
		return err
	}
	return nil
}

func (m *PointMilestone) Add(siteID string) error {

	table, err := pointMilestoneTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Milestones == nil {
		m.Milestones = make([]*Milestone, 0)
	}
	milestones, _ := json.Marshal(m.Milestones)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(point_id, point, milestones)
		VALUES
			(?,?,?)
	`, table), m.PointID, m.Point, string(milestones)); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *PointMilestone) Update(siteID string) error {

	table, err := pointMilestoneTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Milestones == nil {
		m.Milestones = make([]*Milestone, 0)
	}
	milestones, _ := json.Marshal(m.Milestones)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			point=?, milestones=?
		WHERE
			id=?
	`, table), m.Point, string(milestones), m.ID); err != nil {
		return err
	}

	return nil
}
func (m *PointMilestone) Delete(siteID string) error {
	table, err := pointMilestoneTableName(siteID, true)
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

func GetPointMilestones(siteID string, pointID ...int) ([]*PointMilestone, error) {
	table, err := pointMilestoneTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(pointID) > 0 {
		if len(pointID) == 1 {
			whereStmts = append(whereStmts, "milestone.point_id=?")
			values = append(values, pointID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range pointID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("milestone.point_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s milestone
	`, pointMilestoneColumn, table)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY milestone.point ASC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	list := make([]*PointMilestone, 0)
	for rows.Next() {
		var m PointMilestone
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}

	return list, nil
}

func GetPointMilestonesWithTxn(siteID string, txn *sql.Tx, pointID ...int) ([]*PointMilestone, error) {
	table, err := pointMilestoneTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(pointID) > 0 {
		if len(pointID) == 1 {
			whereStmts = append(whereStmts, "milestone.point_id=?")
			values = append(values, pointID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range pointID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, "milestone.point_id IN (%s)", strings.Join(placeholder, ","))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s milestone
	`, pointMilestoneColumn, table)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY milestone.point ASC"

	rows, err := txn.Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	list := make([]*PointMilestone, 0)
	for rows.Next() {
		var m PointMilestone
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}

	return list, nil
}

// type HintMilestone struct {
// 	BaseMilestone
// }

// func (m *HintMilestone) Exec(string, *sql.Tx, int, bool) error { return nil }

// type UpgradeMilestone struct {
// 	BaseMilestone
// 	OriRoleID  int `json:"oriRoleID"`
// 	DestRoleID int `json:"destRoleID"`
// }

// func (m *UpgradeMilestone) Exec(siteID string, txn *sql.Tx, uid int, isRevert bool) error {

// 	if isRevert {
// 		if err := unbindUserRole(siteID, txn, uid, m.DestRoleID); err != nil {
// 			return err
// 		}

// 		if m.OriRoleID > 0 {
// 			if err := bindUserRole(siteID, txn, uid, m.OriRoleID); err != nil {
// 				return err
// 			}
// 		}

// 	} else {
// 		if m.OriRoleID > 0 {
// 			if err := unbindUserRole(siteID, txn, uid, m.OriRoleID); err != nil {
// 				return err
// 			}
// 		}

// 		if err := bindUserRole(siteID, txn, uid, m.DestRoleID); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
