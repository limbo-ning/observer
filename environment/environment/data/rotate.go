package data

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"obsessiontech/common/config"
	"obsessiontech/common/datasource"
	"obsessiontech/common/ipc"
	"obsessiontech/common/util"
)

const (
	rotation_year    = "year"
	rotation_quarter = "quarter"
	rotation_month   = "month"
	rotation_week    = "week"

	active     = "active"
	rotating   = "rotating"
	archived   = "archived"
	activating = "activating"
	activated  = "activated"
)

var rotationCallLock sync.RWMutex
var rotationCalls = make(map[string]*time.Timer)

var archiveRollbackCallLock sync.RWMutex
var archiveRollbackCalls = make(map[string]map[string]*time.Timer)

func formatArchiveDate(t time.Time) string {
	return t.Format("20060102")
}
func parseArchiveDate(str string) (time.Time, error) {
	return time.ParseInLocation("20060102", str, time.Local)
}

var Config struct {
	IsEnvironmentArchiveWorker                  bool
	EnvironmentArchiveWorkerHostType            string
	EnvironmentArchiveWorkerHost                string
	EnvironmentArchiveWorkerReconnectTimeOutSec time.Duration
}

var rotationPersistentClientLock sync.RWMutex
var rotationPersistentClient map[*ipc.Connection]bool

func init() {
	config.GetConfig("config.yaml", &Config)

	if Config.EnvironmentArchiveWorkerHost != "" {
		if Config.IsEnvironmentArchiveWorker {
			startHost()
		} else {
			startPersistentClient()
		}
	}

	if Config.IsEnvironmentArchiveWorker {
		recoverInterruptedProgress()
	}
}

func startHost() {
	connChan, err := ipc.StartHost(Config.EnvironmentArchiveWorkerHostType, Config.EnvironmentArchiveWorkerHost)
	if err != nil {
		log.Println("error establish environment rotation host: ", err)
		panic(err)
	}

	go func() {
		for conn := range connChan {
			go hostListen(conn)
		}
		log.Println("establish environment rotation host closed")
	}()
}

func startPersistentClient() {
	rotationPersistentClientLock.Lock()
	defer rotationPersistentClientLock.Unlock()

	conn, err := ipc.StartClient(Config.EnvironmentArchiveWorkerHostType, Config.EnvironmentArchiveWorkerHost)
	if err != nil {
		log.Println("error set up environment rotation client: ", err)
		if Config.EnvironmentArchiveWorkerReconnectTimeOutSec > 0 {
			time.AfterFunc(time.Second*Config.EnvironmentArchiveWorkerReconnectTimeOutSec, startPersistentClient)
		}
		return
	}

	if err := send(conn, new(ListenReq)); err != nil {
		log.Println("error set up environment rotation client: ", err)
		if Config.EnvironmentArchiveWorkerReconnectTimeOutSec > 0 {
			time.AfterFunc(time.Second*Config.EnvironmentArchiveWorkerReconnectTimeOutSec, startPersistentClient)
		}
		conn.Cancel()
		return
	}

	rotationPersistentClient[conn] = true

	go func() {
		clientListen(conn)

		rotationPersistentClientLock.Lock()
		defer rotationPersistentClientLock.Unlock()
		delete(rotationPersistentClient, conn)

		if Config.EnvironmentArchiveWorkerReconnectTimeOutSec > 0 {
			time.AfterFunc(time.Second*Config.EnvironmentArchiveWorkerReconnectTimeOutSec, startPersistentClient)
		}
	}()
}

func TriggerRotation(siteID string, immediate bool) (timer *time.Timer) {

	if !Config.IsEnvironmentArchiveWorker {

		rotationPersistentClientLock.RLock()
		defer rotationPersistentClientLock.RUnlock()

		for conn := range rotationPersistentClient {
			if err := send(conn, &TriggerRotationReq{SiteID: siteID, Immediate: immediate}); err != nil {
				log.Println("error send ipc trigger rotation: ", err)
			}
		}

		return
	}

	if immediate {
		Rotate(siteID)
		return
	}

	defer func() {
		if timer == nil {
			rotationCallLock.Lock()
			defer rotationCallLock.Unlock()

			log.Println("trigger rotation: ", siteID)

			rotationCalls[siteID] = time.AfterFunc(time.Until(util.GetDate(time.Now()).AddDate(0, 0, 1).Add(5*time.Minute*time.Duration(len(rotationCalls)+1))), func() {
				Rotate(siteID)

				rotationCallLock.Lock()
				defer rotationCallLock.Unlock()
				delete(rotationCalls, siteID)
			})
		}
	}()

	rotationCallLock.RLock()
	defer rotationCallLock.RUnlock()

	timer = rotationCalls[siteID]

	return
}

