package query

import "errors"

type ProductScan struct {
	left, right Scan
	leftOK      bool
}

func NewProductScan(left, right Scan) (*ProductScan, error) {
	s := &ProductScan{left: left, right: right}
	if err := s.BeforeFirst(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ProductScan) BeforeFirst() error {
	s.leftOK = false
	if err := s.left.BeforeFirst(); err != nil {
		return err
	}
	if err := s.right.BeforeFirst(); err != nil {
		return err
	}
	ok, err := s.left.Next()
	if err != nil {
		return err
	}
	s.leftOK = ok
	return nil
}

func (s *ProductScan) Next() (bool, error) {
	if !s.leftOK {
		return false, nil
	}
	ok, err := s.right.Next()
	if err != nil || ok {
		return ok, err
	}

	// 右が終端に達したら、左を1行進めて右を巻き戻す。
	leftOK, err := s.left.Next()
	if err != nil {
		return false, err
	}
	s.leftOK = leftOK
	if !leftOK {
		return false, nil
	}
	if err := s.right.BeforeFirst(); err != nil {
		return false, err
	}
	ok, err = s.right.Next()
	if err != nil {
		return false, err
	}
	if !ok {
		// 右が空なら、直積全体も空。
		s.leftOK = false
	}
	return ok, nil
}

func (s *ProductScan) GetInt(field string) (int32, error) {
	if s.left.HasField(field) {
		return s.left.GetInt(field)
	}
	return s.right.GetInt(field)
}

func (s *ProductScan) GetString(field string) (string, error) {
	if s.left.HasField(field) {
		return s.left.GetString(field)
	}
	return s.right.GetString(field)
}

func (s *ProductScan) GetVal(field string) (Constant, error) {
	if s.left.HasField(field) {
		return s.left.GetVal(field)
	}
	return s.right.GetVal(field)
}

func (s *ProductScan) HasField(field string) bool {
	return s.left.HasField(field) || s.right.HasField(field)
}

func (s *ProductScan) Close() error {
	s.leftOK = false
	return errors.Join(s.left.Close(), s.right.Close())
}

var _ Scan = (*ProductScan)(nil)
