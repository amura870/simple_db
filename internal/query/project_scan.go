package query

import "fmt"

type ProjectScan struct {
	s      Scan
	fields []string
}

func NewProjectScan(s Scan, fields []string) *ProjectScan {
	return &ProjectScan{s: s, fields: append([]string(nil), fields...)}
}

func (s *ProjectScan) BeforeFirst() error  { return s.s.BeforeFirst() }
func (s *ProjectScan) Next() (bool, error) { return s.s.Next() }
func (s *ProjectScan) Close() error        { return s.s.Close() }

func (s *ProjectScan) HasField(field string) bool {
	for _, name := range s.fields {
		if name == field {
			return true
		}
	}
	return false
}

func (s *ProjectScan) GetInt(field string) (int32, error) {
	if !s.HasField(field) {
		return 0, fmt.Errorf("field not projected: %s", field)
	}
	return s.s.GetInt(field)
}

func (s *ProjectScan) GetString(field string) (string, error) {
	if !s.HasField(field) {
		return "", fmt.Errorf("field not projected: %s", field)
	}
	return s.s.GetString(field)
}

func (s *ProjectScan) GetVal(field string) (Constant, error) {
	if !s.HasField(field) {
		return Constant{}, fmt.Errorf("field not projected: %s", field)
	}
	return s.s.GetVal(field)
}

var _ Scan = (*ProjectScan)(nil)
