package site

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/gps"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/relation"
	"obsessiontech/environment/site/initialization"
)

func init() {
	initialization.Register(MODULE_SITE, []string{"site_module"})
}

const TYPE_PUBLIC = "PUBLIC"
const TYPE_HIDDEN = "HIDDEN"

const (
	STATUS_ACTIVE   = "ACTIVE"
	STATUS_INACTIVE = "INACTIVE"
	STATUS_INDEBT   = "INDEBT"
)

type Site struct {
	SiteID     string                 `json:"ID"`
	UID        int                    `json:"UID"`
	Favicon    string                 `json:"favicon"`
	Status     string                 `json:"status"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	ParentSite string                 `json:"parentSite"`
	Address    string                 `json:"address"`
	Longitude  float64                `json:"longitude"`
	Latitude   float64                `json:"latitude"`
	GeoType    string                 `json:"geoType"`
	Info       map[string]interface{} `json:"info"`
}

const siteColumns = "site.id, site.uid, site.favicon, site.status, site.name, site.parent_site, site.address, site.longitude, site.latitude, site.geo_type, site.info"

func (s *Site) scan(rows *sql.Rows, geoType string) error {
	var info string
	if err := rows.Scan(&s.SiteID, &s.UID, &s.Favicon, &s.Status, &s.Name, &s.ParentSite, &s.Address, &s.Longitude, &s.Latitude, &s.GeoType, &info); err != nil {
		return err
	}
	if geoType != "" && s.GeoType != "" {
		var err error
		s.Longitude, s.Latitude, err = gps.TranslateGeoType(s.Longitude, s.Latitude, s.GeoType, geoType)
		if err != nil {
			log.Println("error tranlate geo: ", err)
		} else {
			s.GeoType = geoType
		}
	}
	return json.Unmarshal([]byte(info), &s.Info)
}

func GenerateSiteID(UID int) string {

	bytesBuffer := bytes.NewBuffer([]byte{})
	if err := binary.Write(bytesBuffer, binary.BigEndian, time.Now().UnixNano()); err != nil {
		panic(err)
	}

	return hex.EncodeToString(bytesBuffer.Bytes())
}

func (site *Site) CheckAuth(siteID string, actionAuth authority.ActionAuthSet, args ...interface{}) error {

	for _, a := range actionAuth {
		action := strings.TrimPrefix(a.Action, MODULE_SITE+"#")

		switch action {
		case ACTION_ADMIN_EDIT_SITE:
		case ACTION_C_EDIT_SITE:
		default:
			continue
		}

		return nil
	}

	return errors.New("无权限")
}

func GetSite(siteID string) (*Site, error) {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			c_site site
		WHERE
			id = ?
	`, siteColumns), siteID)
	if err != nil {
		log.Println("error get site: ", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var s Site
		if err := s.scan(rows, ""); err != nil {
			return nil, err
		}

		return &s, nil
	}

	return nil, errors.New("站点不存在")
}

func GetSiteSeries(siteID, status, siteType, geoType string) ([]*Site, error) {
	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if siteID == "" {
		return nil, errors.New("请查询有效的siteID")
	}

	whereStmts = append(whereStmts, "( id = ? OR parent_site = ? )")
	values = append(values, siteID, siteID)

	switch status {
	case STATUS_ACTIVE:
	case STATUS_INACTIVE:
	case STATUS_INDEBT:
	default:
		status = STATUS_ACTIVE
	}

	whereStmts = append(whereStmts, "status = ?")
	values = append(values, status)

	switch siteType {
	case TYPE_HIDDEN:
	case TYPE_PUBLIC:
	default:
		siteType = TYPE_PUBLIC
	}

	whereStmts = append(whereStmts, "type = ?")
	values = append(values, siteType)

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			c_site site
		WHERE
			%s
	`, siteColumns, strings.Join(whereStmts, " AND ")), values...)
	if err != nil {
		log.Println("error get site series: ", err)
		return nil, err
	}
	defer rows.Close()

	result := make([]*Site, 0)

	for rows.Next() {
		var s Site
		if err := s.scan(rows, geoType); err != nil {
			return nil, err
		}

		result = append(result, &s)
	}

	return result, nil
}

func MySites(uid int, geoType string) ([]*Site, error) {
	result := make([]*Site, 0)

	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			%s
		FROM
			c_site site
		WHERE
			uid = ?
	`, siteColumns), uid)
	if err != nil {
		log.Println("error get site: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s Site
		if err := s.scan(rows, geoType); err != nil {
			return nil, err
		}

		result = append(result, &s)
	}

	return result, nil
}

