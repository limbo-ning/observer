package excel_test

import (
	"obsessiontech/common/excel"
)

type TestUploader struct {
	SheetParser *Parser
}

func (u *TestUploader) GetSheetParsers() ([]excel.ISheetParser, error) {
	return []excel.ISheetParser{u.SheetParser}, nil
}

type Parser struct {
	Cells []*Cell
}

func (p *Parser) GetSheetDirection() excel.SheetDirection {
	return excel.SheetDirectionRow
}
func (p *Parser) GetSheetIndex() int {
	return 0
}
func (p *Parser) GetSheetRows() (int, int) {
	return -1, -1
}
func (p *Parser) UnMarshalEntry(raw map[string]any) (any, error) {
	return raw, nil
}
func (p *Parser) GetCellScanners() ([]excel.ICellScaner[excel.SheetRow], error) {
	cs := make([]excel.ICellScaner[excel.SheetRow], 0)
	for _, c := range p.Cells {
		cs = append(cs, c)
	}
	return cs, nil
}

type Cell struct {
	Index int
}

func (c *Cell) ScanCell(row *excel.SheetRow, entry *map[string]any) error {
	cell, err := row.Get(c.Index)
	if err != nil {
		return err
	}

	v, err := cell.GetString()
	if err != nil {
		return err
	}

	(*entry)["key"] = v
	return nil
}
