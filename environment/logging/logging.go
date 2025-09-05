package logging

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
	"obsessiontech/environment/user"
)

var e_log_incomplete = errors.New("记录信息不全")

type Logging struct {
	ID         int         `json:"ID"`
	UID        int         `json:"UID"`
	ModuleID   string      `json:"moduleID"`
	Source     string      `json:"source"`
	SourceID   string      `json:"sourceID"`
	Action     string      `json:"action"`
	Payload    interface{} `json:"payload"`
	CreateTime util.Time   `json:"createTime"`
}

const loggingColumns = "logging.id,logging.uid,logging.module_id,logging.source,logging.source_id,logging.action,logging.payload,logging.create_time"

func loggingTableName(siteID string) string {
	return siteID + "_logging"
}

func (l *Logging) scan(rows *sql.Rows) error {
	var payload string
	if err := rows.Scan(&l.ID, &l.UID, &l.ModuleID, &l.Source, &l.SourceID, &l.Action, &payload, &l.CreateTime); err != nil {
		return err
	}
	l.Payload = make(map[string]interface{})

	if err := json.Unmarshal([]byte(payload), &l.Payload); err != nil {
		return err
	}
	return nil
}

func Log(siteID string, uid int, moduleID, source, sourceID, action string, payload interface{}) error {

	module, err := GetModule(siteID)
	if err != nil {
		log.Println("error get logging module: ", siteID, err)
		return err
	}

	if !module.ShouldLog(moduleID, source, action) {
		log.Println("should not log: ", siteID, moduleID, source, sourceID, action)
		return nil
	}

	var l Logging

	l.UID = uid
	l.ModuleID = moduleID
	l.Source = source
	l.SourceID = sourceID
	l.Action = action
	l.Payload = payload

	return l.add(siteID)
}

func (l *Logging) add(siteID string) error {
	if l.Source == "" || l.SourceID == "" || l.Action == "" {
		return e_log_incomplete
	}

	if l.Payload == nil {
		l.Payload = make(map[string]interface{})
	}

	payload, _ := json.Marshal(l.Payload)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(uid,module_id,source,source_id,action,payload)
		VALUES
			(?,?,?,?,?,?)
	`, loggingTableName(siteID)), l.UID, l.ModuleID, l.Source, l.SourceID, l.Action, string(payload)); err != nil {
		log.Println("error add logging: ", err)
		return err
	}

	return nil
}

func GetLoggings(siteID, moduleID, source, sourceID, action string, uid int, beginTime, endTime *time.Time, pageNo, pageSize int) ([]*Logging, map[int]*user.UserBrief, int, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if moduleID != "" {
		whereStmts = append(whereStmts, "logging.module_id=?", "logging.source=?")
		values = append(values, moduleID, source)
	}

	if source != "" {
		whereStmts = append(whereStmts, "logging.source=?")
		values = append(values, source)
	}

	if sourceID != "" {
		whereStmts = append(whereStmts, "logging.source_id=?")
		values = append(values, sourceID)
	}

	if action != "" {
		whereStmts = append(whereStmts, "logging.action=?")
		values = append(values, action)
	}

	if uid > 0 {
		whereStmts = append(whereStmts, "logging.uid=?")
		values = append(values, uid)
	}

	if beginTime != nil {
		whereStmts = append(whereStmts, "logging.create_time >= ?")
		values = append(values, beginTime)
	}

	if endTime != nil {
		whereStmts = append(whereStmts, "logging.create_time <= ?")
		values = append(values, endTime)
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s logging
	`, loggingColumns, loggingTableName(siteID))
	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s logging
	`, loggingTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
		countSQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	var total int
	result := make([]*Logging, 0)
	userMap := make(map[int]*user.UserBrief)

	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count logging: ", err)
		return nil, nil, 0, err
	}

	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	SQL += "\nORDER BY logging.id DESC LIMIT ?,?"
	values = append(values, (pageNo-1)*pageSize, pageSize)

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get logging: ", err)
		return nil, nil, 0, err
	}

	defer rows.Close()

	userIDs := make([]int, 0)

	for rows.Next() {
		var l Logging
		if err := l.scan(rows); err != nil {
			log.Println("error scan logging: ", err)
			return nil, nil, 0, err
		}

		result = append(result, &l)
		userIDs = append(userIDs, l.UID)
	}

	if len(userIDs) > 0 {
		userList, err := user.GetUserBrief(siteID, userIDs...)
		if err != nil {
			log.Println("error get logging user: ", err)
		} else {
			for _, u := range userList {
				userMap[u.UserID] = u
			}
		}
	}

	return result, userMap, total, nil
}
