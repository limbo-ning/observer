package clientAgent

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/common/encrypt"
	"obsessiontech/common/excel"
	"obsessiontech/environment/authority"
)

const type_encrypt = "enc_a"

type Setting struct {
	ID          int    `json:"ID"`
	ClientAgent string `json:"clientAgent"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func settingTableName(siteID string) string {
	return siteID + "_clientagent_setting"
}

const settingColumn = "id, clientagent, description, setting_key, type, value"

func (s *Setting) scan(rows *sql.Rows) error {
	if err := rows.Scan(&s.ID, &s.ClientAgent, &s.Description, &s.Key, &s.Type, &s.Value); err != nil {
		return err
	}
	return nil
}

func GetSettings(siteID string, actionAuth authority.ActionAuthSet, clientAgent, key, settingType, q string) ([]*Setting, error) {
	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	whereStmts = append(whereStmts, "clientagent = ?")
	values = append(values, clientAgent)

	if key != "" {
		whereStmts = append(whereStmts, "setting_key = ?")
		values = append(values, key)
	}
	if settingType != "" {
		whereStmts = append(whereStmts, "type = ?")
		values = append(values, settingType)
	}
	if q != "" {
		qq := "%" + q + "%"
		whereStmts = append(whereStmts, "setting_key LIKE ?")
		values = append(values, qq)
	}
	result := make([]*Setting, 0)

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s
	`, settingColumn, settingTableName(siteID))

	if len(whereStmts) != 0 {
		SQL += "\nWHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s Setting
		if err := s.scan(rows); err != nil {
			return nil, err
		}

		if s.Type == type_encrypt {
			if len(actionAuth) == 0 {
				return nil, errors.New("无权限查看加密内容")
			}

			token := actionAuth[0].Session

			s.Value = encrypt.Base64Encrypt(token + s.Value + token)
		}

		result = append(result, &s)
	}

	return result, nil
}

func (s *Setting) Validate(siteID string) error {
	module, err := GetModule(siteID)
	if err != nil {
		return err
	}

	keys, exists := module.SettingTypeKeys[s.Type]
	if !exists {
		return errors.New("不正确的类型")
	}

	for _, k := range keys {
		if s.Key == k {
			return nil
		}
	}

	return errors.New("不正确的键值")
}

func (s *Setting) Add(siteID string) error {

	if err := s.Validate(siteID); err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		INSERT INTO %s
			(clientagent, setting_key, type, value, description)
		VALUES
			(?,?,?,?,?)
	`, settingTableName(siteID)), s.ClientAgent, s.Key, s.Type, s.Value, s.Description); err != nil {
		return err
	}
	return nil
}
func (s *Setting) Update(siteID string) error {
	if err := s.Validate(siteID); err != nil {
		return err
	}

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		UPDATE
			%s
		SET
			clientagent=?, setting_key=?, type=?, value=?, description=?
		WHERE
			id = ?
	`, settingTableName(siteID)), s.ClientAgent, s.Key, s.Type, s.Value, s.Description, s.ID); err != nil {
		return err
	}
	return nil
}
func (s *Setting) Delete(siteID string) error {
	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id = ?
	`, settingTableName(siteID)), s.ID); err != nil {
		return err
	}
	return nil
}

func ClearSetting(siteID, clientAgent, key, settingType string) error {

	whereStmts := make([]string, 0)
	values := make([]any, 0)

	if clientAgent != "" {
		whereStmts = append(whereStmts, "clientagent = ?")
		values = append(values, clientAgent)
	}
	if key != "" {
		whereStmts = append(whereStmts, "setting_key = ?")
		values = append(values, key)
	}
	if settingType != "" {
		whereStmts = append(whereStmts, "type = ?")
		values = append(values, settingType)
	}

	if len(whereStmts) == 0 {
		return errors.New("需要参数")
	}

	SQL := fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			%s
	`, settingTableName(siteID), strings.Join(whereStmts, " AND "))

	if _, err := datasource.GetConn().Exec(SQL, values...); err != nil {
		log.Println("error clear setting: ", SQL, values, err)
		return err
	}

	return nil
}

func UploadExcel(siteID string, files map[string][]*multipart.FileHeader, params map[string][]string) ([]any, error) {
	uploaderStrs, exists := params["uploader"]
	if !exists {
		return nil, errors.New("需要uploader")
	}
	uploaderStr := uploaderStrs[0]

	uploader := new(excel.Uploader)
	if err := json.Unmarshal([]byte(uploaderStr), &uploader); err != nil {
		return nil, err
	}

	excelFiles, exists := files["excel"]
	if !exists {
		return nil, errors.New("需要上传excel文件")
	}
	excelFile := excelFiles[0]

	f, err := excelFile.Open()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return excel.ParseExcel(data, uploader)
}
