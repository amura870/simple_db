package metadata

import (
	"fmt"

	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

const MaxViewDef = 100

type ViewMgr struct{ layout *record.Layout }

func NewViewMgr(isNew bool, tables *TableMgr, t *tx.Transaction) (*ViewMgr, error) {
	if isNew {
		// Viewのメタデータは名前と定義のクエリの２つ
		s, err := catalogSchema(
			record.Field{Name: "viewname", Type: record.Varchar, Length: MaxName},
			record.Field{Name: "viewdef", Type: record.Varchar, Length: MaxViewDef},
		)
		if err != nil {
			return nil, err
		}
		if err := tables.CreateTable("viewcat", s, t); err != nil {
			return nil, err
		}
	}
	l, err := tables.GetLayout("viewcat", t)
	if err != nil {
		return nil, err
	}
	return &ViewMgr{layout: l}, nil
}

func (m *ViewMgr) CreateView(name, definition string, t *tx.Transaction) error {
	return withScan(t, "viewcat", m.layout, func(s *record.TableScan) error {
		if err := s.Insert(); err != nil {
			return err
		}
		if err := s.SetString("viewname", name); err != nil {
			return err
		}
		return s.SetString("viewdef", definition)
	})
}

func (m *ViewMgr) GetViewDef(name string, t *tx.Transaction) (string, error) {
	var definition string
	err := withScan(t, "viewcat", m.layout, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("view not found: %s", name)
			}
			n, err := s.GetString("viewname")
			if err != nil {
				return err
			}
			if n == name {
				definition, err = s.GetString("viewdef")
				return err
			}
		}
	})
	return definition, err
}