type Rotation struct {
	DataType string `json:"dataType"`
	Active   string `json:"active"`
	Archive  string `json:"archive"`
}

type archiveTable struct {
	TableName string
	BeginTime time.Time
	EndTime   time.Time
	Status    string
}

type siteArchive struct {
	Lock sync.RWMutex
	Pool map[string][]*archiveTable
}

var archivesPool = make(map[string]*siteArchive)
var archivesPoolLock sync.RWMutex

func ClearArchiveTable(siteID, dataType string) {

	if Config.IsEnvironmentArchiveWorker {
		rotationPersistentClientLock.RLock()
		defer rotationPersistentClientLock.RUnlock()

		for conn := range rotationPersistentClient {
			if err := send(conn, &ClearArchiveEntryReq{SiteID: siteID, DataType: dataType}); err != nil {
				log.Println("error send ipc clear archive: ", err)
			}
		}
	}

	archivesPoolLock.Lock()
	defer archivesPoolLock.Unlock()

	site, exists := archivesPool[siteID]
	if !exists {
		return
	}

	site.Lock.Lock()
	defer site.Lock.Unlock()

	delete(site.Pool, dataType)
}

func parseArchiveTableInfo(siteID, dataType, tableName string) (*archiveTable, error) {
	table := TableName(siteID, dataType) + "_"

	result := new(archiveTable)

	result.TableName = tableName

	parts := strings.Split(tableName[len(table):], "_")
	if len(parts) < 2 {
		return nil, errors.New("invalid table name: " + tableName)
	}
	begin, err := parseArchiveDate(parts[0])
	if err != nil {
		return nil, errors.New("invalid table begin: " + tableName)
	}
	end, err := parseArchiveDate(parts[1])
	if err != nil {
		return nil, errors.New("invalid table end: " + tableName)
	}
	result.BeginTime = begin
	result.EndTime = end

	if len(parts) == 3 {
		switch parts[2] {
		case activating:
			result.Status = activating
		case active:
			result.Status = active
		default:
			return nil, errors.New("intermediate table status: " + tableName)
		}
	} else {
		result.Status = archived
	}

	return result, nil
}

func getArchiveTables(siteID, dataType string) (site *siteArchive, result []*archiveTable) {

	defer func() {
		if result == nil {
			table := TableName(siteID, dataType) + "_"

			result = make([]*archiveTable, 0)

			SQL := "SHOW TABLES LIKE '" + table + "%'"

			rows, err := datasource.GetConn().Query(SQL)
			if err != nil {
				log.Println("error show archive tables: ", err)
				return
			}
			defer rows.Close()

			for rows.Next() {
				var t string
				if err := rows.Scan(&t); err != nil {
					log.Println("error show archive tables: ", err)
					continue
				}

				archive, err := parseArchiveTableInfo(siteID, dataType, t)
				if err != nil {
					log.Println("error parse archive table: ", err)
					continue
				}

				if archive.Status == active {
					TriggerArchiveRollback(siteID, dataType, t)
				}
				result = append(result, archive)
			}

			sort.Slice(result, func(i, j int) bool {
				return result[i].EndTime.After(result[j].EndTime)
			})

			if site == nil {
				archivesPoolLock.Lock()
				defer archivesPoolLock.Unlock()

				site = archivesPool[siteID]
				if site == nil {
					site = new(siteArchive)
					site.Pool = make(map[string][]*archiveTable)
					archivesPool[siteID] = site
				}

			}

			site.Lock.Lock()
			defer site.Lock.Unlock()

			site.Pool[dataType] = result

			log.Println("fetched archive tables from datasource ", siteID, dataType, len(result))
		}
	}()

	var exists bool

	archivesPoolLock.RLock()
	defer archivesPoolLock.RUnlock()

	site, exists = archivesPool[siteID]
	if !exists {
		return
	}

	site.Lock.RLock()
	defer site.Lock.RUnlock()

	result = site.Pool[dataType]

	return
}

