package metadata

import (
	"sync"

	"example.com/simpledb/internal/record"
	"example.com/simpledb/internal/tx"
)

type StatInfo struct{ blocks, records int }

func (s StatInfo) BlocksAccessed() int         { return s.blocks }
func (s StatInfo) RecordsOutput() int          { return s.records }
func (s StatInfo) DistinctValues(_ string) int { return 1 + s.records/3 }

type StatMgr struct {
	mu     sync.Mutex
	tables *TableMgr
	stats  map[string]StatInfo
	calls  int
}

func NewStatMgr(tables *TableMgr, t *tx.Transaction) (*StatMgr, error) {
	m := &StatMgr{tables: tables}
	if err := m.refresh(t); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *StatMgr) GetStatInfo(name string, layout *record.Layout, t *tx.Transaction) (StatInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.calls > 100 {
		if err := m.refresh(t); err != nil {
			return StatInfo{}, err
		}
	}
	if si, ok := m.stats[name]; ok {
		return si, nil
	}
	si, err := calcStats(name, layout, t)
	if err != nil {
		return StatInfo{}, err
	}
	m.stats[name] = si
	return si, nil
}

// 呼び出し元がmuを保持する。コンストラクタからの呼び出し時だけは不要。
func (m *StatMgr) refresh(t *tx.Transaction) error {
	next := make(map[string]StatInfo)
	err := withScan(t, "tblcat", m.tables.tcat, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil || !ok {
				return err
			}
			name, err := s.GetString("tblname")
			if err != nil {
				return err
			}
			layout, err := m.tables.GetLayout(name, t)
			if err != nil {
				return err
			}
			si, err := calcStats(name, layout, t)
			if err != nil {
				return err
			}
			next[name] = si
		}
	})
	if err != nil {
		return err
	}
	m.stats, m.calls = next, 0
	return nil
}

func calcStats(name string, layout *record.Layout, t *tx.Transaction) (StatInfo, error) {
	si := StatInfo{}
	err := withScan(t, name, layout, func(s *record.TableScan) error {
		for {
			ok, err := s.Next()
			if err != nil || !ok {
				return err
			}
			si.records++
			rid, err := s.GetRID()
			if err != nil {
				return err
			}
			si.blocks = rid.BlockNumber + 1
		}
	})
	if err != nil {
		return StatInfo{}, err
	}
	return si, nil
}
