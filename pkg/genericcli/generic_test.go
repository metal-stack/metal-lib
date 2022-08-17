package genericcli

func (t testCRUD) Get(id string) (*testResponse, error) {
	return t.client.Get(id)
}

func (t testCRUD) List() ([]*testResponse, error) {
	return t.client.List()
}

func (t testCRUD) Create(rq *testCreate) (*testResponse, error) {
	return t.client.Create(rq)
}

func (t testCRUD) Update(rq *testUpdate) (*testResponse, error) {
	return t.client.Update(rq)
}

func (t testCRUD) Delete(id string) (*testResponse, error) {
	return t.client.Delete(id)
}

func (t testCRUD) ToCreate(r *testResponse) (*testCreate, error) {
	return &testCreate{
		ID:   r.ID,
		Name: r.Name,
	}, nil
}

func (t testCRUD) ToUpdate(r *testResponse) (*testUpdate, error) {
	return &testUpdate{
		ID:   r.ID,
		Name: r.Name,
	}, nil
}