func (r *Rotation) getActiveTime() time.Time {
	switch r.Active {
	case rotation_year:
		return util.GetDate(time.Now()).AddDate(-1, 0, 0)
	case rotation_quarter:
		return util.GetDate(time.Now()).AddDate(0, -3, 0)
	case rotation_month:
		return util.GetDate(time.Now()).AddDate(0, -1, 0)
	case rotation_week:
		return util.GetDate(time.Now()).AddDate(0, 0, -7)
	}

	return time.Time{}
}

func (r *Rotation) getArchiveTime(current time.Time) (begin, end time.Time) {
	year, month, _ := current.Date()

	switch r.Archive {
	case rotation_year:
		begin = time.Date(year, time.January, 1, 0, 0, 0, 0, current.Location())
		end = begin.AddDate(1, 0, 0)
	case rotation_quarter:
		switch month {
		case time.January:
			fallthrough
		case time.February:
			fallthrough
		case time.March:
			begin = time.Date(year, time.January, 1, 0, 0, 0, 0, current.Location())
		case time.April:
			fallthrough
		case time.May:
			fallthrough
		case time.June:
			begin = time.Date(year, time.April, 1, 0, 0, 0, 0, current.Location())
		case time.July:
			fallthrough
		case time.August:
			fallthrough
		case time.September:
			begin = time.Date(year, time.July, 1, 0, 0, 0, 0, current.Location())
		default:
			begin = time.Date(year, time.October, 1, 0, 0, 0, 0, current.Location())
		}
		end = begin.AddDate(0, 3, 0)
	case rotation_month:
		begin = time.Date(year, month, 1, 0, 0, 0, 0, current.Location())
		end = begin.AddDate(0, 1, 0)
	case rotation_week:
		weekday := current.Weekday()
		switch weekday {
		case time.Sunday:
			begin = util.GetDate(current)
		case time.Monday:
			begin = util.GetDate(current).AddDate(0, 0, -1)
		case time.Tuesday:
			begin = util.GetDate(current).AddDate(0, 0, -2)
		case time.Wednesday:
			begin = util.GetDate(current).AddDate(0, 0, -3)
		case time.Thursday:
			begin = util.GetDate(current).AddDate(0, 0, -4)
		case time.Friday:
			begin = util.GetDate(current).AddDate(0, 0, -5)
		case time.Saturday:
			begin = util.GetDate(current).AddDate(0, 0, -6)
		}
		end = begin.AddDate(0, 0, 7)
	}

	log.Println("get archive date: ", current, r.Archive, begin, end)

	return
}

func Rotate(siteID string) error {

	log.Println("do rotation: ", siteID)

	dataModule, err := GetModule(siteID)
	if err != nil {
		log.Println("error rotate: get data module", err)
		return err
	}

	for _, r := range dataModule.Rotations {
		if err := rotate(siteID, r.DataType, r, dataModule.RotationBatchSize); err != nil {
			log.Println("error rotate: ", siteID, r.DataType, err)
		}
	}

	log.Println("done rotation: ", siteID)

	return nil
}

