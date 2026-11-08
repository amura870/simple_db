package metadata

import (
	"errors"

	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

const MaxName = 16

// TableScanの度に実施する処理をまとめる
func withScan(t *tx.Transaction, name string, layout *record.Layout, fn func(*record.TableScan) error) (err error) {
	s, err := record.NewTableScan(t, name, layout)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	return fn(s)
}

// フィールドをスキーマ型に変換する
func catalogSchema(fields ...record.Field) (*record.Schema, error) {
	s := &record.Schema{}
	for _, f := range fields {
		var err error
		if f.Type == record.Integer {
			err = s.AddIntField(f.Name)
		} else {
			err = s.AddStringField(f.Name, f.Length)
		}
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}
