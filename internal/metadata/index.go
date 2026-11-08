package metadata

import (
	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

// インデックス本体は第12章で実装する。ここでは定義と見積もりを扱う。
type IndexInfo struct {
	Name, TableName, FieldName string
	Layout                     *record.Layout
	stats                      StatInfo
	blockSize                  int
}

func newIndexInfo(name, table, field string, layout *record.Layout,
	si StatInfo, blockSize int) (IndexInfo, error) {
	s := &record.Schema{}
	if err := s.AddIntField("block"); err != nil {
		return IndexInfo{}, err
	}
	if err := s.AddIntField("id"); err != nil {
		return IndexInfo{}, err
	}
	for _, f := range layout.Schema().Fields() {
		if f.Name != field {
			continue
		}
		var err error
		if f.Type == record.Integer {
			err = s.AddIntField("dataval")
		} else {
			err = s.AddStringField("dataval", f.Length)
		}
		if err != nil {
			return IndexInfo{}, err
		}
	}
	il, err := record.NewLayout(s)
	if err != nil {
		return IndexInfo{}, err
	}
	return IndexInfo{name, table, field, il, si, blockSize}, nil
}

// 値が均等に分布していると仮定した、等価検索のヒット件数。
func (i IndexInfo) RecordsOutput() int {
	return i.stats.RecordsOutput() / i.stats.DistinctValues(i.FieldName)
}

func (i IndexInfo) DistinctValues(field string) int {
	if field == i.FieldName {
		return 1
	}
	return i.stats.DistinctValues(field)
}

// 第12章のHashIndex.searchCostに相当する概算。整数除算も本文に合わせる。
func (i IndexInfo) BlocksAccessed() int {
	rpb := i.blockSize / i.Layout.SlotSize()
	blocks := i.stats.RecordsOutput() / rpb
	return blocks / 100
}

type IndexMgr struct {
	layout *record.Layout
	tables *TableMgr
	stats  *StatMgr
}

func NewIndexMgr(isNew bool, tables *TableMgr, stats *StatMgr, t *tx.Transaction) (*IndexMgr, error) {
	if isNew {
		s, err := catalogSchema(
			record.Field{Name: "indexname", Type: record.Varchar, Length: MaxName},
			record.Field{Name: "tablename", Type: record.Varchar, Length: MaxName},
			record.Field{Name: "fieldname", Type: record.Varchar, Length: MaxName})
		if err != nil {
			return nil, err
		}
		if err := tables.CreateTable("idxcat", s, t); err != nil {
			return nil, err
		}
	}
	l, err := tables.GetLayout("idxcat", t)
	if err != nil {
		return nil, err
	}
	return &IndexMgr{l, tables, stats}, nil
}

func (m *IndexMgr) CreateIndex(name, table, field string, t *tx.Transaction) error {
	return withScan(t, "idxcat", m.layout, func(s *record.TableScan) error {
		if err := s.Insert(); err != nil {
			return err
		}
		if err := s.SetString("indexname", name); err != nil {
			return err
		}
		if err := s.SetString("tablename", table); err != nil {
			return err
		}
		return s.SetString("fieldname", field)
	})
}

func (m *IndexMgr) GetIndexInfo(table string, t *tx.Transaction) (map[string]IndexInfo, error) {
	l, err := m.tables.GetLayout(table, t)
	if err != nil {
		return nil, err
	}
	si, err := m.stats.GetStatInfo(table, l, t)
	if err != nil {
		return nil, err
	}
	result := make(map[string]IndexInfo)
	err = withScan(t, "idxcat", m.layout, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil || !ok {
				return err
			}
			tbl, err := s.GetString("tablename")
			if err != nil {
				return err
			}
			if tbl != table {
				continue
			}
			name, err := s.GetString("indexname")
			if err != nil {
				return err
			}
			field, err := s.GetString("fieldname")
			if err != nil {
				return err
			}
			info, err := newIndexInfo(name, table, field, l, si, t.BlockSize())
			if err != nil {
				return err
			}
			result[field] = info
		}
	})
	return result, err
}