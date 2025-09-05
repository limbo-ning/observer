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
	"obsessiontech/environment/category"
)

var E_mission_not_found = errors.New("任务不存在")
var E_mission_not_avail = errors.New("任务未可用")
var E_no_quota = errors.New("没有任务次数")

const (
	MISSION_ACTIVE   = "ACTIVE"
	MISSION_INACTIVE = "INACTIVE"

	MISSION_OPEN    = "OPEN"
	MISSION_SUCCESS = "SUCCESS"
	MISSION_FAILED  = "FAILED"
	MISSION_CLOSED  = "CLOSED"
)

type MissionData struct {
	ID          int                  `json:"ID"`
	RelateID    map[string]string    `json:"relateID"`
	Type        string               `json:"type"`
	Status      string               `json:"status"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Profile     map[string][]string  `json:"profile"`
	Sort        int                  `json:"sort"`
	Prequisites MissionPrerequisites `json:"prerequisites"`
	Completes   MissionCompletes     `json:"completes"`

	BeginTime util.Time `json:"beginTime"`
	EndTime   util.Time `json:"endTime"`

	Section *util.Interval `json:"sectionInterval"`
	Active  *util.Interval `json:"activeInterval"`

	CreateTime util.Time `json:"createTime"`
	UpdateTime util.Time `json:"updateTime"`
}

type Mission struct {
	MissionData

	Quota                  int        `json:"quota"`
	CompleteCount          int        `json:"completeCount"`
	SectionBeginTime       *util.Time `json:"sectionBeginTime,omitempty"`
	SectionEndTime         *util.Time `json:"sectionEndTime,omitempty"`
	SectionActiveBeginTime *util.Time `json:"sectionActiveBeginTime,omitempty"`
	SectionActiveEndTime   *util.Time `json:"sectionActiveEndTime,omitempty"`
}

func missionTable(siteID string) string {
	return siteID + "_mission"
}

const missionColumn = "mission.id,mission.relate_id,mission.type,mission.name,mission.status,mission.description,mission.profile,mission.sort,mission.prerequisites,mission.completes,mission.section_interval,mission.active_interval,mission.begin_time,mission.end_time,mission.create_time,mission.update_time"

func (m *MissionData) scan(rows *sql.Rows) error {

	var relateID, profile, prerequisites, completes, section, active string
	if err := rows.Scan(&m.ID, &relateID, &m.Type, &m.Name, &m.Status, &m.Description, &profile, &m.Sort, &prerequisites, &completes, &section, &active, &m.BeginTime, &m.EndTime, &m.CreateTime, &m.UpdateTime); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(relateID), &m.RelateID); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(profile), &m.Profile); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(prerequisites), &m.Prequisites); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(completes), &m.Completes); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(section), &m.Section); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(active), &m.Active); err != nil {
		return err
	}

	return nil
}

func (m *Mission) check(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, targetTime time.Time, missions map[int]*Mission, completeRequests map[int]*MissionCompleteRequest, postCheck *map[*Mission][]IMissionPostCheckPrerequisite) error {

	sectionBeginTime, sectionEndTime, err := m.Section.GetInterval(targetTime, true)
	if err != nil {
		if err == util.E_not_in_interval {
			m.Status = MISSION_INACTIVE
			return nil
		}
		return err
	}

	if sectionBeginTime != nil && sectionBeginTime != &util.DefaultMin {
		m.SectionBeginTime = new(util.Time)
		*m.SectionBeginTime = util.Time(*sectionBeginTime)
	}
	if sectionEndTime != nil && sectionEndTime != &util.DefaultMax {
		m.SectionEndTime = new(util.Time)
		*m.SectionEndTime = util.Time(*sectionEndTime)
	}

	sectionActiveBeginTime, sectionActiveEndTime, err := m.Active.GetInterval(targetTime, true)
	if err != nil {
		if err == util.E_not_in_interval {
			m.Status = MISSION_INACTIVE
			return nil
		}
		return err
	}

	if sectionActiveBeginTime != nil && sectionActiveBeginTime != &util.DefaultMin {
		m.SectionActiveBeginTime = new(util.Time)
		*m.SectionActiveBeginTime = util.Time(*sectionActiveBeginTime)
	}
	if sectionActiveEndTime != nil && sectionActiveEndTime != &util.DefaultMax {
		m.SectionActiveEndTime = new(util.Time)
		*m.SectionActiveEndTime = util.Time(*sectionActiveEndTime)
	}

	m.Quota = -1
	m.CompleteCount = -1

	for _, pre := range m.Prequisites {
		if err := pre.CheckMission(siteID, txn, m, actionAuth, targetTime, missions, completeRequests); err != nil {
			log.Println("error run prerequisite: ", pre.GetType(), err)
			if err != E_mission_interrupt {
				return err
			}
			break
		}

		if post, ok := pre.(IMissionPostCheckPrerequisite); ok {
			list, exists := (*postCheck)[m]
			if !exists {
				list = make([]IMissionPostCheckPrerequisite, 0)
			}
			(*postCheck)[m] = append(list, post)
		}
	}

	if m.Status == MISSION_OPEN {
		m.Status = MISSION_ACTIVE
	}

	return nil
}

func GetMissionTypes(siteID string) ([]string, error) {
	result := make([]string, 0)

	SQL := fmt.Sprintf(`
		SELECT
			DISTINCT mission.type
		FROM
			%s mission
	`, missionTable(siteID))

	rows, err := datasource.GetConn().Query(SQL)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, nil
}

func getMissionsWithTxn(siteID string, txn *sql.Tx, forUpdate bool, missionType string, missionID ...int) ([]*Mission, error) {
	result := make([]*Mission, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if missionType != "" {
		whereStmts = append(whereStmts, "mission.type = ?")
		values = append(values, missionType)
	} else if len(missionID) == 0 {
		return result, nil
	}

	if len(missionID) == 1 {
		if missionID[0] != -1 {
			whereStmts = append(whereStmts, "mission.id = ?")
			values = append(values, missionID[0])
		}
	} else if len(missionID) > 1 {
		placeHolder := make([]string, 0)
		for _, id := range missionID {
			placeHolder = append(placeHolder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("mission.id IN (%s)", strings.Join(placeHolder, ",")))
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s mission
	`, missionColumn, missionTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nORDER BY mission.sort DESC"

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
		log.Println("error get mission: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Mission
		if err := m.scan(rows); err != nil {
			return nil, err
		}

		result = append(result, &m)
	}

	return result, nil
}

