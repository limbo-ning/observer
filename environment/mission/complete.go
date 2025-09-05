package mission

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
)

const (
	COMPLETE_REVERT  = "REVERT"
	COMPLETE_FINAL   = "FINAL"
	COMPLETE_PENDING = "PENDING"
	COMPLETE_CLOSED  = "CLOSED"
)

type Complete struct {
	ID           int                    `json:"ID"`
	UID          int                    `json:"UID"`
	MissionID    int                    `json:"missionID"`
	Status       string                 `json:"status"`
	Section      string                 `json:"section"`
	Result       map[string]interface{} `json:"result"`
	Ext          map[string]interface{} `json:"ext"`
	CompleteTime util.Time              `json:"completeTime"`
}

const completeColumn = "missioncomplete.id,missioncomplete.uid,missioncomplete.mission_id,missioncomplete.status,missioncomplete.result,missioncomplete.ext,missioncomplete.complete_time"

func completeTable(siteID string) string {
	return siteID + "_missioncomplete"
}

func (c *Complete) scan(rows *sql.Rows) error {
	var result, ext string
	if err := rows.Scan(&c.ID, &c.UID, &c.MissionID, &c.Status, &result, &ext, &c.CompleteTime); err != nil {
		log.Println("error scan mission complete: ", err)
		return err
	}
	if err := json.Unmarshal([]byte(result), &c.Result); err != nil {
		log.Println("error scan mission complete: ", err)
		return err
	}
	if err := json.Unmarshal([]byte(ext), &c.Ext); err != nil {
		log.Println("error scan mission complete: ", err)
		return err
	}
	return nil
}

func GetCompletes(siteID string, uid int, status []string, targetTime *time.Time, missionType string, missionID ...int) (map[int][]*Complete, error) {
	missionList, err := getMissionsWithTxn(siteID, nil, false, missionType, missionID...)
	if err != nil {
		return nil, err
	}
	missions := make(map[int]*Mission)
	targetTimes := make(map[int][]*time.Time)
	if targetTime != nil {
		for _, m := range missionList {
			missions[m.ID] = m
			targetTimes[m.ID] = []*time.Time{targetTime}
		}
	}
	return getCompletes(siteID, nil, uid, status, missions, targetTimes, false)
}

func buildMissionCompleteCriteria(missions map[int]*Mission, missionSectionTime map[int][]*time.Time) ([]string, []interface{}, error) {
	ors := make([]string, 0)
	orValues := make([]interface{}, 0)
	for mid, mission := range missions {
		targetTimes, exists := missionSectionTime[mid]
		if exists && len(targetTimes) > 0 {
			for _, targetTime := range targetTimes {
				if targetTime == nil {
					break
				}

				sectionStart, sectionEnd, err := mission.Section.GetInterval(*targetTime, true)
				if err != nil {
					if err == util.E_not_in_interval {
						break
					} else {
						return nil, nil, err
					}
				}

				missionSQL := "missioncomplete.mission_id = ?"
				orValues = append(orValues, mid)
				if sectionStart != nil {
					missionSQL += " AND missioncomplete.complete_time >= ?"
					orValues = append(orValues, sectionStart)
				}
				if sectionEnd != nil {
					missionSQL += " AND missioncomplete.complete_time <= ?"
					orValues = append(orValues, sectionEnd)
				}

				ors = append(ors, "("+missionSQL+")")
			}
		} else {
			ors = append(ors, "missioncomplete.mission_id = ?")
			orValues = append(orValues, mid)
		}
	}

	return ors, orValues, nil
}

