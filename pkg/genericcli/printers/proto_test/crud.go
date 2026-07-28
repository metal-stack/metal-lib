package proto_test

// TestCRUD can be used for tests using MultiArgGenericCLI.
type TestCRUD struct {
	Foo *Foo
}

func (m *TestCRUD) Get(ids ...string) (*Foo, error) {
	return m.Foo, nil
}

func (m *TestCRUD) List() ([]*Foo, error) {
	return []*Foo{m.Foo}, nil
}

func (m *TestCRUD) Create(rq *Foo) (*Foo, error) {
	panic("not implemented")
}

func (m *TestCRUD) Update(rq *Foo) (*Foo, error) {
	return rq, nil
}

func (m *TestCRUD) Delete(ids ...string) (*Foo, error) {
	panic("not implemented")
}

func (m *TestCRUD) Convert(r *Foo) ([]string, *Foo, *Foo, error) {
	return []string{}, r, r, nil
}