func getMissions(siteID string, actionAuth authority.ActionAuthSet, relateID map[string]string, cids []int, targetTime *time.Time, missionType, status, q string, pageNo, pageSize int, missionID []int, authType, empowerType string, empowerID ...string) ([]*Mission, int, error) {

	result := make([]*Mission, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT mission.id)
		FROM
			%s mission
	`, missionTable(siteID))
	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s mission
	`, missionColumn, missionTable(siteID))

	if authType == "" {
		authType = ACTION_VIEW
	}
	authSQL, authWhere, authValues, err := authority.JoinEmpower(siteID, "mission", actionAuth, AdminActions, authType, "mission", "id", empowerType, empowerID...)
	if err != nil {
		return nil, 0, err
	}
	countSQL += authSQL
	SQL += authSQL
	if authWhere != nil {
		whereStmts = append(whereStmts, authWhere...)
	}
	if authValues != nil {
		values = append(values, authValues...)
	}

	if len(cids) > 0 {
		joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "mission", "", cids...)
		if err != nil {
			return nil, 0, err
		}

		countSQL += joinSQL
		SQL += joinSQL
		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if len(relateID) > 0 {
		for key, value := range relateID {
			whereStmts = append(whereStmts, fmt.Sprintf("JSON_EXTRACT(mission.relate_id, '$.%s') = ?", key))
			values = append(values, value)
		}
	}

	if targetTime != nil {
		whereStmts = append(whereStmts, "mission.begin_time <= ?")
		whereStmts = append(whereStmts, "mission.end_time >= ?")
		values = append(values, targetTime, targetTime)
	}

	if missionType != "" {
		whereStmts = append(whereStmts, "mission.type = ?")
		values = append(values, missionType)
	}

	if status != "" {
		whereStmts = append(whereStmts, "mission.status = ?")
		values = append(values, status)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(mission.name LIKE ? OR mission.description LIKE ?)")
		values = append(values, qq, qq)
	}

	if len(missionID) > 0 {
		if len(missionID) == 1 {
			if missionID[0] != -1 {
				whereStmts = append(whereStmts, "mission.id = ?")
				values = append(values, missionID[0])
			}
		} else {
			placeHolder := make([]string, 0)
			for _, id := range missionID {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("mission.id IN (%s)", strings.Join(placeHolder, ",")))
		}
		pageSize = -1
	}

	if len(whereStmts) > 0 {
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	var total int
	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count mission: ", err)
		return nil, 0, err
	}

	SQL += "\nGROUP BY mission.id"

	SQL += "\nORDER BY mission.sort DESC, mission.id DESC"
	if pageSize != -1 {
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageNo <= 0 {
			pageNo = 1
		}

		SQL += "\nLIMIT ?,?"
		values = append(values, (pageNo-1)*pageSize, pageSize)
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get mission: ", SQL, values, err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Mission
		if err := m.scan(rows); err != nil {
			return nil, 0, err
		}

		result = append(result, &m)
	}

	return result, total, nil
}

