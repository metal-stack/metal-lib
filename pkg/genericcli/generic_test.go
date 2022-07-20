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
