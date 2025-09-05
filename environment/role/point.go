package role

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/mission"
	"obsessiontech/environment/site"
)

var e_point_not_exists = errors.New("会员积分不存在")

type Point struct {
	ID          int                 `json:"ID"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Profile     map[string][]string `json:"profile"`
	Sort        int                 `json:"sort"`
}

const pointColumn = "point.id, point.name, point.description, point.profile, point.sort"

func pointTableName(siteID string, traceParent bool) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_ROLE, traceParent)
	if err != nil {
		return "", err
	}
	return moduleSite + "_point", nil
}

func (m *Point) scan(rows *sql.Rows) error {
	var profile string
	if err := rows.Scan(&m.ID, &m.Name, &m.Description, &profile, &m.Sort); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(profile), &m.Profile); err != nil {
		return err
	}
	return nil
}

func (m *Point) Add(siteID string) error {

	table, err := pointTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Profile == nil {
		m.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(m.Profile)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name, description, profile, sort)
		VALUES
			(?,?,?,?)
	`, table), m.Name, m.Description, string(profile), m.Sort); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}
func (m *Point) Update(siteID string) error {

	table, err := pointTableName(siteID, true)
	if err != nil {
		return err
	}

	if m.Profile == nil {
		m.Profile = make(map[string][]string)
	}
	profile, _ := json.Marshal(m.Profile)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?, description=?, profile=?, sort=?
		WHERE
			id=?
	`, table), m.Name, m.Description, string(profile), m.Sort, m.ID); err != nil {
		return err
	}

	return nil
}
func (m *Point) Delete(siteID string) error {
	table, err := pointTableName(siteID, true)
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

func (m *Point) milestone(siteID string, txn *sql.Tx, uid, previousPoint, change int, isRevert bool) error {

	if change == 0 {
		return nil
	}

	milestones, err := GetPointMilestonesWithTxn(siteID, txn, m.ID)
	if err != nil {
		return err
	}

	if change > 0 {

		for _, milestone := range milestones {
			if previousPoint < milestone.Point {
				if previousPoint+change >= milestone.Point {
					for _, milestone := range milestone.Milestones {

						var completes map[int][]*mission.Complete
						if isRevert {
							completes, err = mission.GetCompletes(siteID, uid, []string{mission.COMPLETE_FINAL}, nil, "", milestone.MissionIDs...)
							if err != nil {
								return err
							}
						}

						for _, mid := range milestone.MissionIDs {
							if isRevert {
								list, exists := completes[mid]
								if exists && len(list) > 0 {
									if err := mission.RevertMissionComplete(siteID, authority.ActionAuthSet{{UID: uid, Action: mission.ACTION_ADMIN_COMPLETE}}, list[0].ID); err != nil {
										log.Println("error revert milestone mission: UID-", uid, mid, list[0].ID, err)
									}
								}
							} else {
								if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: uid, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
									log.Println("error complete milestone mission: UID-", uid, mid, err)
								}
							}
						}
					}
				} else {
					break
				}
			}
		}
	} else {

		isRevert = !isRevert

		for i := len(milestones) - 1; i >= 0; i-- {
			milestone := milestones[i]
			if previousPoint >= milestone.Point {
				if previousPoint+change < milestone.Point {
					for _, milestone := range milestone.Milestones {
						var completes map[int][]*mission.Complete
						if isRevert {
							completes, err = mission.GetCompletes(siteID, uid, []string{mission.COMPLETE_FINAL}, nil, "", milestone.MissionIDs...)
							if err != nil {
								return err
							}
						}

						for _, mid := range milestone.MissionIDs {
							if isRevert {
								list, exists := completes[mid]
								if exists && len(list) > 0 {
									if err := mission.RevertMissionComplete(siteID, authority.ActionAuthSet{{UID: uid, Action: mission.ACTION_ADMIN_COMPLETE}}, list[0].ID); err != nil {
										log.Println("error revert milestone mission: UID-", uid, mid, list[0].ID, err)
									}
								}
							} else {
								if _, err := mission.CompleteMission(siteID, authority.ActionAuthSet{{UID: uid, Action: mission.ACTION_ADMIN_COMPLETE}}, mid, nil, time.Now()); err != nil {
									log.Println("error complete milestone mission: UID-", uid, mid, err)
								}
							}
						}
					}
				} else {
					break
				}
			}
		}
	}

	return nil
}

func GetPoints(siteID string, pointID ...int) ([]*Point, map[int][]*PointMilestone, error) {
	table, err := pointTableName(siteID, true)
	if err != nil {
		return nil, nil, err
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(pointID) > 0 {
		if len(pointID) == 1 {
			whereStmts = append(whereStmts, "point.id=?")
			values = append(values, pointID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range pointID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("point.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s point
	`, pointColumn, table)

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY point.sort DESC"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	result := make([]*Point, 0)
	pointIDs := make([]int, 0)

	for rows.Next() {
		var m Point
		if err := m.scan(rows); err != nil {
			return nil, nil, err
		}
		result = append(result, &m)
		pointIDs = append(pointIDs, m.ID)
	}

	milestones := make(map[int][]*PointMilestone)

	milestoneList, err := GetPointMilestones(siteID, pointIDs...)
	if err != nil {
		return nil, nil, err
	}

	for _, milestone := range milestoneList {
		if _, exists := milestones[milestone.PointID]; !exists {
			milestones[milestone.PointID] = make([]*PointMilestone, 0)
		}
		milestones[milestone.PointID] = append(milestones[milestone.PointID], milestone)
	}

	return result, milestones, nil
}

func GetPoint(siteID string, pointID int) (*Point, error) {
	table, err := pointTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s point
		WHERE
			id = ?
	`, pointColumn, table)

	rows, err := datasource.GetConn().Query(SQL, pointID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var m Point
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		return &m, nil
	}

	return nil, e_point_not_exists
}

func GetPointWithTxn(siteID string, txn *sql.Tx, pointID int, forUpdate bool) (*Point, error) {

	table, err := pointTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s point
		WHERE
			id = ?
	`, pointColumn, table)

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, pointID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var m Point
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		return &m, nil
	}

	return nil, e_point_not_exists
}
