package excel

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/tealeg/xlsx/v3"
)

var E_skip_entry = errors.New("skipentry")
var E_skip_cell = errors.New("skipcell")

type SheetDirection int

const (
	SheetDirectionRow SheetDirection = iota
	SheetDirectionColumn
)

const (
	ValueTime   = "time"
	ValueString = "string"
	ValueInt    = "int"
	ValueFloat  = "float"
)

type SheetCell xlsx.Cell
type SheetRow xlsx.Row
type SheetColumn struct {
	index int
	sheet *xlsx.Sheet
}

func (s *SheetRow) Get(col int) (*SheetCell, error) {
	row := xlsx.Row(*s)
	cell := row.GetCell(col)

	sheetCell := new(SheetCell)
	*sheetCell = SheetCell(*cell)

	return sheetCell, nil
}

func (s *SheetRow) Len() int {
	return s.Sheet.Cols.Len
}

func (s *SheetColumn) Get(row int) (*SheetCell, error) {

	cell, err := s.sheet.Cell(row, s.index)
	if err != nil {
		return nil, err
	}
	sheetCell := new(SheetCell)
	*sheetCell = SheetCell(*cell)

	return sheetCell, nil
}

func (s *SheetColumn) Len() int {
	return s.sheet.MaxRow
}

func (c *SheetCell) GetTime() (*time.Time, error) {
	cell := xlsx.Cell(*c)
	t, err := cell.GetTime(cell.Row.Sheet.File.Date1904)
	if err != nil {
		return nil, err
	}

	Y, M, D := t.Date()
	h, m, s := t.Clock()

	localTime := time.Date(Y, M, D, h, m, s, 0, time.Local)

	return &localTime, nil
}
func (c *SheetCell) GetString() (string, error) {
	cell := xlsx.Cell(*c)
	return cell.FormattedValue()
}
func (c *SheetCell) GetFloat() (float64, error) {
	cell := xlsx.Cell(*c)
	return cell.Float()
}
func (c *SheetCell) GetInt() (int, error) {
	cell := xlsx.Cell(*c)
	return cell.Int()
}
func (c *SheetCell) IsEmpty() bool {
	cell := xlsx.Cell(*c)
	str, err := cell.FormattedValue()
	if err != nil {
		log.Println("error get formated value: ", err)
		return true
	}

	return strings.TrimSpace(str) == ""
}

type IExcelParser interface {
	GetSheetParsers() ([]ISheetParser, error)
}
type ISheetParser interface {
	GetSheetIndex() int
	GetSheetDirection() SheetDirection
	UnMarshalEntry(map[string]any) (any, error)
}
type ISheetColumnParser interface {
	GetSheetColumns() (int, int)
	GetSheetHeaderColumnIndex() []int
	GetColumnCellScanners(headers []*SheetColumn) ([]ICellScaner[SheetColumn], error)
}
type ISheetRowParser interface {
	GetSheetRows() (int, int)
	GetSheetHeaderRowIndex() []int
	GetRowCellScanners(headers []*SheetRow) ([]ICellScaner[SheetRow], error)
}
type ICellScaner[S SheetRow | SheetColumn] interface {
	ScanCell(*S, *map[string]any) error
}

func ParseExcel(excelData []byte, parser IExcelParser) ([]any, error) {

	f, err := xlsx.OpenBinary(excelData)
	if err != nil {
		log.Println("error open excel file: ", err)
		return nil, err
	}

	sheetParsers, err := parser.GetSheetParsers()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0)

	for _, sp := range sheetParsers {
		sheetIndex := sp.GetSheetIndex()
		if sheetIndex >= 0 {
			if len(f.Sheets) > sheetIndex {
				sheet := f.Sheets[sheetIndex]
				list, err := doSheet(sheet, sp)
				if err != nil {
					return nil, err
				}
				result = append(result, list...)
			} else {
				break
			}
		} else {
			for _, sheet := range f.Sheets {
				list, err := doSheet(sheet, sp)
				if err != nil {
					return nil, err
				}
				result = append(result, list...)
			}
		}
	}

	return result, nil
}

func doSheet(sheet *xlsx.Sheet, sp ISheetParser) ([]any, error) {

	log.Println("do sheet:", sheet.MaxRow, sheet.MaxCol)

	maxRow := sheet.MaxRow
	maxCol := sheet.MaxCol

	direction := sp.GetSheetDirection()

	result := make([]any, 0)

	switch direction {
	case SheetDirectionRow:

		if maxRow <= 0 {
			return result, nil
		}

		srp, ok := sp.(ISheetRowParser)
		if !ok {
			return nil, errors.New("invalid sheet row parser")
		}
		headerIndices := srp.GetSheetHeaderRowIndex()
		headers := make([]*SheetRow, 0)
		for _, i := range headerIndices {
			h, err := sheet.Row(i)
			if err != nil {
				return nil, err
			}
			sr := new(SheetRow)
			*sr = SheetRow(*h)
			headers = append(headers, sr)
		}
		cs, err := srp.GetRowCellScanners(headers)
		if err != nil {
			return nil, err
		}

		start, end := srp.GetSheetRows()

		rowIndex := start
		if rowIndex < 0 {
			rowIndex = 0
		}

		for {
			log.Println("scan row: ", rowIndex, end, maxRow)
			if end > 0 && rowIndex > end {
				break
			}
			if rowIndex > maxRow {
				log.Println("break: ", rowIndex > sheet.MaxRow)
				break
			}

			row, err := sheet.Row(rowIndex)
			if err != nil {
				return nil, err
			}

			sheetRow := new(SheetRow)
			*sheetRow = SheetRow(*row)

			entry := make(map[string]any)
			for _, c := range cs {
				err := c.ScanCell(sheetRow, &entry)
				if err != nil {
					if err == E_skip_cell {
						continue
					}
					if err == E_skip_entry {
						entry = nil
						break
					}
					return nil, err
				}
			}

			if len(entry) > 0 {
				result = append(result, entry)
			}

			rowIndex++
		}

	case SheetDirectionColumn:

		if maxCol <= 0 {
			return result, nil
		}

		scp, ok := sp.(ISheetColumnParser)
		if !ok {
			return nil, errors.New("invalid sheet column parser")
		}
		headerIndices := scp.GetSheetHeaderColumnIndex()
		headers := make([]*SheetColumn, 0)
		for _, i := range headerIndices {
			sr := &SheetColumn{
				index: i,
				sheet: sheet,
			}
			headers = append(headers, sr)
		}
		cs, err := scp.GetColumnCellScanners(headers)
		if err != nil {
			return nil, err
		}

		start, end := scp.GetSheetColumns()
		columnIndex := start
		if columnIndex < 0 {
			columnIndex = 0
		}

		for {
			log.Println("scan column: ", columnIndex)
			if end > 0 && columnIndex > end {
				break
			}
			if columnIndex > maxCol {
				break
			}
			column := sheet.Col(columnIndex)
			if column == nil {
				break
			}

			sheetColumn := &SheetColumn{
				index: columnIndex,
				sheet: sheet,
			}

			entry := make(map[string]any)
			for _, c := range cs {
				err := c.ScanCell(sheetColumn, &entry)
				if err != nil {
					if err == E_skip_cell {
						continue
					}
					if err == E_skip_entry {
						entry = nil
						break
					}
					return nil, err
				}
			}

			if len(entry) > 0 {
				result = append(result, entry)
			}

			columnIndex++
		}
	default:
		return nil, errors.New("invalid sheet direction")
	}

	return result, nil
}
