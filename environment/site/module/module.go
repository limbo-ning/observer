package module

import (
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/category"
	"obsessiontech/environment/site"
)

func GetModuleList(cids []int, q string, pageNo, pageSize int, moduleID ...string) ([]*site.Module, int, error) {

	modules := make([]*site.Module, 0)
	var total int

	sql := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s as module
	`, site.ModuleColumns, site.ModuleTableName)

	countsql := fmt.Sprintf(`
		SELECT
			COUNT(1)
		FROM
			%s as module
	`, site.ModuleTableName)

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if cids != nil && len(cids) > 0 {
		for _, cid := range cids {
			joinSQL, joinWhere, joinValues, err := category.JoinCategoryMapping("c", "module", "", cid)
			if err != nil {
				return nil, 0, err
			}
			if joinSQL == "" {
				return modules, total, nil
			}
			sql += joinSQL
			countsql += joinSQL

			whereStmts = append(whereStmts, joinWhere...)
			values = append(values, joinValues...)
		}
	}

	if len(moduleID) > 0 {
		placeHolder := make([]string, 0)
		for _, id := range moduleID {
			if id != "" {
				placeHolder = append(placeHolder, "?")
				values = append(values, id)
			}
		}
		if len(placeHolder) > 0 {
			whereStmts = append(whereStmts, fmt.Sprintf("module.id IN (%s)", strings.Join(placeHolder, ",")))
			pageNo = 1
			pageSize = len(moduleID)
		}
	}
	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "(module.id LIKE ? OR module.name LIKE ? OR module.description LIKE ?)")
		values = append(values, qq, qq, qq)
	}

	if len(whereStmts) > 0 {
		sql += "\nWHERE " + strings.Join(whereStmts, " AND ")
		countsql += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	if err := datasource.GetConn().QueryRow(countsql, values...).Scan(&total); err != nil {
		log.Println("error count modules: ", err)
		return nil, 0, err
	}

	if pageSize <= 0 {
		pageSize = 20
	}
	if pageNo <= 0 {
		pageNo = 0
	} else {
		pageNo = (pageNo - 1) * pageSize
	}
	values = append(values, pageNo, pageSize)

	sql += "\nLIMIT ?,?"

	rows, err := datasource.GetConn().Query(sql, values...)
	if err != nil {
		log.Println("error get modules: ", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var m site.Module
		if err := m.Scan(rows); err != nil {
			return nil, 0, err
		}

		modules = append(modules, &m)
	}

	return modules, total, nil
}

func AddModuleCategory(moduleID string, cid int) error {
	return category.AddCategoryMapping("c", "module", moduleID, cid)
}

func DeleteModuleCategory(moduleID string, cid int) error {
	return category.DeleteCategoryMapping("c", "module", moduleID, cid)
}
