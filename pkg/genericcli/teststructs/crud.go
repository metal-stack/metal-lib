package teststructs

import "fmt"

type (
	// ProtoTestCRUD can be used for tests using MultiArgGenericCLI.
	ProtoTestCRUD struct {
		Foo *Foo
	}

	TestClient interface {
		Get(id string) (*TestResponse, error)
		List() ([]*TestResponse, error)
		Create(rq *TestCreate) (*TestResponse, error)
		Update(rq *TestUpdate) (*TestResponse, error)
		Delete(id string) (*TestResponse, error)
		Convert(r *TestResponse) ([]string, *TestCreate, *TestUpdate, error)
	}
	TestCRUD   struct{ client TestClient }
	TestCreate struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	TestUpdate struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	TestResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
)

func (m *ProtoTestCRUD) Get(ids ...string) (*Foo, error) {
	return m.Foo, nil
}

func (m *ProtoTestCRUD) List() ([]*Foo, error) {
	return []*Foo{m.Foo}, nil
}

func (m *ProtoTestCRUD) Create(rq *Foo) (*Foo, error) {
	panic("not implemented")
}

func (m *ProtoTestCRUD) Update(rq *Foo) (*Foo, error) {
	return rq, nil
}

func (m *ProtoTestCRUD) Delete(ids ...string) (*Foo, error) {
	panic("not implemented")
}

func (m *ProtoTestCRUD) Convert(r *Foo) ([]string, *Foo, *Foo, error) {
	return []string{}, r, r, nil
}

func NewTestCRUD(client TestClient) *TestCRUD {
	return &TestCRUD{client: client}
}

func (t TestCRUD) Get(ids ...string) (*TestResponse, error) {
	id, err := getExactlyOneArg(ids)
	if err != nil {
		return nil, err
	}

	return t.client.Get(id)
}

func (t TestCRUD) List() ([]*TestResponse, error) {
	return t.client.List()
}

func (t TestCRUD) Create(rq *TestCreate) (*TestResponse, error) {
	return t.client.Create(rq)
}

func (t TestCRUD) Update(rq *TestUpdate) (*TestResponse, error) {
	return t.client.Update(rq)
}

func (t TestCRUD) Delete(ids ...string) (*TestResponse, error) {
	id, err := getExactlyOneArg(ids)
	if err != nil {
		return nil, err
	}

	return t.client.Delete(id)
}

func (t TestCRUD) Convert(r *TestResponse) ([]string, *TestCreate, *TestUpdate, error) {
	return []string{r.ID},
		&TestCreate{
			ID:   r.ID,
			Name: r.Name,
		}, &TestUpdate{
			ID:   r.ID,
			Name: r.Name,
		}, nil
}

func getExactlyOneArg(args []string) (string, error) {
	switch count := len(args); count {
	case 0:
		return "", fmt.Errorf("a single positional arg is required, none was provided")
	case 1:
		return args[0], nil
	default:
		return "", fmt.Errorf("a single positional arg is required, %d were provided", count)
	}
}
