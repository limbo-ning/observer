package event

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
)

var Config struct {
	IsScheduler bool `json:"isScheduler"`
}

var scheduledEventsInstance *scheduledEvents

func init() {
	config.GetConfig("config.yaml", &Config)

	if Config.IsScheduler {
		scheduledEventsInstance = new(scheduledEvents)
		scheduledEventsInstance.pool = make(map[string]*scheduledEventsSitePool)
	}
}

func ScheduleEvents() {

	if !Config.IsScheduler {
		return
	}

	tables, err := scanSchedulerTables()
	if err != nil {
		log.Println("error scan scheduler tables: ", err)
		return
	}

	log.Println("scheduler tables to scan: ", tables)

	for _, t := range tables {
		scanSchedulers(t)
	}
}

type scheduledEventsSitePool struct {
	Lock sync.RWMutex
	Pool map[int]*time.Timer
}

type scheduledEvents struct {
	lock sync.RWMutex
	pool map[string]*scheduledEventsSitePool
}

func (b *scheduledEvents) getSitePool(siteID string) *scheduledEventsSitePool {
	b.lock.RLock()
	siteP, exists := b.pool[siteID]
	b.lock.RUnlock()

	if !exists {
		b.lock.Lock()
		defer b.lock.Unlock()

		siteP, exists = b.pool[siteID]

		if !exists {
			siteP = new(scheduledEventsSitePool)
			siteP.Pool = make(map[int]*time.Timer)

			b.pool[siteID] = siteP
		}
	}
	return siteP
}

type Scheduler struct {
	ID       int            `json:"ID"`
	Name     string         `json:"name"`
	Schedule *util.Interval `json:"schedule"`
	Template *Event         `json:"template"`
}

const schedulerColumn = "scheduler.id, scheduler.name, scheduler.schedule, scheduler.template"

func schedulerTableName(siteID string) string {
	return siteID + "_eventscheduler"
}

func (s *Scheduler) scan(rows *sql.Rows) error {
	var schedule, template string

	if err := rows.Scan(&s.ID, &s.Name, &schedule, &template); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(schedule), &s.Schedule); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(template), &s.Template); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) ScheduleEvent(siteID string) error {

	if !Config.IsScheduler {
		return errors.New("主机不是scheduler")
	}

	log.Println("schedule event: ", siteID, s.ID)

	start, end, err := s.Schedule.GetInterval(time.Now(), false)
	if err != nil {
		if err != util.E_not_in_interval {
			log.Println("error schedule event: ", err)
			return err
		}
	}

	if start == nil {
		log.Println("error schedule event: ", util.E_not_in_interval)
		return util.E_not_in_interval
	}

	for start.Before(time.Now()) {
		start, end, err = s.Schedule.GetInterval(end.Add(time.Second), false)
		if err != nil {
			if err != util.E_not_in_interval {
				log.Println("error schedule event: ", err)
				return err
			}
		}
		if start == nil {
			log.Println("error schedule event: ", util.E_not_in_interval)
			return util.E_not_in_interval
		}
	}
	timer := time.AfterFunc(
		time.Until(*start),
		func() {
			s.Template.SubRelateID["schedulerID"] = fmt.Sprintf("%d", s.ID)
			if err := datasource.Txn(func(txn *sql.Tx) {
				if _, err := s.Template.clone(siteID, txn); err != nil {
					log.Println("error clone scheduled event: ", err)
					panic(err)
				}
			}); err != nil {
				log.Println("error clone scheduled event: ", err)
			}
			s.ScheduleEvent(siteID)
		},
	)

	sitePool := scheduledEventsInstance.getSitePool(siteID)
	sitePool.Lock.Lock()
	defer sitePool.Lock.Unlock()

	if prev, exists := sitePool.Pool[s.ID]; exists {
		if !prev.Stop() {
			select {
			case <-prev.C:
			default:
			}
		}
	}

	sitePool.Pool[s.ID] = timer

	log.Println("schedule event done: ", s.ID, start, time.Until(*start).Hours())

	return nil
}