func GetMissions(siteID string, actionAuth authority.ActionAuthSet, skipCheck bool, relateID map[string]string, cids []int, missionType, status, q string, targetTime *time.Time, pageNo, pageSize int, missionID []int, authType, empowerType string, empowerID ...string) ([]*Mission, int, error) {

	missionList, total, err := getMissions(siteID, actionAuth, relateID, cids, targetTime, missionType, status, q, pageNo, pageSize, missionID, authType, empowerType, empowerID...)
	if err != nil {
		return nil, 0, err
	}

	if targetTime != nil && !skipCheck {
		missions := make(map[int]*Mission)

		listToCheck := make([]*Mission, 0)

		for _, m := range missionList {
			missions[m.ID] = m

			if m.Status == MISSION_OPEN {
				if time.Time(m.BeginTime).After(time.Now()) || time.Time(m.EndTime).Before(time.Now()) {
					m.Status = MISSION_INACTIVE
				} else {
					listToCheck = append(listToCheck, m)
				}
			}
		}

		appendCheckMissionList := make([]*Mission, 0)
		for _, m := range listToCheck {
			for _, pre := range m.Prequisites {
				if appendMission, ok := pre.(IMissionRequireAppendMissionCheckingPrerequisite); ok {
					appended, err := appendMission.AppendMissionChecking(siteID, nil, m, actionAuth, missions)
					if err != nil {
						return nil, 0, err
					}
					appendCheckMissionList = append(appendCheckMissionList, appended...)
				}
			}
		}

		uidsCompleteRequests := make(map[int]*MissionCompleteRequest)
		for _, m := range listToCheck {
			for _, pre := range m.Prequisites {
				if requestComplete, ok := pre.(IMissionRequireCompletePrerequisite); ok {
					requestComplete.UpdateCompleteRequire(siteID, nil, m, actionAuth, *targetTime, missions, uidsCompleteRequests)
				}
			}
		}
		for _, m := range appendCheckMissionList {
			for _, pre := range m.Prequisites {
				if requestComplete, ok := pre.(IMissionRequireCompletePrerequisite); ok {
					requestComplete.UpdateCompleteRequire(siteID, nil, m, actionAuth, *targetTime, missions, uidsCompleteRequests)
				}
			}
		}

		for uid, requests := range uidsCompleteRequests {
			counts, err := countCompletes(siteID, nil, uid, []string{COMPLETE_FINAL, COMPLETE_PENDING}, missions, requests.TargetTimes, false, true)
			if err != nil {
				return nil, 0, err
			}
			requests.Result = counts
		}

		postCheck := make(map[*Mission][]IMissionPostCheckPrerequisite)
		for _, m := range append(appendCheckMissionList, missionList...) {
			if err := m.check(siteID, nil, actionAuth, *targetTime, missions, uidsCompleteRequests, &postCheck); err != nil {
				return nil, 0, err
			}
		}

		for m, posts := range postCheck {
			for _, p := range posts {
				if err := p.PostCheckMission(siteID, nil, m, actionAuth, *targetTime, missions); err != nil {
					return nil, 0, err
				}
			}

		}
		for _, m := range listToCheck {
			if m.Quota == 0 {
				m.Status = MISSION_INACTIVE
			}
			if m.Quota > 0 && m.CompleteCount >= m.Quota {
				m.Status = MISSION_INACTIVE
			}
		}
	} else {
		for _, m := range missionList {
			if m.Status == MISSION_OPEN {
				m.Status = MISSION_INACTIVE
			}
		}
	}

	return missionList, total, nil
}

