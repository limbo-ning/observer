package monitor

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"obsessiontech/common/datasource"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/entity"
	"strconv"
	"strings"
)

const (
	COMPARATOR_E  = "="
	COMPARATOR_NE = "!="
	COMPARATOR_LE = "<="
	COMPARATOR_GE = ">="
	COMPARATOR_L  = "<"
	COMPARATOR_G  = ">"
)

type FlagLimit struct {
	ID             int        `json:"ID"`
	StationID      int        `json:"stationID"`
	MonitorID      int        `json:"monitorID"`
	Flag           string     `json:"flag"`
	Region         string     `json:"region"`
	regionSegments []segments `json:"-"`
}

func (l *FlagLimit) GetStationID() int { return l.StationID }

type segment struct {
	Comparator string
	Value      float64
}

var e_invalid_limit = errors.New("错误限值格式")
var e_invalid_limit_comparator = errors.New("错误限值比较符")
var e_invalid_limit_value = errors.New("错误限值数值")

type segments []*segment

func (c *segment) compare(value float64) bool {
	switch c.Comparator {
	case COMPARATOR_E:
		return value == c.Value
	case COMPARATOR_NE:
		return value != c.Value
	case COMPARATOR_G:
		return value > c.Value
	case COMPARATOR_L:
		return value < c.Value
	case COMPARATOR_LE:
		return value <= c.Value
	case COMPARATOR_GE:
		return value >= c.Value
	}

	return false
}

func (c *segment) parse(input string) error {

	var value string
	for _, r := range input {
		switch r {
		case '!':
			fallthrough
		case '=':
			fallthrough
		case '>':
			fallthrough
		case '<':
			if value != "" {
				return e_invalid_limit
			}
			c.Comparator += string([]rune{r})
		default:
			value += string([]rune{r})
		}
	}

	switch c.Comparator {
	case COMPARATOR_E:
	case COMPARATOR_GE:
	case COMPARATOR_G:
	case COMPARATOR_NE:
	case COMPARATOR_L:
	case COMPARATOR_LE:
	default:
		log.Println("error parse comparator: ", input, c.Comparator)
		return e_invalid_limit_comparator
	}

	var err error
	c.Value, err = strconv.ParseFloat(value, 64)
	if err != nil {
		log.Println("error parse value: ", input, err)
		return e_invalid_limit_value
	}

	return nil
}

const limitColumns = "flagLimit.id, flagLimit.monitor_id, flagLimit.station_id, flagLimit.flag, flagLimit.region"

func limitTableName(siteID string) string {
	return siteID + "_monitorflaglimit"
}

func (l *FlagLimit) scan(rows *sql.Rows) error {

	if err := rows.Scan(&l.ID, &l.MonitorID, &l.StationID, &l.Flag, &l.Region); err != nil {
		return err
	}

	if err := l.parseRegionSegment(); err != nil {
		return err
	}
	return nil
}

func (l *FlagLimit) parseRegionSegment() error {
	l.regionSegments = make([]segments, 0)
	if l.Region == "" {
		return nil
	}

	segments := strings.Split(l.Region, ";")
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		ands := strings.Split(seg, ",")
		segments := make([]*segment, 0)
		for _, part := range ands {
			result := new(segment)
			if err := result.parse(part); err != nil {
				return err
			}
			segments = append(segments, result)
		}

		l.regionSegments = append(l.regionSegments, segments)
	}

	return nil
}

func (l *FlagLimit) IsInRegion(value float64) (checked bool) {

	for _, s := range l.regionSegments {
		checked = true
		for _, each := range s {
			if !each.compare(value) {
				checked = false
				break
			}
		}
		if checked {
			return
		}
	}

	checked = false

	return false
}

