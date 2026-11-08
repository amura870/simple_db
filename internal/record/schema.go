package record

import "fmt"

type FieldType int

const (
	Integer FieldType = iota
	Varchar
)

type Field struct {
	Name   string
	Type   FieldType
	Length int // 文字列型の最大文字数. 整数では0
}

// テーブルのフィールドのリストを持つ
type Schema struct {
	fields []Field
}

func (s *Schema) AddIntField(name string) error {
	return s.add(Field{Name: name, Type: Integer})
}

func (s *Schema) AddStringField(name string, maxChars int) error {
	if maxChars < 0 {
		return fmt.Errorf("negative string length: %d", maxChars)
	}
	return s.add(Field{Name: name, Type: Varchar, Length: maxChars})
}

func (s *Schema) add(f Field) error {
	if f.Name == "" {
		return fmt.Errorf("empty field name")
	}
	for _, existing := range s.fields {
		if existing.Name == f.Name {
			return fmt.Errorf("duplicate filed: %s", f.Name)
		}
	}
	s.fields = append(s.fields, f)
	return nil
}

func (s *Schema) Fields() []Field {
	return append([]Field(nil), s.fields...)
}
