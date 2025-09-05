package role

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/site"
)

var e_user_point_not_exists = errors.New("用户会员积分不存在")

type UserPoint struct {
	UID     int `json:"UID"`
	PointID int `json:"pointID"`
	Point   int `json:"point"`
}

const userPointColumn = "userPoint.uid, userPoint.point_id, userPoint.point"

func userPointTableName(siteID string, traceParent bool) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_ROLE, traceParent)
	if err != nil {
		return "", err
	}
	return moduleSite + "_userpoint", nil
}

func (m *UserPoint) scan(rows *sql.Rows) error {
	if err := rows.Scan(&m.UID, &m.PointID, &m.Point); err != nil {
		return err
	}
	return nil
}

func (m *UserPoint) add(siteID string, txn *sql.Tx) error {
	table, err := userPointTableName(siteID, true)
	if err != nil {
		return err
	}
	if _, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(uid, point_id, point)
		VALUES
			(?,?,?)
	`, table), m.UID, m.PointID, m.Point); err != nil {
		return err
	}
	return nil
}
func (m *UserPoint) update(siteID string, txn *sql.Tx) error {
	table, err := userPointTableName(siteID, true)
	if err != nil {
		return err
	}
	if _, err := txn.Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			point=?
		WHERE
			uid=? AND point_id=?
	`, table), m.Point, m.UID, m.PointID); err != nil {
		return err
	}
	return nil
}
func (m *UserPoint) delete(siteID string, txn *sql.Tx) error {
	table, err := userPointTableName(siteID, true)
	if err != nil {
		return err
	}
	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			uid=? AND point_id=?
	`, table), m.Point, m.UID, m.PointID); err != nil {
		return err
	}
	return nil
}

func GetUserPoints(siteID string, uid ...int) (map[int]map[int]int, error) {

	result := make(map[int]map[int]int)
	if len(uid) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(uid) == 1 {
		whereStmts = append(whereStmts, "uid = ?")
		values = append(values, uid[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range uid {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("uid IN (%s)", strings.Join(placeholder, ",")))
	}

	table, err := userPointTableName(siteID, true)
	if err != nil {
		return nil, err
	}
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s userPoint
		WHERE
			%s
	`, userPointColumn, table, strings.Join(whereStmts, " AND ")), values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m UserPoint
		if err := m.scan(rows); err != nil {
			return nil, err
		}

		if _, exists := result[m.UID]; !exists {
			result[m.UID] = make(map[int]int)
		}
		result[m.UID][m.PointID] = m.Point
	}

	return result, nil
}

func GetOrCreateUserPointWithTxn(siteID string, txn *sql.Tx, uid, pointID int) (*UserPoint, error) {

	result, err := getUserPointWithTxn(siteID, txn, uid, pointID, true)
	if err == nil {
		return result, nil
	}

	if err != e_user_point_not_exists {
		return nil, err
	}

	m := new(UserPoint)
	m.UID = uid
	m.PointID = pointID

	if err := m.add(siteID, txn); err != nil {
		return nil, err
	}

	return m, nil
}

func getUserPointWithTxn(siteID string, txn *sql.Tx, uid, pointID int, forUpdate bool) (*UserPoint, error) {
	table, err := userPointTableName(siteID, true)
	if err != nil {
		return nil, err
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s userPoint
		WHERE
			uid = ? AND point_id = ?
	`, userPointColumn, table)

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, uid, pointID)
	if err != nil {
		log.Println("error get balance account with lock: ", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var m UserPoint
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		return &m, nil
	}

	return nil, e_user_point_not_exists
}

type UserPointFlow struct {
	ID          int                    `json:"ID"`
	UID         int                    `json:"UID"`
	PointID     int                    `json:"pointID"`
	Amount      int                    `json:"amount"`
	Type        string                 `json:"type"`
	Incident    map[string]interface{} `json:"incident"`
	Description string                 `json:"description"`
	CreateTime  util.Time              `json:"createTime"`
	UpdateTime  util.Time              `json:"updateTime"`
}

const userPointFlowColumn = "userPointFlow.id, userPointFlow.uid, userPointFlow.point_id, userPointFlow.amount, userPointFlow.type, userPointFlow.incident, userPointFlow.description, userPointFlow.create_time, userPointFlow.update_time"

func userPointFlowTableName(siteID string, traceParent bool) (string, error) {
	moduleSite, _, err := site.GetSiteModule(siteID, MODULE_ROLE, traceParent)
	if err != nil {
		return "", err
	}
	return moduleSite + "_userpointflow", nil
}

func (m *UserPointFlow) scan(rows *sql.Rows) error {
	var incident string
	if err := rows.Scan(&m.ID, &m.UID, &m.PointID, &m.Amount, &m.Type, &incident, &m.Description, &m.CreateTime, &m.UpdateTime); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(incident), &m.Incident); err != nil {
		return err
	}
	return nil
}

func (m *UserPointFlow) add(siteID string, txn *sql.Tx) error {
	table, err := userPointFlowTableName(siteID, true)
	if err != nil {
		return err
	}
	if m.Incident == nil {
		m.Incident = make(map[string]interface{})
	}
	incident, _ := json.Marshal(m.Incident)
	if _, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(uid, point_id, amount, type, incident, description)
		VALUES
			(?,?,?,?,?,?)
	`, table), m.UID, m.PointID, m.Amount, m.Type, string(incident), m.Description); err != nil {
		return err
	}
	return nil
}

func ChangeUserPoint(siteID string, txn *sql.Tx, uid, pointID, amount int, flowType, description string, incident map[string]interface{}, isRevert bool) error {

	if amount == 0 {
		return nil
	}

	userPoint, err := GetOrCreateUserPointWithTxn(siteID, txn, uid, pointID)
	if err != nil {
		return err
	}

	if userPoint.Point+amount < 0 && !isRevert {
		return errors.New("积分不足")
	}

	point, err := GetPointWithTxn(siteID, txn, pointID, false)
	if err != nil {
		return err
	}

	if err := point.milestone(siteID, txn, uid, userPoint.Point, amount, isRevert); err != nil {
		return err
	}

	userPoint.Point += amount

	if err := userPoint.update(siteID, txn); err != nil {
		return err
	}

	flow := new(UserPointFlow)
	flow.UID = uid
	flow.PointID = pointID
	flow.Amount = amount
	flow.Type = flowType
	flow.Description = description
	flow.Incident = incident

	if err := flow.add(siteID, txn); err != nil {
		return err
	}

	return nil
}