func (s *Scheduler) Add(siteID string) error {

	if s.Template == nil {
		return errors.New("需要任务template")
	}

	_, err := s.Template.Validate(siteID)
	if err != nil {
		return err
	}

	schedule, _ := json.Marshal(s.Schedule)
	template, _ := json.Marshal(s.Template)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name,schedule,template)
		VALUES
			(?,?,?)
	`, schedulerTableName(siteID)), s.Name, string(schedule), string(template)); err != nil {
		log.Println("error add scheduler: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error add scheduler: ", err)
		return err
	} else {
		s.ID = int(id)
	}

	s.ScheduleEvent(siteID)
	return nil
}

func (s *Scheduler) Update(siteID string) error {

	if s.Template == nil {
		return errors.New("需要任务template")
	}

	_, err := s.Template.Validate(siteID)
	if err != nil {
		return err
	}

	schedule, _ := json.Marshal(s.Schedule)
	template, _ := json.Marshal(s.Template)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?,schedule=?,template=?
		WHERE
			id = ?
	`, schedulerTableName(siteID)), s.Name, string(schedule), string(template), s.ID); err != nil {
		log.Println("error update scheduler: ", err)
		return err
	}

	s.ScheduleEvent(siteID)

	return nil
}

func (s *Scheduler) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, schedulerTableName(siteID)), s.ID); err != nil {
		log.Println("error delete scheduler: ", err)
		return err
	}

	sitePool := scheduledEventsInstance.getSitePool(siteID)
	sitePool.Lock.Lock()
	defer sitePool.Lock.Unlock()

	if prev, exists := sitePool.Pool[s.ID]; exists {
		if !prev.Stop() {
			select {
			case <-prev.C:
			default:
			}
		}
	}

	delete(sitePool.Pool, s.ID)

	return nil
}

func scanSchedulerTables() ([]string, error) {
	rows, err := datasource.GetConn().Query(`
		SHOW TABLES LIKE '%_eventscheduler'
	`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		if strings.Index(strings.TrimSuffix(table, "_eventscheduler"), "_") > 0 {
			continue
		}
		result = append(result, table)
	}
	return result, nil
}

func scanSchedulers(table string) error {
	parts := strings.Split(table, "_")
	if len(parts) != 2 {
		log.Println("error not scheduler table: ", table)
		return fmt.Errorf("not scheduler table: %s", table)
	}
	siteID := parts[0]

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s scheduler
	`, schedulerColumn, table))

	if err != nil {
		log.Println("error scan schduler: ", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var scheduler Scheduler

		if err := scheduler.scan(rows); err != nil {
			log.Println("error scan scheduler: ", err)
			continue
		}

		if _, err := scheduler.Template.Validate(siteID); err != nil {
			log.Println("error validate after scan scheduler: ", err)
			continue
		}

		if err := scheduler.ScheduleEvent(siteID); err != nil {
			log.Println("error schedule scheduler: ", err)
			continue
		}
	}

	return nil
}

func GetSchedulers(siteID string, eventType, eventMainRelateID, q string, pageNo, pageSize int, schedulerID ...int) ([]*Scheduler, int, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s scheduler
	`, schedulerTableName(siteID))

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s scheduler
	`, schedulerColumn, schedulerTableName(siteID))

	if eventType != "" {
		whereStmts = append(whereStmts, "JSON_EXTRACT(scheduler.template, '$.type') = ?")
		values = append(values, eventType)
	}

	if eventMainRelateID != "" {
		whereStmts = append(whereStmts, "JSON_EXTRACT(scheduler.template, '$.mainRelateID') = ?")
		values = append(values, eventMainRelateID)
	}

	if len(schedulerID) > 0 {
		pageSize = -1
		if len(schedulerID) == 1 {
			whereStmts = append(whereStmts, "scheduler.id = ?")
			values = append(values, schedulerID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range schedulerID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("scheduler.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if q != "" {
		whereStmts = append(whereStmts, "scheduler.name LIKE ?")
		values = append(values, "%"+q+"%")
	}

	if len(whereStmts) > 0 {
		countSQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	var total int

	if err := datasource.GetConn().QueryRow(countSQL, values...).Scan(&total); err != nil {
		log.Println("error count scheduler: ", err)
		return nil, 0, err
	}

	SQL += "\nORDER BY scheduler.id DESC"
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
		log.Println("error get scheduler: ", SQL, values, err)
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]*Scheduler, 0)

	for rows.Next() {
		var e Scheduler
		if err := e.scan(rows); err != nil {
			return nil, 0, err
		}
		result = append(result, &e)
	}
	return result, total, nil
}
