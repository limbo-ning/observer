package initialization

import (
	"database/sql"
	"fmt"

	"obsessiontech/common/datasource"
)

var initializers = make(map[string][]string)

func Register(moduleID string, tables []string) {
	if _, exists := initializers[moduleID]; exists {
		panic("duplicate moduleID in initializer:" + moduleID)
	}

	initializers[moduleID] = tables
}

func Initialize(siteID string, txn *sql.Tx, moduleID string) error {
	tables, exists := initializers[moduleID]
	if !exists {
		return nil
	}

	for _, table := range tables {
		if err := CreateTable(siteID, txn, siteID+"_"+table, table); err != nil {
			return err
		}
	}

	return nil
}

func CreateTable(siteID string, txn *sql.Tx, tableName, sourceTableName string) error {

	var op = func(thisTxn *sql.Tx) error {
		if exists, err := ExistsTable(siteID, thisTxn, tableName); err != nil {
			return err
		} else if !exists {
			if _, err := thisTxn.Exec(fmt.Sprintf(`CREATE TABLE %s_%s like %s_%s`, siteID, tableName, "prototype", sourceTableName)); err != nil {
				return err
			}
		}

		return nil
	}

	if txn == nil {
		return datasource.Txn(func(t *sql.Tx) {
			if err := op(t); err != nil {
				panic(err)
			}
		})
	} else {
		return op(txn)
	}

}

func ExistsTable(siteID string, txn *sql.Tx, tableName string) (bool, error) {

	var rows *sql.Rows
	var err error

	if txn == nil {
		rows, err = datasource.GetConn().Query(fmt.Sprintf(`SHOW TABLES LIKE '%s_%s'`, siteID, tableName))
	} else {
		rows, err = txn.Query(fmt.Sprintf(`SHOW TABLES LIKE '%s_%s'`, siteID, tableName))
	}

	if err != nil {
		return false, err
	}

	defer rows.Close()

	if rows.Next() {
		return true, nil
	}
	return false, nil
}
