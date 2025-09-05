package upload

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"obsessiontech/common/datasource"
)

type DataUploader struct {
	ID       int            `json:"ID"`
	Name     string         `json:"name"`
	Uploader *ExcelUploader `json:"uploader"`
}

const dataUploaderColumn = "datauploader.id, datauploader.name, datauploader.uploader"

func dataUploaderTemplateTable(siteID string) string {
	return siteID + "_datauploader"
}

func (u *DataUploader) scan(rows *sql.Rows) error {
	var uploader string
	if err := rows.Scan(&u.ID, &u.Name, &uploader); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(uploader), &u.Uploader); err != nil {
		return err
	}
	return nil
}

func (u *DataUploader) Add(siteID string) error {

	if u.Uploader == nil {
		return errors.New("请提交uploader")
	}

	uploader, err := json.Marshal(u.Uploader)
	if err != nil {
		return err
	}

	if ret, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(name, uploader)
		VALUES
			(?,?)
	`, dataUploaderTemplateTable(siteID)), u.Name, string(uploader)); err != nil {
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		return err
	} else {
		u.ID = int(id)
	}

	return nil
}

func (u *DataUploader) Update(siteID string) error {

	if u.Uploader == nil {
		return errors.New("请提交uploader")
	}

	uploader, err := json.Marshal(u.Uploader)
	if err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			name=?,uploader=?
		WHERE
			id=?
	`, dataUploaderTemplateTable(siteID)), u.Name, string(uploader), u.ID); err != nil {
		return err
	}

	return nil
}

func (u *DataUploader) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, dataUploaderTemplateTable(siteID)), u.ID); err != nil {
		return err
	}

	return nil
}

func GetDataUploader(siteID, q string) ([]*DataUploader, error) {
	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if q != "" {
		whereStmts = append(whereStmts, "name like ?")
		values = append(values, "%"+q+"%")
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s datauploader
	`, dataUploaderColumn, dataUploaderTemplateTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]*DataUploader, 0)

	for rows.Next() {
		var d DataUploader
		if err := d.scan(rows); err != nil {
			return nil, err
		}
		result = append(result, &d)
	}

	return result, nil
}
