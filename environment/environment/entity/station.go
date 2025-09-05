package entity

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

const (
	ACTIVE      = "ACTIVE"
	INACTIVE    = "INACTIVE"
	MAINTENANCE = "MAINTENANCE"
)

var e_need_mn = errors.New("需要MN")
var e_duplicate_mn = errors.New("MN已存在")
var e_need_entity = errors.New("需要所属组织")
var e_station_not_found = errors.New("排放点不存在")

type Station struct {
	ID          int                    `json:"ID"`
	EntityID    int                    `json:"entityID"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	OnlineTime  util.Time              `json:"onlineTime"`
	Status      string                 `json:"status"`
	MN          string                 `json:"mn"`
	Protocol    string                 `json:"protocol"`
	Redirect    string                 `json:"redirect"`
	Ext         map[string]interface{} `json:"ext"`
}

const stationColumns = "station.id, station.entity_id, station.name, station.description, station.online_time, station.status, station.mn, station.protocol, station.redirect, station.ext"

func stationTableName(siteID string) string {
	return siteID + "_station"
}

func (s *Station) scan(rows *sql.Rows) error {
	var ext string
	if err := rows.Scan(&s.ID, &s.EntityID, &s.Name, &s.Description, &s.OnlineTime, &s.Status, &s.MN, &s.Protocol, &s.Redirect, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(ext), &s.Ext); err != nil {
		return err
	}
	return nil
}

func (s *Station) GetEntityID() int { return s.EntityID }

func (s *Station) validate(siteID string) error {
	if s.Name == "" {
		return e_need_name
	}

	if s.EntityID <= 0 {
		return e_need_entity
	}

	if s.Redirect != "" {
		redirects := strings.Split(s.Redirect, ";")
		for _, r := range redirects {
			parts := strings.Split(r, "#")

			if len(parts) > 1 {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(parts[1]), &params); err != nil {
					return fmt.Errorf("转发参数解析错误: %s", err.Error())
				}
			}
		}
	}

	if s.Protocol != "" {
		if s.MN == "" {
			return e_need_mn
		}

		s.MN = strings.TrimSpace(s.MN)

		rows, err := datasource.GetConn().Query(fmt.Sprintf(`
			SELECT
				%s
			FROM
				%s station
			WHERE
				mn = ? AND id != ?
		`, stationColumns, stationTableName(siteID)), s.MN, s.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			return e_duplicate_mn
		}
	}

	if time.Time(s.OnlineTime).IsZero() {
		s.OnlineTime = util.Time(util.DefaultMax)
	}

	switch s.Status {
	case ACTIVE:
	case INACTIVE:
	case MAINTENANCE:
	default:
		s.Status = ACTIVE
	}

	if s.Ext == nil {
		s.Ext = make(map[string]interface{})
	}

	return nil
}

func (s *Station) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := s.validate(siteID); err != nil {
		return err
	}

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	ext, err := json.Marshal(s.Ext)
	if err != nil {
		return err
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(entity_id,name,description,online_time,status,mn,protocol, redirect, ext)
		VALUES
			(?,?,?,?,?,?,?,?,?)
	`, stationTableName(siteID)), s.EntityID, s.Name, s.Description, time.Time(s.OnlineTime), s.Status, s.MN, s.Protocol, s.Redirect, string(ext)); err != nil {
		log.Println("error insert station: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert station: ", err)
		return err
	} else {
		s.ID = int(id)
	}

	return nil
}
func (s *Station) Update(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := s.validate(siteID); err != nil {
		return err
	}

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	ext, err := json.Marshal(s.Ext)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE 
			%s
		SET
			entity_id=?,name=?,description=?,online_time=?,status=?,mn=?,protocol=?,redirect=?,ext=?
		WHERE
			id=?
	`, stationTableName(siteID)), s.EntityID, s.Name, s.Description, time.Time(s.OnlineTime), s.Status, s.MN, s.Protocol, s.Redirect, string(ext), s.ID); err != nil {
		log.Println("error update station: ", err)
		return err
	}

	setStationCache(siteID, false, s)

	return nil
}

func (s *Station) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, stationTableName(siteID)), s.ID); err != nil {
		log.Println("error delete station: ", err)
		return err
	}

	LoadStation(siteID)

	return nil
}

func (s *Station) delete(siteID string, txn *sql.Tx) error {
	if _, err := txn.Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, stationTableName(siteID)), s.ID); err != nil {
		log.Println("error delete station: ", err)
		return err
	}

	return nil
}

func GetStation(siteID string, stationID ...int) ([]*Station, error) {
	if len(stationID) == 0 {
		return []*Station{}, nil
	}
	return GetStations(siteID, nil, nil, "", "", "", stationID...)
}

func GetStations(siteID string, cids []int, entityID []int, status, protocol, q string, stationID ...int) ([]*Station, error) {
	result := make([]*Station, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s station
	`, stationColumns, stationTableName(siteID))

	if len(cids) > 0 {
		joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "station", "", cids...)
		if err != nil {
			return nil, err
		}
		if joinSQL == "" {
			return result, nil
		}

		SQL += joinSQL

		whereStmts = append(whereStmts, joinWhere...)
		values = append(values, joinValues...)
	}

	if len(entityID) > 0 {
		if len(entityID) == 1 {
			whereStmts = append(whereStmts, "station.entity_id = ?")
			values = append(values, entityID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range entityID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("station.entity_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if status != "" {
		whereStmts = append(whereStmts, "station.status = ?")
		values = append(values, status)
	}

	if protocol != "" {
		whereStmts = append(whereStmts, "station.protocol = ?")
		values = append(values, protocol)
	}

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(station.name LIKE ? OR station.mn LIKE ? OR station.description LIKE ? OR station.ext LIKE ?)")
		values = append(values, qq, qq, qq, qq)
	}

	if len(stationID) != 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "station.id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("station.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get station: ", SQL, values, err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var s Station
		if err := s.scan(rows); err != nil {
			log.Println("error get station: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}

func AddStationCategory(siteID string, stationID, categoryID int) error {
	return category.AddCategoryMapping(siteID, "station", stationID, categoryID)
}

func DeleteStationCategory(siteID string, stationID, categoryID int) error {
	return category.DeleteCategoryMapping(siteID, "station", stationID, categoryID)
}
