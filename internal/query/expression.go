package query

import "example.com/simpledb/internal/record"

type Expression struct {
	value   Constant
	field   string
	isField bool
}

func NewConstantExpression(value Constant) Expression {
	return Expression{value: value}
}

func NewFieldExpression(name string) Expression {
	return Expression{field: name, isField: true}
}

func (e Expression) IsFieldName() bool { return e.isField }

// boolがfalseなら、要求した種類の式ではない。
func (e Expression) AsConstant() (Constant, bool) {
	return e.value, !e.isField
}

func (e Expression) AsFieldName() (string, bool) {
	return e.field, e.isField
}

func (e Expression) Evaluate(s Scan) (Constant, error) {
	if e.isField {
		return s.GetVal(e.field)
	}
	return e.value, nil
}

func (e Expression) AppliesTo(schema *record.Schema) bool {
	return !e.isField || schema.HasField(e.field)
}

func (e Expression) String() string {
	if e.isField {
		return e.field
	}
	return e.value.String()
}