func rotate(siteID, dataType string, rotation *Rotation, batchSize int) error {

	activeTable := TableName(siteID, dataType)
	rotatingTable := activeTable + "_" + rotating

	if rows, err := datasource.GetConn().Query("SHOW TABLES LIKE '" + rotatingTable + "'"); err != nil {
		log.Println("error check rotating table: ", err)
		return err
	} else {
		defer rows.Close()
		if rows.Next() {
			log.Println("error rotating in progress: ")
			return errors.New("rotation in progress")
		}
	}

	_, archives := getArchiveTables(siteID, dataType)
	for _, a := range archives {
		if a.Status != archived {
			log.Println("error archive activation in progress: ")
			return errors.New("archive activation in progress")
		}
	}

	activeTime := rotation.getActiveTime()

	_, err := datasource.GetConn().Exec(fmt.Sprintf(`
		CREATE TABLE %s LIKE %s
	`, rotatingTable, activeTable))

	if err != nil {
		log.Println("error create rotating table: ", err)
		return err
	}
	defer func() {
		datasource.GetConn().Exec("DROP TABLE " + rotatingTable)
		ClearArchiveTable(siteID, dataType)
	}()

	var id int64

	for {
		copySQL := fmt.Sprintf(`
			INSERT INTO
				%s
			SELECT
				*
			FROM
				%s
			WHERE
				data_time < ? AND id > ?
			ORDER BY id ASC LIMIT ?
		`, rotatingTable, activeTable)

		log.Println("do archive: ", copySQL, activeTime, id, batchSize)

		copyRet, err := datasource.GetConn().Exec(copySQL, activeTime, id, batchSize)

		if err != nil {
			log.Println("error rotating copy: ", err)
			return err
		}

		copyRows, _ := copyRet.RowsAffected()
		log.Printf("rotating [%s] %d rows copied", activeTable, copyRows)

		if copyRows == 0 {
			return nil
		}

		maxID, copyMin, copyMax, err := getMinMaxFromRotating(rotatingTable, nil, nil)
		if err != nil {
			return err
		}

		var archiveRows int64
		timeCut := activeTime

		for {
			archiveBegin, archiveEnd := rotation.getArchiveTime(timeCut)

			log.Println("check archive: ", archiveBegin, archiveEnd, copyMin, copyMax)
			if archiveEnd.Before(copyMin) {
				log.Println("error rotating : out of range")
				break
			}

			_, min, max, err := getMinMaxFromRotating(rotatingTable, &archiveBegin, &archiveEnd)
			if err != nil {
				log.Println("error archive get min max: ", err)
				return err
			}

			if min.IsZero() {
				log.Println("no archive data in this interval")
				timeCut = archiveBegin.AddDate(0, 0, -1)
				continue
			} else {

				doRows, err := doArchiveOneTrip(siteID, dataType, archiveBegin, archiveEnd, min, max, activeTable, rotatingTable)
				if err != nil {
					log.Println("error archive: ", err)
					return err
				}

				archiveRows += doRows

				if archiveRows == copyRows {
					break
				}

			}
		}

		id = maxID
	}
}

func getMinMaxFromRotating(rotatingTable string, archiveBegin, archiveEnd *time.Time) (maxID int64, min, max time.Time, e error) {

	SQL := fmt.Sprintf(`
		SELECT
			MAX(id), MIN(data_time), MAX(data_time)
		FROM
			%s
	`, rotatingTable)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if archiveBegin != nil {
		whereStmts = append(whereStmts, "data_time >= ?")
		values = append(values, archiveBegin)
	}
	if archiveEnd != nil {
		whereStmts = append(whereStmts, "data_time < ?")
		values = append(values, archiveEnd)
	}

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get min max data_time from rotating: ", err)
		e = err
		return
	}
	defer rows.Close()

	if rows.Next() {
		var idNull sql.NullInt64
		var minNull, maxNull sql.NullTime
		if err := rows.Scan(&idNull, &minNull, &maxNull); err != nil {
			log.Println("error scan min max data_time from rotating: ", rotatingTable, err)
			e = err
			return
		}
		if idNull.Valid {
			maxID = idNull.Int64
		}
		if minNull.Valid {
			min = minNull.Time
		}
		if maxNull.Valid {
			max = maxNull.Time
		}
	}

	log.Println("get min max from rotating: ", rotatingTable, archiveBegin, archiveEnd, maxID, min, max)

	return
}

func createArchiveTable(activeTable, archiveTable string) error {

	log.Println("create archive table: ", archiveTable)

	rows, err := datasource.GetConn().Query(fmt.Sprintf("SHOW COLUMNS FROM %s", activeTable))
	if err != nil {
		log.Println("error rotating create archive show columns: ", activeTable, archiveTable, err)
		return err
	}
	defer rows.Close()

	fields := make([]string, 0)

	for rows.Next() {
		var field, fieldType, fieldNull, fieldKey, fieldDefault, fieldExtra sql.NullString

		if err := rows.Scan(&field, &fieldType, &fieldNull, &fieldKey, &fieldDefault, &fieldExtra); err != nil {
			log.Println("error rotating create archive scan show columns: ", activeTable, archiveTable, err)
			return err
		}

		if field.Valid && fieldType.Valid {
			fields = append(fields, fmt.Sprintf("%s %s", field.String, fieldType.String))
		}
	}

	createSyntax := fmt.Sprintf(`CREATE TABLE %s (
		%s
	) ENGINE=ARCHIVE DEFAULT CHARSET=utf8`, archiveTable, strings.Join(fields, ",\n"))

	if _, err := datasource.GetConn().Exec(createSyntax); err != nil {
		log.Println("error rotating create archive: ", activeTable, archiveTable, createSyntax, err)
		return err
	}
	return nil
}

