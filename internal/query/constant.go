package query

import (
	"fmt"
	"strconv"
	"strings"

	"example.com/simpledb/internal/record"
)

// 定数を表す.
// フィールドを非公開にして生成経路を２つのコンストラクタに制限することで、kindと値を一致させる
type Constant struct {
	kind record.FieldType // Int or VarChar
	ival int32            // intの定数
	sval string           // stringの定数
}

func NewIntConstant(value int32) Constant {
	return Constant{kind: record.Integer, ival: value}
}

func NewStringConstant(value string) Constant {
	return Constant{kind: record.Varchar, sval: value}
}

func (c Constant) AsInt() (int32, error) {
	if c.kind != record.Integer {
		return 0, fmt.Errorf("constant is not an integer")
	}
	return c.ival, nil
}

func (c Constant) AsString() (string, error) {
	if c.kind != record.Varchar {
		return "", fmt.Errorf("constant is not a string")
	}
	return c.sval, nil
}

func (c Constant) Equals(other Constant) bool {
	return c == other
}

func (c Constant) CompareTo(other Constant) (int, error) {
	if c.kind != other.kind {
		return 0, fmt.Errorf("cannot compare different constant types")
	}
	if c.kind == record.Varchar {
		return strings.Compare(c.sval, other.sval), nil
	}
	// cがintの時
	if c.ival < other.ival {
		return -1, nil
	}
	if c.ival > other.ival {
		return 1, nil
	}
	return 0, nil
}

func (c Constant) String() string {
	if c.kind == record.Varchar {
		return c.sval
	}
	return strconv.FormatInt(int64(c.ival), 10)
}
