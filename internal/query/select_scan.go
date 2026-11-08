package query

import (
	"fmt"

	"example.com/simpledb/internal/record"
)

type SelectScan struct {
	s    Scan
	pred *Predicate
}

func NewSelectScan(s Scan, pred *Predicate) *SelectScan {
	return &SelectScan{s: s, pred: pred}
}

func (s *SelectScan) BeforeFirst() error         { return s.s.BeforeFirst() }
func (s *SelectScan) Close() error               { return s.s.Close() }
func (s *SelectScan) HasField(field string) bool { return s.s.HasField(field) }

func (s *SelectScan) Next() (bool, error) {
	for {
		ok, err := s.s.Next()
		if err != nil || !ok {
			return false, err
		}
		match, err := s.pred.IsSatisfied(s.s)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
}

func (s *SelectScan) GetInt(field string) (int32, error)     { return s.s.GetInt(field) }
func (s *SelectScan) GetString(field string) (string, error) { return s.s.GetString(field) }
func (s *SelectScan) GetVal(field string) (Constant, error)  { return s.s.GetVal(field) }

func (s *SelectScan) updateScan() (UpdateScan, error) {
	u, ok := s.s.(UpdateScan)
	if !ok {
		return nil, fmt.Errorf("underlying scan is not updatable")
	}
	return u, nil
}

func (s *SelectScan) SetInt(field string, value int32) error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.SetInt(field, value)
}

func (s *SelectScan) SetString(field, value string) error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.SetString(field, value)
}

func (s *SelectScan) SetVal(field string, value Constant) error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.SetVal(field, value)
}

func (s *SelectScan) Insert() error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.Insert()
}

func (s *SelectScan) Delete() error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.Delete()
}

func (s *SelectScan) GetRID() (record.RID, error) {
	u, err := s.updateScan()
	if err != nil {
		return record.RID{}, err
	}
	return u.GetRID()
}

func (s *SelectScan) MoveToRID(rid record.RID) error {
	u, err := s.updateScan()
	if err != nil {
		return err
	}
	return u.MoveToRID(rid)
}

var _ UpdateScan = (*SelectScan)(nil)
