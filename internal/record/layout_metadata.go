package record

// Layoutが持つフィールド一覧を取得する
func (l *Layout) Schema() *Schema {
	s := &Schema{}
	for _, name := range l.order {
		s.fields = append(s.fields, l.fields[name].Field)
	}
	return s
}

// カタログの内容をそのまま復元する
func NewStoredLayout(s *Schema, offsets map[string]int, slotSize int) *Layout {
	l := &Layout{
		fields:   make(map[string]fieldLayout),
		slotSize: slotSize,
	}
	for _, f := range s.Fields() {
		l.fields[f.Name] = fieldLayout{Field: f, offset: offsets[f.Name]}
		l.order = append(l.order, f.Name)
	}
	return l
}
