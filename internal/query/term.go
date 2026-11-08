package query

import "example.com/simpledb/internal/record"

// 第10章のPlanに必要となるメソッドだけを先に定義する。
// DistinctValuesは1以上を返すものとする。
type DistinctEstimator interface {
	DistinctValues(field string) int
}

// JavaのInteger.MAX_VALUEに対応する、見積もり用の上限。
const maxReduction = 1<<31 - 1

type Term struct {
	lhs, rhs Expression
}

func NewTerm(lhs, rhs Expression) Term {
	return Term{lhs: lhs, rhs: rhs}
}

func (t Term) IsSatisfied(s Scan) (bool, error) {
	left, err := t.lhs.Evaluate(s)
	if err != nil {
		return false, err
	}
	right, err := t.rhs.Evaluate(s)
	if err != nil {
		return false, err
	}
	return left.Equals(right), nil
}

func (t Term) AppliesTo(schema *record.Schema) bool {
	return t.lhs.AppliesTo(schema) && t.rhs.AppliesTo(schema)
}

func (t Term) ReductionFactor(p DistinctEstimator) int {
	if t.lhs.isField && t.rhs.isField {
		return max(p.DistinctValues(t.lhs.field), p.DistinctValues(t.rhs.field))
	}
	if t.lhs.isField {
		return p.DistinctValues(t.lhs.field)
	}
	if t.rhs.isField {
		return p.DistinctValues(t.rhs.field)
	}
	if t.lhs.value.Equals(t.rhs.value) {
		return 1
	}
	return maxReduction
}

func (t Term) EquatesWithConstant(field string) (Constant, bool) {
	if t.lhs.isField && t.lhs.field == field && !t.rhs.isField {
		return t.rhs.value, true
	}
	if t.rhs.isField && t.rhs.field == field && !t.lhs.isField {
		return t.lhs.value, true
	}
	return Constant{}, false
}

func (t Term) EquatesWithField(field string) (string, bool) {
	if t.lhs.isField && t.rhs.isField {
		if t.lhs.field == field {
			return t.rhs.field, true
		}
		if t.rhs.field == field {
			return t.lhs.field, true
		}
	}
	return "", false
}

func (t Term) String() string {
	return t.lhs.String() + "=" + t.rhs.String()
}
