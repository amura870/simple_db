package record

func (s *Schema) Field(name string) (Field, bool) {
	for _, f := range s.fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

func (s *Schema) HasField(name string) bool {
	_, ok := s.Field(name)
	return ok
}

func (s *Schema) AddAll(other *Schema) error {
	for _, f := range other.Fields() {
		if err := s.add(f); err != nil {
			return err
		}
	}
	return nil
}
