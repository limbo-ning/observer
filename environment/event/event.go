package event

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
	INIT        = "INIT"
	IN_PROGRESS = "IN_PROGRESS"
	SUCCESS     = "SUCCESS"
	FAIL        = "FAIL"
)

type Event struct {
	ID                 int                    `json:"ID"`
	Type               string                 `json:"type"`
	Status             string                 `json:"status,omitempty"`
	MainRelateID       string                 `json:"mainRelateID"`
	SubRelateID        map[string]string      `json:"subRelateID,omitempty"`
	Ext                map[string]interface{} `json:"ext,omitempty"`
	MaxExecuteDuration util.Duration          `json:"maxExecuteDuration,omitempty"`
	CreateTime         *util.Time             `json:"createTime,omitempty"`
	FinishTime         *util.Time             `json:"finishTime,omitempty"`
}

const eventColumn = "event.id, event.type, event.status, event.main_relate_id, event.sub_relate_id, event.ext, event.max_execute_duration, event.create_time, event.finish_time"

func eventTableName(siteID string) string {
	return siteID + "_event"
}

func (e *Event) scan(rows *sql.Rows) error {

	var subRelateID, ext string

	e.CreateTime = new(util.Time)
	e.FinishTime = new(util.Time)
	if err := rows.Scan(&e.ID, &e.Type, &e.Status, &e.MainRelateID, &subRelateID, &ext, &e.MaxExecuteDuration, e.CreateTime, e.FinishTime); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(subRelateID), &e.SubRelateID); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(ext), &e.Ext); err != nil {
		return err
	}

	return nil
}

func GetEvents(siteID string, actionAuth authority.ActionAuthSet, eventType, status string, beginTime, endTime, effectTime *time.Time, pageNo, pageSize int, mainRelateID string, subRelateID map[string]string, authType, empower string, empowerID []string, eventID ...int) ([]*Event, int, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s event
	`, eventTableName(siteID))
	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s event
	`, eventColumn, eventTableName(siteID))

	authCheck := true
	for _, a := range actionAuth {
		switch a.Action {
		case ACTION_ADMIN_VIEW:
			authCheck = false
		}
		if !authCheck {
			break
		}
	}

	if authCheck || empower != "" || authType != "" {
		if authType == "" {
			authType = ACTION_VIEW
		}
		authSQL, authWhere, authValues, err := authority.JoinEmpower(siteID, "event", actionAuth, AdminActions, authType, "event", "type", empower, empowerID...)
		if err != nil {
			return nil, 0, err
		}

		countSQL += authSQL
		SQL += authSQL

		whereStmts = append(whereStmts, authWhere...)
		values = append(values, authValues...)
	}

	if len(eventID) > 0 {
		pageSize = -1
		if len(eventID) == 1 {
			whereStmts = append(whereStmts, "event.id = ?")
			values = append(values, eventID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range eventID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("event.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if eventType != "" {
		whereStmts = append(whereStmts, "event.type = ?")
		values = append(values, eventType)
	}

	if status != "" {
		whereStmts = append(whereStmts, "event.status = ?")
		values = append(values, status)
	}

	if mainRelateID != "" {
		whereStmts = append(whereStmts, "event.main_relate_id = ?")
		values = append(values, mainRelateID)
	}

	if len(subRelateID) > 0 {
		for key, value := range subRelateID {
			whereStmts = append(whereStmts, fmt.Sprintf("JSON_EXTRACT(event.sub_relat_id, '$.%s') = ?", key))
			values = append(values, value)
		}
	}

	if beginTime != nil && !beginTime.IsZero() {
		whereStmts = append(whereStmts, "event.create_time >= ?")
		values = append(values, beginTime)
	}

	if endTime != nil && !endTime.IsZero() {
		whereStmts = append(whereStmts, "event.create_time <= ?")
		values = append(values, endTime)
	}

	if effectTime != nil && !effectTime.IsZero() {
		whereStmts = append(whereStmts, "event.create_time <= ?", "event.finish_time >= ?")
		values = append(values, effectTime, effectTime)
	}

	if len(whereStmts) > 0 {
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	total := 0
	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count event: ", countSQL, values, err)
		return nil, 0, err
	}

	SQL += "\nORDER BY event.id DESC"

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
		log.Println("error get event: ", SQL, values, err)
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]*Event, 0)

	for rows.Next() {
		var e Event
		if err := e.scan(rows); err != nil {
			return nil, 0, err
		}
		result = append(result, &e)
	}
	return result, total, nil
}

