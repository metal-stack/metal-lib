package genericcli

func (t testCRUD) Get(ids ...string) (*testResponse, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		return nil, err
	}

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

func (t testCRUD) Delete(ids ...string) (*testResponse, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		return nil, err
	}

	return t.client.Delete(id)
}

func (t testCRUD) Convert(r *testResponse) ([]string, *testCreate, *testUpdate, error) {
	return []string{r.ID},
		&testCreate{
			ID:   r.ID,
			Name: r.Name,
		}, &testUpdate{
			ID:   r.ID,
			Name: r.Name,
		}, nil
}
