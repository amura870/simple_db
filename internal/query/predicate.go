package query

import (
	"strings"

	"example.com/simpledb/internal/record"
)

type Predicate struct {
	terms []Term
}

// 引数なしなら常にtrue。複数のTermはANDで結合する。
func NewPredicate(terms ...Term) *Predicate {
	return &Predicate{terms: append([]Term(nil), terms...)}
}

func (p *Predicate) ConjoinWith(other *Predicate) {
	p.terms = append(p.terms, other.terms...)
}

func (p *Predicate) IsSatisfied(s Scan) (bool, error) {
	for _, term := range p.terms {
		ok, err := term.IsSatisfied(s)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

func (p *Predicate) ReductionFactor(plan DistinctEstimator) int {
	factor := 1
	for _, term := range p.terms {
		n := term.ReductionFactor(plan)
		// 乗算でオーバーフローしないように上限で止める。
		if n >= maxReduction || factor > maxReduction/n {
			return maxReduction
		}
		factor *= n
	}
	return factor
}

// このスキーマだけで評価できる項を取り出す。
// 該当する項がなければnilを返す（常にtrueの空Predicateとは区別する）。
func (p *Predicate) SelectSubPred(schema *record.Schema) *Predicate {
	result := NewPredicate()
	for _, term := range p.terms {
		if term.AppliesTo(schema) {
			result.terms = append(result.terms, term)
		}
	}
	if len(result.terms) == 0 {
		return nil
	}
	return result
}

// 片方だけでは評価できず、両方を合わせれば評価できる項を取り出す。
func (p *Predicate) JoinSubPred(left, right *record.Schema) (*Predicate, error) {
	combined := &record.Schema{}
	if err := combined.AddAll(left); err != nil {
		return nil, err
	}
	if err := combined.AddAll(right); err != nil {
		return nil, err
	}
	result := NewPredicate()
	for _, term := range p.terms {
		if !term.AppliesTo(left) && !term.AppliesTo(right) && term.AppliesTo(combined) {
			result.terms = append(result.terms, term)
		}
	}
	if len(result.terms) == 0 {
		return nil, nil
	}
	return result, nil
}

func (p *Predicate) EquatesWithConstant(field string) (Constant, bool) {
	for _, term := range p.terms {
		if value, ok := term.EquatesWithConstant(field); ok {
			return value, true
		}
	}
	return Constant{}, false
}

func (p *Predicate) EquatesWithField(field string) (string, bool) {
	for _, term := range p.terms {
		if name, ok := term.EquatesWithField(field); ok {
			return name, true
		}
	}
	return "", false
}

func (p *Predicate) String() string {
	parts := make([]string, len(p.terms))
	for i, term := range p.terms {
		parts[i] = term.String()
	}
	return strings.Join(parts, " and ")
}