func doArchiveOneTrip(siteID, dataType string, archiveBegin, archiveEnd, activeBegin, activeEnd time.Time, activeTable, rotatingTable string) (int64, error) {
	found := false
	var archiveRows int64

	_, existsArchives := getArchiveTables(siteID, dataType)

	defer ClearArchiveTable(siteID, dataType)

	for _, exists := range existsArchives {

		if exists.Status != archived {
			log.Println("skip not archived table: ", exists)
			continue
		}

		log.Println("check exists: ", exists.TableName, exists.BeginTime, exists.EndTime, activeBegin, activeEnd)
		if exists.BeginTime.Equal(archiveBegin) {
			log.Println("found exists archive table:", exists.TableName)
			found = true
			if archiveEnd.Before(exists.EndTime) {
				log.Println("found inconsistent archive: beyond range", exists.TableName, archiveBegin, archiveEnd)
			}

			doRows, err := doArchive(activeTable, fmt.Sprintf("`%s`", exists.TableName), rotatingTable, archiveBegin, archiveEnd)
			if err != nil {
				return 0, err
			}

			if doRows == 0 {
				log.Println("error archive done nothing: ", exists.TableName, archiveBegin, archiveEnd)
				return 0, errors.New("archive done nothing")
			}

			archiveRows += doRows

			if util.GetDate(activeEnd).After(exists.EndTime) {

				rename := fmt.Sprintf("%s_%s_%s", TableName(siteID, dataType), formatArchiveDate(exists.BeginTime), formatArchiveDate(activeEnd))

				SQL := fmt.Sprintf(`
					RENAME TABLE
						%s
					TO
						%s
				`, fmt.Sprintf("`%s`", exists.TableName), fmt.Sprintf("`%s`", rename))

				if _, err := datasource.GetConn().Exec(SQL); err != nil {
					log.Println("error rename archive table: ", exists.TableName, rename, activeBegin, activeEnd, err)
					return 0, err
				}

				exists.TableName = rename
				exists.EndTime = util.GetDate(activeEnd)
			}

			log.Printf("%d rows archived so far", archiveRows)
		}
	}

	if !found {
		archiveTable := fmt.Sprintf("`%s_%s_%s`", TableName(siteID, dataType), formatArchiveDate(archiveBegin), formatArchiveDate(activeEnd))
		if err := createArchiveTable(activeTable, archiveTable); err != nil {
			return 0, err
		}

		doRows, err := doArchive(activeTable, archiveTable, rotatingTable, archiveBegin, archiveEnd)
		if err != nil {
			return 0, err
		}

		if doRows == 0 {
			log.Println("error archive done nothing: ", archiveTable, archiveBegin, archiveEnd)
			return 0, errors.New("archive done nothing")
		}

		archiveRows += doRows
		log.Printf("%d rows archived so far", archiveRows)
	}

	return archiveRows, nil
}

func getTableColumns(txn *sql.Tx, table string) ([]string, error) {
	SQL := fmt.Sprintf(`
		SHOW COLUMNS FROM %s
	`, table)

	var rows *sql.Rows
	var err error
	if txn != nil {
		rows, err = txn.Query(SQL)
	} else {
		rows, err = datasource.GetConn().Query(SQL)
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var field string
		var t, null, key, def, extra interface{}

		if err := rows.Scan(&field, &t, &null, &key, &def, &extra); err != nil {
			return nil, err
		}
		result = append(result, field)
	}

	return result, nil
}