func CountCompletes(siteID string, uid int, status []string, targetTime time.Time, includeBeforeSection bool, missionType string, missionID ...int) (map[int]map[string]int, error) {

	missionList, err := getMissionsWithTxn(siteID, nil, false, missionType, missionID...)
	if err != nil {
		return nil, err
	}

	result := make(map[int]map[string]int)

	if len(missionList) == 0 {
		return result, nil
	}

	missions := make(map[int]*Mission)
	targetTimes := make(map[int][]*time.Time)
	for _, m := range missionList {
		missions[m.ID] = m
		targetTimes[m.ID] = []*time.Time{&targetTime}
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if uid >= 0 {
		whereStmts = append(whereStmts, "missioncomplete.uid = ?")
		values = append(values, uid)
	}

	if len(status) > 0 {
		if len(status) == 1 {
			whereStmts = append(whereStmts, "missioncomplete.status = ?")
			values = append(values, status[0])
		} else {
			placeholder := make([]string, 0)
			for _, s := range status {
				placeholder = append(placeholder, "?")
				values = append(values, s)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("missioncomplete.status IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	ors, orValues, err := buildMissionCompleteCriteria(missions, targetTimes)
	if err != nil {
		return nil, err
	}

	if len(ors) > 0 {
		whereStmts = append(whereStmts, "("+strings.Join(ors, " OR ")+")")
	}

	values = append(values, orValues...)

	SQL := fmt.Sprintf(`
		SELECT
			missioncomplete.mission_id, missioncomplete.status, COUNT(1)
		FROM
			%s missioncomplete
	`, completeTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nGROUP BY missioncomplete.mission_id, missioncomplete.status"

	var rows *sql.Rows
	rows, err = datasource.GetConn().Query(SQL, values...)

	if err != nil {
		log.Println("error count mission complete: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mid, count int
		var status string
		if err := rows.Scan(&mid, &status, &count); err != nil {
			return nil, err
		}

		if _, exists := result[mid]; !exists {
			result[mid] = make(map[string]int)
		}

		result[mid][status] = count
	}

	return result, nil
}

func countCompletes(siteID string, txn *sql.Tx, uid int, status []string, missions map[int]*Mission, missionSectionTime map[int][]*time.Time, includeBeforeSection, forUpdate bool) (map[int]map[string][]*time.Time, error) {
	result := make(map[int]map[string][]*time.Time)

	if len(missionSectionTime) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if uid >= 0 {
		whereStmts = append(whereStmts, "missioncomplete.uid = ?")
		values = append(values, uid)
	}

	if len(status) > 0 {
		if len(status) == 1 {
			whereStmts = append(whereStmts, "missioncomplete.status = ?")
			values = append(values, status[0])
		} else {
			placeholder := make([]string, 0)
			for _, s := range status {
				placeholder = append(placeholder, "?")
				values = append(values, s)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("missioncomplete.status IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	ors, orValues, err := buildMissionCompleteCriteria(missions, missionSectionTime)
	if err != nil {
		return nil, err
	}

	if len(ors) > 0 {
		whereStmts = append(whereStmts, "("+strings.Join(ors, " OR ")+")")
	}

	values = append(values, orValues...)

	SQL := fmt.Sprintf(`
		SELECT
			missioncomplete.mission_id, missioncomplete.status, missioncomplete.complete_time
		FROM
			%s missioncomplete
	`, completeTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	var rows *sql.Rows
	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	}

	if err != nil {
		log.Println("error count mission complete: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var mid int
		var completeTime time.Time
		if err := rows.Scan(&mid, &status, &completeTime); err != nil {
			return nil, err
		}

		statusMap, exists := result[mid]
		if !exists {
			statusMap = make(map[string][]*time.Time)
		}
		statusMap[status] = append(statusMap[status], &completeTime)

		result[mid] = statusMap
	}

	return result, nil
}

func getCompletes(siteID string, txn *sql.Tx, uid int, status []string, missions map[int]*Mission, missionSectionTime map[int][]*time.Time, forUpdate bool) (map[int][]*Complete, error) {

	result := make(map[int][]*Complete)

	if len(missions) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if uid != -1 {
		whereStmts = append(whereStmts, "missioncomplete.uid = ?")
		values = append(values, uid)
	}

	if len(status) > 0 {
		if len(status) == 1 {
			whereStmts = append(whereStmts, "missioncomplete.status = ?")
			values = append(values, status[0])
		} else {
			placeholder := make([]string, 0)
			for _, s := range status {
				placeholder = append(placeholder, "?")
				values = append(values, s)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("missioncomplete.status IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	ors, orValues, err := buildMissionCompleteCriteria(missions, missionSectionTime)
	if err != nil {
		return nil, err
	}

	if len(ors) > 0 {
		whereStmts = append(whereStmts, "("+strings.Join(ors, " OR ")+")")
	}

	values = append(values, orValues...)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s missioncomplete
	`, completeColumn, completeTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY missioncomplete.id DESC"

	var rows *sql.Rows
	if txn == nil {
		rows, err = datasource.GetConn().Query(SQL, values...)
	} else {
		if forUpdate {
			SQL += "\nFOR UPDATE"
		}
		rows, err = txn.Query(SQL, values...)
	}

	if err != nil {
		log.Println("error get mission complete: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Complete
		if err := c.scan(rows); err != nil {
			return nil, err
		}

		list, exists := result[c.MissionID]
		if !exists {
			list = make([]*Complete, 0)
		}
		result[c.MissionID] = append(list, &c)
	}

	return result, nil
}

func getComplete(siteID string, txn *sql.Tx, forUpdate bool, completeID ...int) ([]*Complete, error) {

	result := make([]*Complete, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(completeID) == 0 {
		return result, nil
	} else if len(completeID) == 1 {
		whereStmts = append(whereStmts, "missioncomplete.id = ?")
		values = append(values, completeID[0])
	} else {
		placeHolder := make([]string, 0)
		for _, id := range completeID {
			placeHolder = append(placeHolder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("missioncomplete.id IN (%s)", strings.Join(placeHolder, ",")))
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s missioncomplete
	`, completeColumn, completeTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

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
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Complete
		if err := c.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &c)
	}

	return result, nil
}

func (c *Complete) add(siteID string, txn *sql.Tx) error {
	table := completeTable(siteID)

	result, err := json.Marshal(c.Result)
	if err != nil {
		return err
	}

	ext, err := json.Marshal(c.Ext)
	if err != nil {
		return err
	}

	if ret, err := txn.Exec(fmt.Sprintf(`
		INSERT INTO %s
			(uid,mission_id,status,result,ext,complete_time)
		VALUES
			(?,?,?,?,?,?)
	`, table), c.UID, c.MissionID, c.Status, string(result), string(ext), time.Time(c.CompleteTime)); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add complete: ", err)
		return err
	} else {
		c.ID = int(id)
	}

	return nil
}

func (c *Complete) update(siteID string, txn *sql.Tx) error {
	table := completeTable(siteID)

	result, err := json.Marshal(c.Result)
	if err != nil {
		return err
	}

	ext, err := json.Marshal(c.Ext)
	if err != nil {
		log.Println("error marshal complete ext: ", err)
		return err
	}

	if ret, err := txn.Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			status=?,result=?,ext=?,complete_time=?
		WHERE
			id=?
	`, table), c.Status, string(result), string(ext), time.Time(c.CompleteTime), c.ID); err != nil {
		log.Println("error update complete: ", err)
		return err
	} else if affected, err := ret.RowsAffected(); err != nil {
		log.Println("warning update complete: fail to get rows affected:", c.ID, err)
	} else if affected != 1 {
		log.Println("warning update complete: no change: ", c.ID)
	}

	return nil
}
func (c *Complete) delete(siteID string, txn *sql.Tx) error {
	table := completeTable(siteID)

	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM 
			%s
		WHERE
			id = ?
	`, table), c.ID); err != nil {
		return err
	}

	return nil
}

func RevertMissionComplete(siteID string, actionAuth authority.ActionAuthSet, completeID int) error {
	return datasource.Txn(func(txn *sql.Tx) {
		if err := RevertMissionCompleteWithTxn(siteID, txn, actionAuth, completeID); err != nil {
			panic(err)
		}
	})
}

func RevertMissionCompleteWithTxn(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, completeID int) error {
	log.Println("revert complete: ", completeID)

	completes, err := getComplete(siteID, txn, true, completeID)
	if err != nil {
		log.Println("error get complete to revert: ", err)
		return err
	}

	if len(completes) == 0 {
		log.Println("error get complete to revert: not found ", completeID)
		return errors.New("找不到记录")
	}

	complete := completes[0]

	if complete.Status == COMPLETE_REVERT {
		log.Println("warnning complete already reverted: ", complete.ID)
		return nil
	}

	missionList, err := getMissionsWithTxn(siteID, txn, false, "", complete.MissionID)
	if err != nil {
		return err
	}

	if len(missionList) == 0 {
		log.Println("error revert complete: mission not found ", completeID, complete.MissionID)
		return E_mission_not_found
	}

	mission := missionList[0]

	if err := revertMissionComplete(siteID, txn, actionAuth, mission, complete); err != nil {
		return err
	}

	return nil
}

func revertMissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	log.Println("revert complete: ", complete.ID)
	for _, c := range mission.Completes {
		if err := c.MissionRevert(siteID, txn, actionAuth, mission, complete); err != nil {
			if err != E_mission_interrupt {
				log.Println("error do mission revert: ", c.GetType(), err)
				return err
			}
			log.Println("do mission revert interrupt: ", c.GetType(), err)
			break
		}
	}

	complete.Status = COMPLETE_REVERT

	if err := complete.update(siteID, txn); err != nil {
		return err
	}

	return nil
}