func GrantSite(siteID string, from, to int) error {

	if from <= 0 || to <= 0 {
		return errors.New("需要登录用户")
	}

	return datasource.Txn(func(txn *sql.Tx) {
		site, err := getSiteWithTxn(siteID, "", txn, true)
		if err != nil {
			panic(err)
		}

		if site.UID != from {
			panic(errors.New("不是当前拥有者"))
		}

		site.UID = to

		if _, err := txn.Exec(`
			UPDATE
				c_site
			SET
				uid = ?
			WHERE
				id=?
		`, to, site.SiteID); err != nil {
			panic(err)
		}
	})
}

func getSiteWithTxn(siteID, geoType string, txn *sql.Tx, forUpdate bool) (*Site, error) {

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			c_site site
		WHERE
			id = ?
	`, siteColumns)

	if forUpdate {
		SQL += "FOR UPDATE"
	}

	rows, err := txn.Query(SQL, siteID)
	if err != nil {
		log.Println("error get site: ", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var s Site
		s.scan(rows, geoType)

		return &s, nil
	}

	return nil, errors.New("站点不存在")
}

func (site *Site) insert(txn *sql.Tx) error {

	if site.UID <= 0 {
		return errors.New("需要拥有者")
	}

	if site.Info == nil {
		site.Info = make(map[string]interface{})
	}
	info, _ := json.Marshal(site.Info)

	if site.Longitude > 0 || site.Latitude > 0 || site.GeoType != "" {
		var err error
		site.GeoType, err = gps.ValidateGeoType(site.GeoType)
		if err != nil {
			return err
		}
	}

	if _, err := txn.Exec(`
		INSERT INTO c_site
			(id, uid, favicon, status, name, parent_site, address, longitude, latitude, geo_type, info)
		VALUES
			(?,?,?,?,?,?,?,?,?,?,?)
	`, site.SiteID, site.UID, site.Favicon, site.Status, site.Name, site.ParentSite, site.Address, site.Longitude, site.Latitude, site.GeoType, string(info)); err != nil {
		return err
	}
	return nil
}

func (site *Site) Update(actionAuth authority.ActionAuthSet) error {

	if err := site.CheckAuth(site.SiteID, actionAuth); err != nil {
		return err
	}

	if site.Info == nil {
		site.Info = make(map[string]interface{})
	}
	info, _ := json.Marshal(site.Info)
	if site.Longitude > 0 || site.Latitude > 0 || site.GeoType != "" {
		var err error
		site.GeoType, err = gps.ValidateGeoType(site.GeoType)
		if err != nil {
			return err
		}
	}
	if _, err := datasource.GetConn().Exec(`
		UPDATE
			c_site
		SET
			favicon=?, status=?, name=?, address=?, longitude=?, latitude=?, geo_type=?, info=?
		WHERE
			id=?
	`, site.Favicon, site.Status, site.Name, site.Address, site.Longitude, site.Latitude, site.GeoType, string(info), site.SiteID); err != nil {
		return err
	}
	return nil
}

func GetCNames(siteID string) ([]string, error) {
	exists, err := relation.ExistRelations[string, string]("c", "site", "cname", "", []string{siteID}, nil)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(exists))
	for i, r := range exists {
		result[i] = *r.BID
	}

	return result, nil
}

func BindCName(siteID, cname string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		exists, err := relation.ExistRelationsWithTxn[string, string]("c", txn, "site", "cname", "", nil, []string{cname})
		if err != nil {
			panic(err)
		}
		if len(exists) != 0 {
			panic(errors.New("该域名已被占用"))
		}

		r := &relation.Relation[string, string]{
			A:   "site",
			B:   "cname",
			AID: &siteID,
			BID: &cname,
		}

		if err := r.Add(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func UnbindCName(siteID, cname string) error {
	return datasource.Txn(func(txn *sql.Tx) {
		exists, err := relation.ExistRelationsWithTxn[string, string]("c", txn, "site", "cname", "", nil, []string{cname})
		if err != nil {
			panic(err)
		}
		if len(exists) == 0 {
			panic(errors.New("该域名未被绑定"))
		}

		if err := exists[0].Delete(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func GetSiteIDByCName(cname string) (string, error) {
	exists, err := relation.ExistRelations[string, string]("c", "site", "cname", "", nil, []string{cname})
	if err != nil {
		return "", err
	}

	if len(exists) == 0 {
		return "", errors.New("没有对应站点")
	}

	return *exists[0].AID, nil
}