func doArchive(activeTable, archiveTable, rotatingTable string, archiveBegin, archiveEnd time.Time) (done int64, err error) {

	err = datasource.Txn(func(txn *sql.Tx) {

		columns, err := getTableColumns(txn, archiveTable)
		if err != nil {
			panic(err)
		}

		SQL := fmt.Sprintf(`
			INSERT IGNORE INTO
				%s
			SELECT
				%s
			FROM
				%s
			WHERE
				data_time >= ? AND data_time < ?
		`, archiveTable, strings.Join(columns, ","), rotatingTable)

		insertRet, err := txn.Exec(SQL, archiveBegin, archiveEnd)
		if err != nil {
			log.Println("error rotating to archive: ", archiveTable, SQL, err)
			panic(err)
		}

		done, _ = insertRet.RowsAffected()
		log.Printf("%d rows inserted into %s rotating %+v - %+v", done, archiveTable, archiveBegin, archiveEnd)

		if done == 0 {
			return
		}

		var deleteRows int64

		if activeTable == rotatingTable {
			deleteRet, err := txn.Exec(fmt.Sprintf(`
				DELETE b FROM
					%s b
				WHERE
					b.data_time >= ? AND b.data_time < ?
			`, rotatingTable), archiveBegin, archiveEnd)
			if err != nil {
				log.Println("error rotating deleting data:", err)
				panic(err)
			}
			deleteRows, _ = deleteRet.RowsAffected()
		} else {
			deleteRet, err := txn.Exec(fmt.Sprintf(`
				DELETE a, b FROM
					%s a
				INNER JOIN
					%s b
				ON
					a.id = b.id
				WHERE
					b.data_time >= ? AND b.data_time < ?
			`, activeTable, rotatingTable), archiveBegin, archiveEnd)
			if err != nil {
				log.Println("error rotating deleting data:", err)
				panic(err)
			}
			deleteRows, _ = deleteRet.RowsAffected()
			deleteRows = deleteRows / 2

		}
		log.Printf("%d rows deleted from %s rotating %+v - %+v", deleteRows, activeTable, archiveBegin, archiveEnd)
		if deleteRows != done {
			log.Printf("error rotate %s: rows not match [%d deleted] [%d inserted]", activeTable, deleteRows, done)
			panic(errors.New("delete rows not match"))
		}
	})

	return
}

func TriggerArchiveRollback(siteID, dataType, table string) (timer *time.Timer) {

	log.Println("trigger rollback activated archive: ", siteID, dataType, table)

	if !Config.IsEnvironmentArchiveWorker {

		rotationPersistentClientLock.RLock()
		defer rotationPersistentClientLock.RUnlock()

		for conn := range rotationPersistentClient {
			if err := send(conn, &TriggerArchiveRollbackReq{SiteID: siteID, DataType: dataType, Table: table}); err != nil {
				log.Println("error send ipc trigger archive rollback: ", err)
			}
		}

		return
	}

	dataModule, err := GetModule(siteID)
	if err != nil {
		return
	}

	defer func() {
		if timer == nil {
			archiveRollbackCallLock.Lock()
			defer archiveRollbackCallLock.Unlock()

			tableTimers, exists := archiveRollbackCalls[siteID]
			if !exists {
				tableTimers = make(map[string]*time.Timer)
				archiveRollbackCalls[siteID] = tableTimers
			}

			tableTimers[dataType] = time.AfterFunc(time.Minute*dataModule.ArchiveActiveTimeoutMin, func() {
				rollbackArchive(siteID, dataType)

				archiveRollbackCallLock.Lock()
				defer archiveRollbackCallLock.Unlock()

				timer = archiveRollbackCalls[siteID][dataType]
				if timer != nil {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
				}

				delete(archiveRollbackCalls[siteID], dataType)

				if len(archiveRollbackCalls[siteID]) == 0 {
					delete(archiveRollbackCalls, siteID)
				}
			})
		}
	}()

	archiveRollbackCallLock.RLock()
	defer archiveRollbackCallLock.RUnlock()

	timer = archiveRollbackCalls[siteID][dataType]

	if timer != nil {
		if timer.Stop() {
			timer.Reset(time.Minute * dataModule.ArchiveActiveTimeoutMin)
			log.Println("trigger rollback activated archive reset timer: ", siteID, dataType, table)
		}
	}

	return
}

func ActivateArchive(siteID, dataType, table string) error {

	if !Config.IsEnvironmentArchiveWorker {

		rotationPersistentClientLock.RLock()
		defer rotationPersistentClientLock.RUnlock()

		for conn := range rotationPersistentClient {
			if err := send(conn, &ArchiveActivateReq{SiteID: siteID, DataType: dataType, Table: table}); err != nil {
				log.Println("error send ipc activate archive: ", err)
			}
		}

		return nil
	}

	tableName := siteID + "_" + table

	dataModule, err := GetModule(siteID)
	if err != nil {
		return err
	}

	if err := activateArchive(siteID, dataType, tableName, dataModule.RotationBatchSize); err != nil {
		return err
	}

	return nil
}

