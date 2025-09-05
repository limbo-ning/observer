package site

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
)

var E_site_module_not_found = errors.New("功能模块未启用")
var E_site_module_indebt = errors.New("功能模块欠费")
var E_site_module_inactive = errors.New("功能模块关闭")

var lock sync.RWMutex
var pool = make(map[string]*sitePool)

var siteModuleNotFound = &SiteModule{}

type sitePool struct {
	Lock    sync.RWMutex
	Pool    map[string]*SiteModule
	HitTime time.Time
}

type SiteModule struct {
	ModuleID   string                 `json:"ID"`
	Status     string                 `json:"status"`
	Param      map[string]interface{} `json:"param"`
	CreateTime util.Time              `json:"createTime"`
	UpdateTime util.Time              `json:"updateTime"`
}

const siteModuleColumn = "site_module.module_id,site_module.status,site_module.param"

func (m *SiteModule) scan(rows *sql.Rows) error {
	var param string
	if err := rows.Scan(&m.ModuleID, &m.Status, &param); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(param), &m.Param); err != nil {
		return err
	}
	return nil
}

func wildcastModuleIDs(moduleID string) []string {
	IDs := make([]string, 0)

	if moduleID == "" {
		return IDs
	}

	parts := strings.Split(moduleID, "_")
	if len(parts) == 0 {
		IDs = append(IDs, moduleID)
		return IDs
	}

	for i := range parts {
		wildcast := IDs[0:i]
		wildcast = append(wildcast, "*")
		IDs = append(IDs, strings.Join(wildcast, "_"))
	}

	IDs = append(IDs, moduleID)

	return IDs
}

func siteModuleTable(siteID string) string {
	return siteID + "_site_module"
}

func GetSiteModules(siteID, moduleID, prefix string) ([]*SiteModule, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if moduleID != "" {
		whereStmts = append(whereStmts, "module_id = ?")
		values = append(values, moduleID)
	}

	if prefix != "" {
		whereStmts = append(whereStmts, "module_id LIKE ?")
		values = append(values, prefix+"%")
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s site_module
	`, siteModuleColumn, siteModuleTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]*SiteModule, 0)
	for rows.Next() {
		var sm SiteModule
		if err := sm.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &sm)
	}

	return result, nil
}

func GetSiteModule(siteID, moduleID string, flags ...bool) (string, *SiteModule, error) {

	fromDataSource := false
	if len(flags) >= 2 && flags[1] {
		fromDataSource = true
	}
	sm, err := fetchSiteModule(siteID, moduleID, fromDataSource)
	if err != nil {
		return "", nil, err
	}

	if sm != siteModuleNotFound {
		switch sm.Status {
		case STATUS_INACTIVE:
			return "", sm, E_site_module_inactive
		case STATUS_INDEBT:
			return "", sm, E_site_module_indebt
		default:
			return siteID, sm, nil
		}
	}

	if len(flags) >= 1 && flags[0] {
		thisSite, err := GetSite(siteID)
		if err != nil {
			return "", nil, err
		}
		if thisSite.ParentSite != "" {
			return GetSiteModule(thisSite.ParentSite, moduleID, flags...)
		}
	}

	return "", nil, E_site_module_not_found
}

func fetchSiteModule(siteID, moduleID string, fromDatasource bool) (*SiteModule, error) {

	if !fromDatasource {
		sm := fetchSiteModuleFromCache(siteID, moduleID)
		if sm != nil {
			return sm, nil
		}
	}

	return fetchSiteModuleFromDatasource(siteID, moduleID)
}

func fetchSiteModuleFromCache(siteID, moduleID string) *SiteModule {

	lock.RLock()
	defer lock.RUnlock()

	siteCache, exists := pool[siteID]
	if exists {
		siteCache.HitTime = time.Now()
		siteCache.Lock.RLock()
		defer siteCache.Lock.RUnlock()

		sm, exists := siteCache.Pool[moduleID]
		if exists {
			return sm
		}
	}

	return nil
}

func fetchSiteModuleFromDatasource(siteID, moduleID string) (*SiteModule, error) {

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s site_module
		WHERE
			module_id = ?
	`, siteModuleColumn, siteModuleTable(siteID)), moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var m *SiteModule
	if rows.Next() {
		m = &SiteModule{}
		if err := m.scan(rows); err != nil {
			return nil, err
		}
	} else {
		log.Println("site module not found in ds")
		m = siteModuleNotFound
	}

	lock.Lock()
	defer lock.Unlock()

	siteCache, exists := pool[siteID]
	if !exists {
		siteCache = new(sitePool)
		siteCache.HitTime = time.Now()
		siteCache.Pool = make(map[string]*SiteModule)
		pool[siteID] = siteCache
	}

	siteCache.Lock.Lock()
	defer siteCache.Lock.Unlock()
	siteCache.Pool[moduleID] = m

	return m, nil
}

func GetSiteModuleWithTxn(siteID string, txn *sql.Tx, moduleID string, forUpdate bool) (*SiteModule, error) {

	SQL := fmt.Sprintf(fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s site_module
		WHERE
			module_id = ?
	`, siteModuleColumn, siteModuleTable(siteID)))

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var m SiteModule
		if err := m.scan(rows); err != nil {
			return nil, err
		}
		switch m.Status {
		case STATUS_INACTIVE:
			return &m, E_site_module_inactive
		case STATUS_INDEBT:
			return &m, E_site_module_indebt
		default:
			return &m, nil
		}
	}

	return nil, E_site_module_not_found
}

func (sm *SiteModule) Save(siteID string, txn *sql.Tx) error {
	param, _ := json.Marshal(sm.Param)

	if _, err := txn.Exec(`
		INSERT INTO `+siteModuleTable(siteID)+`
			(module_id, status, param)
		VALUE
			(?,?,?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status), param = VALUES(param), update_time = Now()
	`, sm.ModuleID, sm.Status, string(param)); err != nil {
		return err
	}

	lock.RLock()
	defer lock.RUnlock()

	siteCache, exists := pool[siteID]
	if exists {
		siteCache.Lock.Lock()
		defer siteCache.Lock.Unlock()

		if _, exists := siteCache.Pool[sm.ModuleID]; exists {
			siteCache.Pool[sm.ModuleID] = sm
		}
	}

	return nil
}

func JoinSiteModuleAuth(siteID, joinTable, joinColumn string) (string, []string, []interface{}) {

	joinSQL := fmt.Sprintf(`
		LEFT JOIN
			%s sm
		ON
			sm.module_id = %s.%s
	`, siteModuleTable(siteID), joinTable, joinColumn)

	joinWhere := make([]string, 0)
	joinValues := make([]interface{}, 0)

	joinWhere = append(joinWhere, fmt.Sprintf("((sm.id IS NOT NULL AND sm.status = ?) OR (sm.id IS NULL AND %s.%s = ?))", joinTable, joinColumn))
	joinValues = append(joinValues, STATUS_ACTIVE, "*")

	return joinSQL, joinWhere, joinValues
}
