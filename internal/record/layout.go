package record

import (
	"fmt"
	"unicode/utf8"

	"example.com/simpledb/internal/file"
)

const intBytes = 4

type fieldLayout struct {
	Field
	offset int
}

// スキーマを受け取り、各フィールドのオフセットとスロットサイズを計算する
type Layout struct {
	fields   map[string]fieldLayout // フィールド名とオフセットの対応関係
	order    []string
	slotSize int // 1つのレコードを格納するのに必要な容量
}

// SchemaからLayoutを作る
func NewLayout(s *Schema) (*Layout, error) {
	l := &Layout{
		fields: make(map[string]fieldLayout),
	}
	pos := intBytes // 先頭のintBytes分は、空きor使用中のフラグに使う
	maxInt := int(^uint(0) >> 1)

	for _, f := range s.Fields() {
		size := intBytes // 整数の長さはintBytesバイト
		// 文字列の場合は長さが可変なので、サイズを別途計算
		if f.Type == Varchar {
			if f.Length > (maxInt-intBytes)/utf8.UTFMax {
				return nil, fmt.Errorf("record too large: %s", f.Name)
			}
			size = file.MaxLength(f.Length * utf8.UTFMax) // 最大のケース (UTFMax) を考慮して領域を確保
		}
		if size > maxInt-pos {
			return nil, fmt.Errorf("record too large")
		}
		l.fields[f.Name] = fieldLayout{
			Field:  f,
			offset: pos,
		}
		l.order = append(l.order, f.Name)
		pos += size
	}
	l.slotSize = pos
	return l, nil
}

func (l *Layout) SlotSize() int {
	return l.slotSize
}

// フィールド名からオフセットを取得する
func (l *Layout) Offset(name string) (int, error) {
	f, ok := l.fields[name]
	if !ok {
		return 0, fmt.Errorf("unknown field: %s", name)
	}
	return f.offset, nil
}