func activateArchive(siteID, dataType, tableName string, batchSize int) error {

	if batchSize <= 0 {
		batchSize = 10000
	}

	log.Println("activate archive: ", siteID, tableName, batchSize)

	if rows, err := datasource.GetConn().Query("SHOW TABLES LIKE '" + tableName + "'"); err != nil {
		log.Println("error check rotating table: ", err)
		return err
	} else {
		defer rows.Close()
		if !rows.Next() {
			log.Println("error activate table: not exists ", tableName)
			return errors.New("not exists")
		}
	}

	activeTable := TableName(siteID, dataType)
	rotatingTable := activeTable + "_" + rotating

	if rows, err := datasource.GetConn().Query("SHOW TABLES LIKE '" + rotatingTable + "'"); err != nil {
		log.Println("error check rotating table: ", err)
		return err
	} else {
		defer rows.Close()
		if rows.Next() {
			log.Println("error rotating in progress: ")
			return errors.New("rotation in progress")
		}
	}

	activatingTable := tableName + "_" + activating
	if rows, err := datasource.GetConn().Query("SHOW TABLES LIKE '" + activatingTable + "'"); err != nil {
		log.Println("error check activating table: ", err)
		return err
	} else {
		defer rows.Close()
		if rows.Next() {
			log.Println("error activating in progress: ")
			return errors.New("activating in progress")
		}
	}

	activatedTable := tableName + "_" + active
	if rows, err := datasource.GetConn().Query("SHOW TABLES LIKE '" + activatedTable + "'"); err != nil {
		log.Println("error check activating table: ", err)
		return err
	} else {
		defer rows.Close()
		if rows.Next() {
			log.Println("error already activated: ")
			return errors.New("already activated")
		}
	}

	SQL := fmt.Sprintf(`
		RENAME TABLE
			%s
		TO
			%s
	`, fmt.Sprintf("`%s`", tableName), fmt.Sprintf("`%s`", activatingTable))

	if _, err := datasource.GetConn().Exec(SQL); err != nil {
		log.Println("error rename activating table: ", tableName, activatingTable, err)
		return err
	}

	ClearArchiveTable(siteID, dataType)

	go func() (e error) {

		sourceColumns, err := getTableColumns(nil, activatingTable)
		if err != nil {
			return err
		}

		tmpTable := activatingTable + "target"

		_, err = datasource.GetConn().Exec(fmt.Sprintf(`
			CREATE TABLE %s LIKE %s
		`, tmpTable, activeTable))

		if err != nil {
			log.Println("error create rotating table: ", err)
			return err
		}

		targetColumns, err := getTableColumns(nil, tmpTable)
		if err != nil {
			return err
		}

		columns := make([]string, 0)
		for _, col := range targetColumns {
			exists := false
			for _, source := range sourceColumns {
				if col == source {
					columns = append(columns, col)
					exists = true
					break
				}
			}

			if !exists {
				switch col {
				case ORIGIN_DATA:
					fallthrough
				case REVIEWED:
					fallthrough
				case FLAG:
					columns = append(columns, "")
				default:
					columns = append(columns, "0")
				}
			}
		}

		rows, err := datasource.GetConn().Query(fmt.Sprintf(`
			SELECT COUNT(1) FROM %s
		`, activatingTable))

		if err != nil {
			log.Println("error create rotating table: ", err)
			return err
		}

		var total int
		if rows.Next() {
			rows.Scan(&total)
		}
		rows.Close()

		var count int
		var totalRows int64

		defer func() {
			if e != nil {
				log.Println("error activate ", tableName, err)

				SQL = fmt.Sprintf(`
					RENAME TABLE
						%s
					TO
						%s
				`, fmt.Sprintf("`%s`", activatingTable), fmt.Sprintf("`%s`", tableName))

				if _, err := datasource.GetConn().Exec(SQL); err != nil {
					log.Println("fatal error rename activating table after failure: ", activatingTable, tableName, err)
				}

				if _, err := datasource.GetConn().Exec(fmt.Sprintf("DROP TABLE %s", tmpTable)); err != nil {
					log.Println("error drop tmp table after failure: ", tmpTable, err)
				}
			}
		}()

		for {

			SQL := fmt.Sprintf(`
				INSERT IGNORE INTO
					%s
				SELECT
					%s
				FROM
					%s activating
				LIMIT ?, ?
			`, tmpTable, strings.Join(columns, ","), activatingTable)

			ret, err := datasource.GetConn().Exec(SQL, count*batchSize, batchSize)
			if err != nil {
				log.Println("error activating: ", activatingTable, SQL, err)
				e = err
				return
			}

			rowsAffected, err := ret.RowsAffected()
			if err != nil {
				e = err
				return
			} else if rowsAffected == 0 && count*batchSize > total {
				log.Println("activate archive complete: ", count*batchSize, totalRows, total)
				break
			}
			totalRows += rowsAffected
			count++
			log.Println("activating in progress : ", activatingTable, rowsAffected, totalRows)
		}

		SQL = fmt.Sprintf(`
			RENAME TABLE
				%s
			TO
				%s
		`, fmt.Sprintf("`%s`", tmpTable), fmt.Sprintf("`%s`", activatedTable))

		if _, err := datasource.GetConn().Exec(SQL); err != nil {
			log.Println("fatal error rename activated table: ", activatedTable, tableName, err)
			e = err
			return
		}

		SQL = fmt.Sprintf(`
			RENAME TABLE
				%s
			TO
				%s
		`, fmt.Sprintf("`%s`", activatingTable), fmt.Sprintf("`%s_%s`", tableName, activated))

		if _, err := datasource.GetConn().Exec(SQL); err != nil {
			log.Println("fatal error rename activating table: ", activatingTable, tableName, err)
			e = err
			return
		}

		TriggerArchiveRollback(siteID, dataType, tableName)
		ClearArchiveTable(siteID, dataType)

		return
	}()

	return nil
}