func CompleteMission(siteID string, actionAuth authority.ActionAuthSet, missionID int, completeResult map[string]interface{}, completeTime time.Time) (*Complete, error) {

	mm, err := GetModule(siteID)
	if err != nil {
		return nil, err
	}

	var result *Complete

	if err := datasource.Txn(func(txn *sql.Tx) {

		missionList, err := getMissionsWithTxn(siteID, txn, false, "", missionID)
		if err != nil {
			log.Println("error get mission list: ", err)
			panic(err)
		}

		if len(missionList) == 0 {
			panic(E_mission_not_avail)
		}

		m := missionList[0]

		var mt *MissionType
		for _, t := range mm.Types {
			if t.Type == m.Type {
				mt = t
			}
		}

		if mt == nil {
			panic(E_mission_not_avail)
		}

		missions := make(map[int]*Mission)
		for _, m := range missionList {
			missions[m.ID] = m
		}

		appendCheckMissionList := make([]*Mission, 0)
		for _, m := range missionList {
			for _, pre := range m.Prequisites {
				if appendMission, ok := pre.(IMissionRequireAppendMissionCheckingPrerequisite); ok {
					appended, err := appendMission.AppendMissionChecking(siteID, txn, m, actionAuth, missions)
					if err != nil {
						log.Println("error append mission checking: ", err)
						panic(err)
					}
					appendCheckMissionList = append(appendCheckMissionList, appended...)
				}
			}
			for _, pre := range m.Completes {
				if appendMission, ok := pre.(IMissionRequireAppendMissionCheckingPrerequisite); ok {
					appended, err := appendMission.AppendMissionChecking(siteID, txn, m, actionAuth, missions)
					if err != nil {
						log.Println("error append mission complete checking: ", err)
						panic(err)
					}
					appendCheckMissionList = append(appendCheckMissionList, appended...)
				}
			}
		}

		uidsCompleteRequests := make(map[int]*MissionCompleteRequest)
		for _, m := range append(missionList, appendCheckMissionList...) {
			for _, pre := range m.Prequisites {
				if requestComplete, ok := pre.(IMissionRequireCompletePrerequisite); ok {
					requestComplete.UpdateCompleteRequire(siteID, txn, m, actionAuth, completeTime, missions, uidsCompleteRequests)
				}
			}
		}

		for uid, requests := range uidsCompleteRequests {
			counts, err := countCompletes(siteID, txn, uid, []string{COMPLETE_FINAL, COMPLETE_PENDING}, missions, requests.TargetTimes, false, true)
			if err != nil {
				log.Println("error count completes: ", err)
				panic(err)
			}
			requests.Result = counts
		}

		postCheck := make(map[*Mission][]IMissionPostCheckPrerequisite)
		for _, m := range append(missionList, appendCheckMissionList...) {
			if err := m.check(siteID, txn, actionAuth, completeTime, missions, uidsCompleteRequests, &postCheck); err != nil {
				log.Println("error check: ", err)
				panic(err)
			}
		}

		if m.Status != MISSION_ACTIVE {
			panic(E_mission_not_avail)
		}

		if posts, exists := postCheck[m]; exists {
			for _, p := range posts {
				if err := p.PostCheckMission(siteID, txn, m, actionAuth, completeTime, missions); err != nil {
					log.Println("error post check: ", err)
					panic(err)
				}
			}
		}

		if m.Status != MISSION_ACTIVE {
			panic(E_mission_not_avail)
		}

		if m.Quota == 0 {
			panic(E_no_quota)
		} else if m.Quota > 0 && m.CompleteCount >= m.Quota {
			panic(E_no_quota)
		}

		result, err = completeMission(siteID, txn, actionAuth, m, missions, completeResult, completeTime)
		if err != nil {
			log.Println("error complete mission: ", err)
			panic(err)
		}

	}); err != nil {
		return nil, err
	}

	return result, nil
}

