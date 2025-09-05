package excel

import (
	"errors"
)

type Uploader struct {
	Name   string         `json:"name"`
	Sheets []*SheetConfig `json:"sheets"`
}

type SheetConfig struct {
	SheetIndices []int          `json:"sheetIndices"`
	Direction    SheetDirection `json:"direction"`
	Content      [2]int         `json:"content"`
	// Headers      []int          `json:"headers"`
	Entries []*EntryConfig `json:"entries"`
}

type EntryConfig struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Field string `json:"field"`
}

func (u *Uploader) GetSheetParsers() ([]ISheetParser, error) {

	result := make([]ISheetParser, 0)

	for _, s := range u.Sheets {
		parsers, err := s.getSheetParsers()
		if err != nil {
			return nil, err
		}

		result = append(result, parsers...)
	}

	return result, nil
}

func (s *SheetConfig) getSheetParsers() ([]ISheetParser, error) {
	result := make([]ISheetParser, 0)

	if len(s.SheetIndices) == 0 {
		p := new(SheetParser)
		p.SheetIndex = -1
		p.SheetConfig = s

		result = append(result, p)
	} else {
		for _, i := range s.SheetIndices {
			p := new(SheetParser)
			p.SheetIndex = i
			p.SheetConfig = s

			result = append(result, p)
		}
	}

	return result, nil
}

type SheetParser struct {
	*SheetConfig
	SheetIndex int
}

func (p *SheetParser) GetSheetDirection() SheetDirection {
	return p.Direction
}
func (p *SheetParser) GetSheetIndex() int {
	return p.SheetIndex
}
func (p *SheetParser) UnMarshalEntry(raw map[string]any) (any, error) {
	return raw, nil
}

func (p *SheetParser) GetSheetColumns() (int, int) {
	return p.Content[0], p.Content[1]
}
func (p *SheetParser) GetSheetHeaderColumnIndex() []int {
	return nil
	// return p.Headers
}
func (p *SheetParser) GetColumnCellScanners(headers []*SheetColumn) ([]ICellScaner[SheetColumn], error) {
	result := make([]ICellScaner[SheetColumn], 0)
	for _, e := range p.Entries {
		s := new(columnScanner)
		s.EntryConfig = e
		result = append(result, s)
	}
	return result, nil
}

func (p *SheetParser) GetSheetRows() (int, int) {
	return p.Content[0], p.Content[1]
}
func (p *SheetParser) GetSheetHeaderRowIndex() []int {
	return nil
	// return p.Headers
}
func (p *SheetParser) GetRowCellScanners(headers []*SheetRow) ([]ICellScaner[SheetRow], error) {
	result := make([]ICellScaner[SheetRow], 0)
	for _, e := range p.Entries {
		s := new(rowScanner)
		s.EntryConfig = e
		result = append(result, s)
	}
	return result, nil
}

type rowScanner struct {
	*EntryConfig
}
type columnScanner struct {
	*EntryConfig
}

func (s *EntryConfig) scan(cell *SheetCell, entry *map[string]any) error {
	var value any

	switch s.Type {
	case ValueFloat:
		flt, err := cell.GetFloat()
		if err != nil {
			return err
		}
		value = flt
	case ValueInt:
		i, err := cell.GetInt()
		if err != nil {
			return err
		}
		value = i
	case ValueString:
		s, err := cell.GetString()
		if err != nil {
			return err
		}
		value = s
	case ValueTime:
		t, err := cell.GetTime()
		if err != nil {
			return err
		}
		value = t
	default:
		return errors.New("不支持的类型")
	}

	(*entry)[s.Field] = value

	return nil
}
func (s *columnScanner) ScanCell(column *SheetColumn, entry *map[string]any) error {

	cell, err := column.Get(s.Index)
	if err != nil {
		return err
	}

	if cell.IsEmpty() {
		return nil
	}

	return s.scan(cell, entry)
}
func (s *rowScanner) ScanCell(row *SheetRow, entry *map[string]any) error {

	cell, err := row.Get(s.Index)
	if err != nil {
		return err
	}

	if cell.IsEmpty() {
		return nil
	}

	return s.scan(cell, entry)
}