func (l *FlagLimit) Validate(siteID string) error {

	if l.MonitorID <= 0 {
		return e_need_monitor_id
	}

	monitorModule, err := GetModule(siteID)
	if err != nil {
		return err
	}

	var flagInstance *Flag
	for _, f := range monitorModule.Flags {
		if f.Flag == l.Flag {
			flagInstance = f
			break
		}
	}

	if flagInstance == nil {
		return errors.New("标记不存在")
	}

	if err := l.parseRegionSegment(); err != nil {
		return err
	}

	if len(l.regionSegments) > 0 {
		if CheckFlag(FLAG_DATA_INVARIANCE, flagInstance.Bits) {
			if len(l.regionSegments) != 1 {
				return fmt.Errorf("错误的%s限值", flagInstance.Name)
			}
			if len(l.regionSegments[0]) != 1 {
				return fmt.Errorf("错误的%s限值", flagInstance.Name)
			}
			if l.regionSegments[0][0].Comparator != COMPARATOR_E && l.regionSegments[0][0].Comparator != COMPARATOR_GE {
				return fmt.Errorf("错误的%s限值", flagInstance.Name)
			}
			if l.regionSegments[0][0].Value <= 0 {
				return fmt.Errorf("错误的%s限值", flagInstance.Name)
			}
		}
	}

	return nil
}

func (l *FlagLimit) Add(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := l.Validate(siteID); err != nil {
		return err
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{l.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[l.StationID] {
		return errors.New("无权限")
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(monitor_id,station_id,flag,region)
		VALUES
			(?,?,?,?)
		ON DUPLICATE KEY UPDATE
		region=VALUES(region)
	`, limitTableName(siteID)), l.MonitorID, l.StationID, l.Flag, l.Region); err != nil {
		log.Println("error insert monitor flag limit: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert monitor flag limit: ", err)
		return err
	} else {
		l.ID = int(id)
	}

	return nil
}

func (l *FlagLimit) Update(siteID string, actionAuth authority.ActionAuthSet) error {

	if err := l.Validate(siteID); err != nil {
		return err
	}

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{l.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[l.StationID] {
		return errors.New("无权限")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			monitor_id=?,station_id=?,flag=?,region=?
		WHERE
			id = ?
	`, limitTableName(siteID)), l.MonitorID, l.StationID, l.Flag, l.Region, l.ID); err != nil {
		log.Println("error update monitor flag limit: ", err)
		return err
	}

	return nil
}

func (l *FlagLimit) Delete(siteID string, actionAuth authority.ActionAuthSet) error {

	if filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth, []int{l.StationID}, entity.ACTION_ENTITY_EDIT); err != nil {
		return err
	} else if !filtered[l.StationID] {
		return errors.New("无权限")
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, limitTableName(siteID)), l.ID); err != nil {
		log.Println("error delete monitor flag limit: ", err)
		return err
	}

	return nil
}

func GetFlagLimits(siteID string, stationID []int, monitorID []int, flag []string) ([]*FlagLimit, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if len(monitorID) > 0 {
		if len(monitorID) == 1 {
			whereStmts = append(whereStmts, "flagLimit.monitor_id = ?")
			values = append(values, monitorID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range monitorID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("flagLimit.monitor_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(stationID) != 0 {
		if len(stationID) == 1 {
			whereStmts = append(whereStmts, "flagLimit.station_id = ?")
			values = append(values, stationID[0])
		} else {
			placeholder := make([]string, 0)
			for _, id := range stationID {
				placeholder = append(placeholder, "?")
				values = append(values, id)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("flagLimit.station_id IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	if len(flag) != 0 {
		if len(flag) == 1 {
			whereStmts = append(whereStmts, "flagLimit.flag = ?")
			values = append(values, flag[0])
		} else {
			placeholder := make([]string, 0)
			for _, f := range flag {
				placeholder = append(placeholder, "?")
				values = append(values, f)
			}
			whereStmts = append(whereStmts, fmt.Sprintf("flagLimit.flag IN (%s)", strings.Join(placeholder, ",")))
		}
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s flagLimit
	`, limitColumns, limitTableName(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get monitor flag limit: ", err)
		return nil, err
	}

	defer rows.Close()

	result := make([]*FlagLimit, 0)

	for rows.Next() {
		var s FlagLimit
		if err := s.scan(rows); err != nil {
			log.Println("error get monitor flag limit: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}