func completeMission(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, m *Mission, missions map[int]*Mission, completeResult map[string]interface{}, completeTime time.Time) (*Complete, error) {

	result := new(Complete)

	result.MissionID = m.ID
	result.Result = completeResult
	if result.Result == nil {
		result.Result = make(map[string]interface{})
	}

	if m.SectionBeginTime != nil {
		result.Section = util.FormatDateTime(time.Time(*m.SectionBeginTime))
	} else {
		result.Section = "-"
	}
	result.Section += "~"
	if m.SectionEndTime != nil {
		result.Section += util.FormatDateTime(time.Time(*m.SectionEndTime))
	} else {
		result.Section += "-"
	}
	result.CompleteTime = util.Time(completeTime)
	result.UID = actionAuth.GetUID()
	result.Ext = make(map[string]interface{})

	if err := result.add(siteID, txn); err != nil {
		log.Println("error add mission complete: ", err)
		return nil, err
	}

	for _, c := range m.Completes {
		if err := c.MissionComplete(siteID, txn, actionAuth, m, missions, result); err != nil {
			if err == E_mission_interrupt {
				break
			}
			log.Println("error process complete: ", err)
			return nil, err
		}
	}

	if result.Status == "" {
		result.Status = COMPLETE_FINAL
	}

	if err := result.update(siteID, txn); err != nil {
		log.Println("error update complete: ", err)
		return nil, err
	}

	if err := m.update(siteID, txn); err != nil {
		log.Println("error update mission: ", err)
		return nil, err
	}

	return result, nil
}

func (m *Mission) Validate(siteID string) error {
	mm, err := GetModule(siteID)
	if err != nil {
		return err
	}

	var mt *MissionType
	for _, t := range mm.Types {
		if t.Type == m.Type {
			mt = t
		}
	}

	if mt == nil {
		return errors.New("invalid mission type")
	}

	if mt.Template != nil && len(m.Prequisites) == 0 && len(m.Completes) == 0 {
		if err := util.Clone(&mt.Template.Prequisites, &m.Prequisites); err != nil {
			return err
		}
		if err := util.Clone(&mt.Template.Completes, &m.Completes); err != nil {
			return err
		}
		if m.Section == nil {
			if err := util.Clone(&mt.Template.Section, &m.Section); err != nil {
				return err
			}
		}
		if m.Active == nil {
			if err := util.Clone(&mt.Template.Active, &m.Active); err != nil {
				return err
			}
		}
		if time.Time(m.BeginTime).IsZero() {
			m.BeginTime = util.Time(mt.Template.BeginTime)
		}
		if time.Time(m.EndTime).IsZero() {
			m.EndTime = util.Time(mt.Template.EndTime)
		}
	}

	switch m.Status {
	case MISSION_OPEN:
	case MISSION_SUCCESS:
	case MISSION_FAILED:
	case MISSION_CLOSED:
	default:
		m.Status = MISSION_OPEN
	}

	for _, mc := range m.Prequisites {
		if err := mc.Validate(siteID); err != nil {
			return err
		}
	}

	for _, mc := range m.Completes {
		if err := mc.Validate(siteID); err != nil {
			return err
		}
	}

	if m.RelateID == nil {
		m.RelateID = make(map[string]string)
	}
	if m.Profile == nil {
		m.Profile = make(map[string][]string)
	}
	if m.Prequisites == nil {
		m.Prequisites = make(MissionPrerequisites, 0)
	}
	if m.Completes == nil {
		m.Completes = make(MissionCompletes, 0)
	}
	if m.Section == nil {
		m.Section = new(util.Interval)
		m.Section.Init()
	}
	if m.Active == nil {
		m.Active = new(util.Interval)
		m.Active.Init()
	}
	if time.Time(m.BeginTime).IsZero() {
		m.BeginTime = util.Time(util.DefaultMin)
	}
	if time.Time(m.EndTime).IsZero() {
		m.EndTime = util.Time(util.DefaultMax)
	}

	return nil
}

