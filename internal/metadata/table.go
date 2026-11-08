package metadata

import (
	"fmt"

	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

type TableMgr struct {
	tcat, fcat *record.Layout
}

func NewTableMgr(isNew bool, t *tx.Transaction) (*TableMgr, error) {
	ts, err := catalogSchema(
		record.Field{Name: "tblname", Type: record.Varchar, Length: MaxName},
		record.Field{Name: "slotsize", Type: record.Integer},
	)
	if err != nil {
		return nil, err
	}
	fs, err := catalogSchema(
		record.Field{Name: "tblname", Type: record.Varchar, Length: MaxName},
		record.Field{Name: "fldname", Type: record.Varchar, Length: MaxName},
		record.Field{Name: "type", Type: record.Integer},
		record.Field{Name: "length", Type: record.Integer},
		record.Field{Name: "offset", Type: record.Integer},
	)
	if err != nil {
		return nil, err
	}
	// TableCatalog
	tl, err := record.NewLayout(ts)
	if err != nil {
		return nil, err
	}
	// FieldCatalog
	fl, err := record.NewLayout(fs)
	if err != nil {
		return nil, err
	}
	m := &TableMgr{tcat: tl, fcat: fl}
	if isNew {
		if err := m.CreateTable("tblcat", ts, t); err != nil {
			return nil, err
		}
		if err := m.CreateTable("fldcat", fs, t); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// 入力のスキーマをもとにテーブルを作成する
func (m *TableMgr) CreateTable(name string, schema *record.Schema, t *tx.Transaction) error {
	layout, err := record.NewLayout(schema)
	if err != nil {
		return err
	}
	// TableCatalogのデータを"tblcat"テーブルに挿入する
	err = withScan(t, "tblcat", m.tcat, func(s *record.TableScan) error {
		if err := s.Insert(); err != nil {
			return err
		}
		if err := s.SetString("tblname", name); err != nil {
			return err
		}
		return s.SetInt("slotsize", int32(layout.SlotSize()))
	})
	if err != nil {
		return err
	}
	// FieldCatalogのデータを"fldcat"テーブルに挿入する
	return withScan(t, "fldcat", m.fcat, func(s *record.TableScan) error {
		for _, f := range schema.Fields() {
			offset, err := layout.Offset(f.Name)
			if err != nil {
				return err
			}
			if err := s.Insert(); err != nil {
				return err
			}
			if err := s.SetString("tblname", name); err != nil {
				return err
			}
			if err := s.SetString("fldname", f.Name); err != nil {
				return err
			}
			if err := s.SetInt("type", int32(f.Type)); err != nil {
				return err
			}
			if err := s.SetInt("length", int32(f.Length)); err != nil {
				return err
			}
			if err := s.SetInt("offset", int32(offset)); err != nil {
				return err
			}
		}
		return nil
	})
}

// テーブルの名前からレイアウトを出力する
func (m *TableMgr) GetLayout(name string, t *tx.Transaction) (*record.Layout, error) {
	slotSize := -1
	err := withScan(t, "tblcat", m.tcat, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil || !ok {
				return err
			}
			n, err := s.GetString("tblname")
			if err != nil {
				return err
			}
			if n != name {
				continue
			}
			v, err := s.GetInt("slotsize")
			slotSize = int(v)
			return err
		}
	})
	if err != nil {
		return nil, err
	}
	if slotSize < 0 {
		return nil, fmt.Errorf("table not found: %s", name)
	}
	schema := &record.Schema{}
	offsets := make(map[string]int)
	err = withScan(t, "fldcat", m.fcat, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil || !ok {
				return err
			}
			n, err := s.GetString("tblname")
			if err != nil {
				return err
			}
			if n != name {
				continue
			}
			field, err := s.GetString("fldname")
			if err != nil {
				return err
			}
			typ, err := s.GetInt("type")
			if err != nil {
				return err
			}
			length, err := s.GetInt("length")
			if err != nil {
				return err
			}
			offset, err := s.GetInt("offset")
			if err != nil {
				return err
			}
			switch record.FieldType(typ) {
			case record.Integer:
				err = schema.AddIntField(field)
			case record.Varchar:
				err = schema.AddStringField(field, int(length))
			default:
				return fmt.Errorf("unknown field type: %d", typ)
			}
			if err != nil {
				return err
			}
			offsets[field] = int(offset)
		}
	})
	if err != nil {
		return nil, err
	}
	return record.NewStoredLayout(schema, offsets, slotSize), nil
}