func GetEventWithTxn(siteID string, txn *sql.Tx, forUpdate bool, eventID ...int) ([]*Event, error) {

	result := make([]*Event, 0)

	if len(eventID) == 0 {
		return result, nil
	}

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(eventID) == 0 {
		whereStmts = append(whereStmts, "event.id = ?")
		values = append(values, eventID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range eventID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("event.id IN (%s)", strings.Join(placeholder, ",")))
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s event
		WHERE
			%s
	`, eventColumn, eventTableName(siteID), strings.Join(whereStmts, " AND "))

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, values...)
	if err != nil {
		log.Println("error get events: ", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var e Event
		if err := e.scan(rows); err != nil {
			log.Println("error get events: ", err)
			return nil, err
		}
		result = append(result, &e)
	}

	return result, nil
}

func CloneEvent(siteID string, actionAuth authority.ActionAuthSet, eventID int) (result *Event, err error) {
	err = datasource.Txn(func(txn *sql.Tx) {

		events, err := GetEventWithTxn(siteID, txn, false, eventID)
		if err != nil {
			panic(err)
		}

		if len(events) == 0 {
			panic(errors.New("事件不存在"))
		}

		e := events[0]

		if err := CheckAuth(siteID, actionAuth, e.Type, ACTION_EDIT); err != nil {
			panic(err)
		}

		result, err = e.clone(siteID, txn)
		if err != nil {
			panic(err)
		}
	})

	if err != nil {
		return nil, err
	}

	return result, err
}

func (e *Event) Validate(siteID string) (IEvent, error) {
	iEvent, err := GetEvent(e.Type)
	if err != nil {
		return nil, err
	}

	if err := iEvent.ValidateEvent(siteID, e); err != nil {
		return nil, err
	}

	if e.SubRelateID == nil {
		e.SubRelateID = make(map[string]string)
	}

	if e.Ext == nil {
		e.Ext = make(map[string]interface{})
	}

	if e.MaxExecuteDuration.GetDuration() == 0 {
		e.MaxExecuteDuration = util.Duration("1m")
	}

	if e.FinishTime == nil {
		e.FinishTime = new(util.Time)
	}

	return iEvent, nil
}

func (e *Event) clone(siteID string, txn *sql.Tx) (*Event, error) {
	result := new(Event)

	if err := util.Clone(e, result); err != nil {
		return nil, err
	}

	delete(result.Ext, INIT)
	delete(result.Ext, IN_PROGRESS)
	delete(result.Ext, SUCCESS)
	delete(result.Ext, FAIL)

	if err := result.add(siteID, txn); err != nil {
		return nil, err
	}

	return result, nil
}

func (e *Event) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := CheckAuth(siteID, actionAuth, e.Type, ACTION_EDIT); err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		if err := e.add(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func (e *Event) add(siteID string, txn *sql.Tx) error {

	iEvent, err := e.Validate(siteID)
	if err != nil {
		return err
	}

	e.Status = INIT
	if e.SubRelateID == nil {
		e.SubRelateID = make(map[string]string)
	}
	subRelateID, _ := json.Marshal(e.SubRelateID)
	if e.Ext == nil {
		e.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(e.Ext)

	if e.MaxExecuteDuration.GetDuration() == 0 {
		e.MaxExecuteDuration = util.Duration("1m")
	}

	if e.FinishTime == nil {
		e.FinishTime = new(util.Time)
	}
	finishTime := time.Now().Add(e.MaxExecuteDuration.GetDuration())
	*e.FinishTime = util.Time(finishTime)

	var ret sql.Result

	SQL := fmt.Sprintf(`
		INSERT INTO %s
			(type, status, main_relate_id, sub_relate_id, ext, max_execute_duration, finish_time)
		VALUES
			(?,?,?,?,?,?,?)
	`, eventTableName(siteID))

	values := []interface{}{e.Type, e.Status, e.MainRelateID, string(subRelateID), string(ext), string(e.MaxExecuteDuration), finishTime}

	ret, err = txn.Exec(SQL, values...)
	if err != nil {
		log.Println("error add event: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add event: ", err)
		return err
	} else {
		e.ID = int(id)
	}

	var onErr = func(err error) {
		e.Feedback(FAIL, map[string]interface{}{
			"at":  util.Time(time.Now()),
			"msg": err.Error(),
		})
		if err := e.UpdateStatusWithTxn(siteID, txn); err != nil {
			log.Println("error update after execute: ", err)
		}
	}

	defer func() {
		if e := recover(); e != nil {
			log.Println("error execute event recover: ", e)
			onErr(fmt.Errorf("严重错误: %v", e))
		}
	}()

	if err := iEvent.ExecuteEvent(siteID, txn, e); err != nil {
		log.Println("error execute event: ", err)
		onErr(err)
		return nil
	}

	return nil
}

func (e *Event) UpdateStatusWithTxn(siteID string, txn *sql.Tx) error {

	switch e.Status {
	case INIT:
	case IN_PROGRESS:
	case SUCCESS:
	case FAIL:
	default:
		e.Status = INIT
	}
	if e.Ext == nil {
		e.Ext = make(map[string]interface{})
	}
	ext, _ := json.Marshal(e.Ext)

	if e.FinishTime == nil {
		e.FinishTime = new(util.Time)
	}
	finishTime := time.Time(*e.FinishTime)

	SQL := fmt.Sprintf(`
		UPDATE
			%s
		SET
			status=?, ext=?, finish_time=?
		WHERE
			id = ?
	`, eventTableName(siteID))

	values := []interface{}{e.Status, string(ext), finishTime, e.ID}

	var err error
	if txn == nil {
		_, err = datasource.GetConn().Exec(SQL, values...)
	} else {
		_, err = txn.Exec(SQL, values...)
	}
	if err != nil {
		log.Println("error update event: ", err)
		return err
	}

	go BroadcastEvent(siteID, e)

	return nil
}

func (e *Event) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	for _, a := range actionAuth {
		switch a.Action {
		case ACTION_ADMIN_EDIT:
			SQL := fmt.Sprintf(`
				DELETE FROM
					%s
				WHERE
					id = ?
			`, eventTableName(siteID))

			_, err := datasource.GetConn().Exec(SQL, e.ID)
			if err != nil {
				log.Println("error delete event: ", err)
				return err
			}

			return nil
		}
	}

	return errors.New("无删除权限")
}

func (e *Event) Feedback(status string, feedback map[string]interface{}) {

	e.Status = status

	feedbacks, exists := e.Ext[status]
	if !exists {
		feedbacks = make([]map[string]interface{}, 0)
	}

	e.Ext[status] = append(feedbacks.([]map[string]interface{}), feedback)

	switch e.Status {
	case SUCCESS:
		fallthrough
	case FAIL:
		if e.FinishTime == nil {
			e.FinishTime = new(util.Time)
		}
		*e.FinishTime = util.Time(time.Now())
	}
}