func (m *Mission) Add(siteID string) error {

	table := missionTable(siteID)

	if err := m.Validate(siteID); err != nil {
		return err
	}
	relateID, _ := json.Marshal(m.RelateID)
	profile, _ := json.Marshal(m.Profile)
	prerequisites, _ := json.Marshal(m.Prequisites)
	completes, _ := json.Marshal(m.Completes)
	sectionInterval, _ := json.Marshal(m.Section)
	activeInterval, _ := json.Marshal(m.Active)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(type,name,relate_id,status,description,profile,sort,prerequisites,completes,section_interval,active_interval,begin_time,end_time)
		VALUES
			(?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, table), m.Type, m.Name, string(relateID), m.Status, m.Description, string(profile), m.Sort, string(prerequisites), string(completes), string(sectionInterval), string(activeInterval), time.Time(m.BeginTime), time.Time(m.EndTime)); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		m.ID = int(id)
	}

	return nil
}

func (m *Mission) Update(siteID string) error {
	return m.update(siteID, nil)
}

func (m *Mission) update(siteID string, txn *sql.Tx) error {
	table := missionTable(siteID)

	err := m.Validate(siteID)
	if err != nil {
		return err
	}

	relateID, _ := json.Marshal(m.RelateID)
	profile, _ := json.Marshal(m.Profile)
	prerequisites, _ := json.Marshal(m.Prequisites)
	completes, _ := json.Marshal(m.Completes)
	sectionInterval, _ := json.Marshal(m.Section)
	activeInterval, _ := json.Marshal(m.Active)

	if txn == nil {
		_, err = datasource.GetConn().Exec(fmt.Sprintf(`
			UPDATE
				%s
			SET
				type=?,name=?,relate_id=?,status=?,description=?,profile=?,sort=?,prerequisites=?,completes=?,section_interval=?,active_interval=?,begin_time=?,end_time=?
			WHERE
				id=?
		`, table), m.Type, m.Name, string(relateID), m.Status, m.Description, string(profile), m.Sort, string(prerequisites), string(completes), string(sectionInterval), string(activeInterval), time.Time(m.BeginTime), time.Time(m.EndTime), m.ID)
	} else {
		_, err = txn.Exec(fmt.Sprintf(`
			UPDATE
				%s
			SET
				type=?,name=?,relate_id=?,status=?,description=?,profile=?,sort=?,prerequisites=?,completes=?,section_interval=?,active_interval=?,begin_time=?,end_time=?
			WHERE
				id=?
		`, table), m.Type, m.Name, string(relateID), m.Status, m.Description, string(profile), m.Sort, string(prerequisites), string(completes), string(sectionInterval), string(activeInterval), time.Time(m.BeginTime), time.Time(m.EndTime), m.ID)
	}

	if err != nil {
		return err
	}

	return nil
}

func (m *Mission) Delete(siteID string) error {
	table := missionTable(siteID)

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

func AddMissionCategory(siteID string, missionID, categoryID int) error {
	return category.AddCategoryMapping(siteID, "mission", missionID, categoryID)
}

func DeleteMissionCategory(siteID string, missionID, categoryID int) error {
	return category.DeleteCategoryMapping(siteID, "mission", missionID, categoryID)
}
