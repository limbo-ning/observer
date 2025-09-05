package entity

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/category"
)

type Entity struct {
	ID        int                    `json:"ID"`
	Name      string                 `json:"name"`
	Address   string                 `json:"address"`
	Longitude float64                `json:"longitude"`
	Latitude  float64                `json:"latitude"`
	GeoType   string                 `json:"geoType"`
	Ext       map[string]interface{} `json:"ext"`
}

var e_need_name = errors.New("需要名称")

const entityColumns = "entity.id, entity.name, entity.address, entity.longitude, entity.latitude, entity.geo_type, entity.ext"

func entityTableName(siteID string) string {
	return siteID + "_entity"
}

func (e *Entity) scan(rows *sql.Rows) error {

	var ext string

	if err := rows.Scan(&e.ID, &e.Name, &e.Address, &e.Longitude, &e.Latitude, &e.GeoType, &ext); err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(ext), &e.Ext); err != nil {
		return err
	}

	return nil
}

func (s *Entity) GetEntityID() int { return s.ID }

func (s *Entity) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	if s.Name == "" {
		return e_need_name
	}

	if s.Ext == nil {
		s.Ext = make(map[string]interface{})
	}

	ext, _ := json.Marshal(s.Ext)

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name,address,longitude,latitude,geo_type,ext)
		VALUES
			(?,?,?,?,?,?)
	`, entityTableName(siteID)), s.Name, s.Address, s.Longitude, s.Latitude, s.GeoType, string(ext)); err != nil {
		log.Println("error insert entity: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert entity: ", err)
		return err
	} else {
		s.ID = int(id)
	}

	return nil
}
func (s *Entity) Update(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	if s.Name == "" {
		return e_need_name
	}

	if s.Ext == nil {
		s.Ext = make(map[string]interface{})
	}

	ext, _ := json.Marshal(s.Ext)

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE 
			%s
		SET
			name=?,address=?,longitude=?,latitude=?,geo_type=?,ext=?
		WHERE
			id=?
	`, entityTableName(siteID)), s.Name, s.Address, s.Longitude, s.Latitude, s.GeoType, string(ext), s.ID); err != nil {
		log.Println("error update entity: ", err)
		return err
	}

	return nil
}

func (s *Entity) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	if s.ID <= 0 {
		return errors.New("无ID")
	}

	if err := CheckAuth(siteID, actionAuth, s.GetEntityID(), ACTION_ENTITY_EDIT); err != nil {
		return err
	}

	stations, err := GetStations(siteID, nil, []int{s.ID}, "", "", "")
	if err != nil {
		return err
	}

	if len(stations) > 0 {
		return errors.New("点位不为空")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, entityTableName(siteID)), s.ID); err != nil {
		log.Println("error delete entity: ", err)
		return err
	}

	return nil
}

func GetEntities(siteID string, entityID ...int) ([]*Entity, error) {
	if len(entityID) == 0 {
		return []*Entity{}, nil
	}
	return GetEntityList(siteID, authority.ActionAuthSet{{Action: ACTION_ADMIN_VIEW}}, "", "", nil, nil, "", entityID...)
}

func GetEntityList(siteID string, actionAuth authority.ActionAuthSet, authAction string, empowerType string, empowerID []string, cids []int, q string, entityID ...int) ([]*Entity, error) {

	result := make([]*Entity, 0)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s entity
	`, entityColumns, entityTableName(siteID))

	if authAction == "" {
		authAction = ACTION_ENTITY_VIEW
	}
	authSQL, authWhere, authValues, err := authority.JoinEmpower(siteID, "entity", actionAuth, AdminActions, authAction, "entity", "id", empowerType, empowerID...)
	if err != nil {
		return nil, err
	}
	SQL += authSQL
	if authWhere != nil {
		whereStmts = append(whereStmts, authWhere...)
	}
	if authValues != nil {
		values = append(values, authValues...)
	}

	if len(cids) > 0 {
		joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping(siteID, "entity", "", cids...)
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

	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(entity.name LIKE ? OR entity.address LIKE ?)")
		values = append(values, qq, qq)
	}

	if len(entityID) != 0 {
		if len(entityID) == 1 {
			whereStmts = append(whereStmts, "entity.id = ?")
			values = append(values, entityID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range entityID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("entity.id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(whereStmts) > 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	SQL += "\nGROUP BY entity.id"

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get entity: ", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var s Entity
		if err := s.scan(rows); err != nil {
			log.Println("error get entity: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}

func AddEntityCategory(siteID string, entityID, categoryID int) error {
	return category.AddCategoryMapping(siteID, "entity", entityID, categoryID)
}

func DeleteEntityCategory(siteID string, entityID, categoryID int) error {
	return category.DeleteCategoryMapping(siteID, "entity", entityID, categoryID)
}
