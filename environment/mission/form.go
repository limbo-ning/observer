package mission

import (
	"database/sql"
	"errors"
	"fmt"

	"obsessiontech/environment/authority"
)

const (
	FORM_TEXT  = "text"
	FORM_IMG   = "img"
	FORM_VIDEO = "video"
)

func init() {
	RegisterMission("form", func() IMissionComplete { return new(Form) })
}

type Form struct {
	BaseMissionComplete
	Fields []*formField `json:"fields,omitempty"`
}

type formField struct {
	Field string `json:"field"`
	Type  string `json:"type"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
}

func (p *Form) Validate(siteID string) error {
	for _, f := range p.Fields {
		switch f.Type {
		case FORM_IMG:
		case FORM_TEXT:
		case FORM_VIDEO:
		default:
			return errors.New("不支持的表单类型")
		}

		if f.Min > 0 && f.Max > 0 && f.Max < f.Min {
			return errors.New("数量限制不合理")
		}
	}

	return nil
}

func (p *Form) MissionComplete(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, missions map[int]*Mission, complete *Complete) error {

	var forms map[string]interface{}
	result, ok := complete.Result[p.ID]
	if !ok {
		forms = make(map[string]interface{})
	} else {
		forms, ok = result.(map[string]interface{})
		if !ok {
			return errors.New("invalid form")
		}
	}

	for _, f := range p.Fields {
		upload := forms[f.Field]
		var count int

		if upload != nil {
			switch f.Type {
			case FORM_TEXT:
				text, ok := upload.(string)
				if !ok {
					return errors.New("invalid form text")
				}
				count = len(text)
			default:
				list, ok := upload.([]interface{})
				if !ok {
					return errors.New("invalid form image list")
				}
				for _, l := range list {
					_, ok := l.(string)
					if !ok {
						return errors.New("invalid form image uri")
					}
				}
				count = len(list)
			}
		}

		switch {
		case f.Min > 0 && count < f.Min:
			return fmt.Errorf("%s需要大于%d", f.Field, f.Min)
		case f.Max > 0 && count > f.Max:
			return fmt.Errorf("%s需要少于%d", f.Field, f.Max)
		}
	}

	return nil
}

func (p *Form) MissionRevert(siteID string, txn *sql.Tx, actionAuth authority.ActionAuthSet, mission *Mission, complete *Complete) error {

	return nil
}
