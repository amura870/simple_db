package query

import (
	"fmt"

	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

// record packageのTableScanにGet/SetValメソッドを追加したもの
type TableScan struct {
	*record.TableScan
	schema *record.Schema
}

func NewTableScan(t *tx.Transaction, name string, layout *record.Layout) (*TableScan, error) {
	s, err := record.NewTableScan(t, name, layout)
	if err != nil {
		return nil, err
	}
	return &TableScan{TableScan: s, schema: layout.Schema()}, nil
}

// 入力のフィールド名の値を取得する
func (s *TableScan) GetVal(name string) (Constant, error) {
	f, ok := s.schema.Field(name)
	if !ok {
		return Constant{}, fmt.Errorf("unknown field: %s", name)
	}
	switch f.Type {
	case record.Integer:
		v, err := s.GetInt(name)
		return NewIntConstant(v), err
	case record.Varchar:
		v, err := s.GetString(name)
		return NewStringConstant(v), err
	default:
		return Constant{}, fmt.Errorf("unknown field type: %d", f.Type)
	}
}

func (s *TableScan) SetVal(name string, value Constant) error {
	f, ok := s.schema.Field(name)
	if !ok {
		return fmt.Errorf("unknown field: %s", name)
	}
	switch f.Type {
	case record.Integer:
		v, err := value.AsInt()
		if err != nil {
			return err
		}
		return s.SetInt(name, v)
	case record.Varchar:
		v, err := value.AsString()
		if err != nil {
			return err
		}
		return s.SetString(name, v)
	default:
		return fmt.Errorf("unknown field type: %d", f.Type)
	}
}

var _ UpdateScan = (*TableScan)(nil)