func rollbackArchive(siteID, dataType string) error {

	log.Println("rollback archive run: ", siteID, dataType)

	_, archives := getArchiveTables(siteID, dataType)
	log.Println("rollback archive tables: ", len(archives))
	for _, archive := range archives {
		if archive.Status == active {

			log.Println("rollback archive: ", archive.TableName, archive.BeginTime, archive.EndTime)

			activatedTable := archive.TableName

			if _, err := datasource.GetConn().Exec(fmt.Sprintf("DROP TABLE %s", activatedTable)); err != nil {
				log.Println("error drop table: ", activatedTable, err)
				return err
			}

			archiveName := strings.TrimSuffix(activatedTable, "_"+active)

			SQL := fmt.Sprintf(`
				RENAME TABLE
					%s
				TO
					%s
			`, fmt.Sprintf("`%s_%s`", archiveName, activated), fmt.Sprintf("`%s`", archiveName))
			if _, err := datasource.GetConn().Exec(SQL); err != nil {
				log.Println("fatal error rename activated table: ", activatedTable, archive.TableName, err)
				return err
			}
		}
	}

	ClearArchiveTable(siteID, dataType)

	return nil
}

func recoverInterruptedProgress() {

	for _, dt := range []string{REAL_TIME, MINUTELY, HOURLY, DAILY} {

		for _, suffix := range []string{activating, rotating} {

			SQL := fmt.Sprintf("SHOW TABLES LIKE '%%_%s_%s'", dt, suffix)

			log.Println("detecting interrupted table: ", SQL)

			rows, err := datasource.GetConn().Query(SQL)
			if err != nil {
				log.Println("error show in progress tables: ", dt, suffix, err)
				return
			}
			defer rows.Close()

			tables := make([]string, 0)

			for rows.Next() {
				var t string
				if err := rows.Scan(&t); err != nil {
					log.Println("error show archive tables: ", err)
					continue
				}

				tables = append(tables, t)
			}

			for _, t := range tables {
				if err := errorRecoverTable(suffix, t); err != nil {
					log.Println("error error recover: ", t, err)
				}
			}

		}
	}
}

func errorRecoverTable(suffix, table string) error {

	log.Println("detect interrupted table: ", table)

	switch suffix {
	case activating:
		if _, err := datasource.GetConn().Exec(fmt.Sprintf("RENAME TABLE %s TO %s", table, strings.ReplaceAll(table, "_"+activating, ""))); err != nil {
			log.Println("error recover activating table: ", table, strings.ReplaceAll(table, "_"+activating, ""), err)
			return err
		}
	case rotating:
		if _, err := datasource.GetConn().Exec(fmt.Sprintf("DROP TABLE %s", table)); err != nil {
			log.Println("error recover rotating table: ", table, err)
			return err
		}
	}

	return nil
}
