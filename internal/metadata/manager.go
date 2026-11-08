package metadata

import (
	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

type MetadataMgr struct {
	tables  *TableMgr
	views   *ViewMgr
	stats   *StatMgr
	indexes *IndexMgr
}

func NewMetadataMgr(isNew bool, t *tx.Transaction) (*MetadataMgr, error) {
	tables, err := NewTableMgr(isNew, t)
	if err != nil {
		return nil, err
	}
	views, err := NewViewMgr(isNew, tables, t)
	if err != nil {
		return nil, err
	}
	stats, err := NewStatMgr(tables, t)
	if err != nil {
		return nil, err
	}
	indexes, err := NewIndexMgr(isNew, tables, stats, t)
	if err != nil {
		return nil, err
	}
	return &MetadataMgr{tables, views, stats, indexes}, nil
}

func (m *MetadataMgr) CreateTable(name string, s *record.Schema, t *tx.Transaction) error {
	return m.tables.CreateTable(name, s, t)
}
func (m *MetadataMgr) GetLayout(name string, t *tx.Transaction) (*record.Layout, error) {
	return m.tables.GetLayout(name, t)
}
func (m *MetadataMgr) CreateView(name, definition string, t *tx.Transaction) error {
	return m.views.CreateView(name, definition, t)
}
func (m *MetadataMgr) GetViewDef(name string, t *tx.Transaction) (string, error) {
	return m.views.GetViewDef(name, t)
}
func (m *MetadataMgr) GetStatInfo(name string, l *record.Layout, t *tx.Transaction) (StatInfo, error) {
	return m.stats.GetStatInfo(name, l, t)
}
func (m *MetadataMgr) CreateIndex(name, table, field string, t *tx.Transaction) error {
	return m.indexes.CreateIndex(name, table, field, t)
}
func (m *MetadataMgr) GetIndexInfo(table string, t *tx.Transaction) (map[string]IndexInfo, error) {
	return m.indexes.GetIndexInfo(table, t)
}
