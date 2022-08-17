package genericcli

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestApplyFromFile(t *testing.T) {
	const testFile = "/apply.yaml"

	tests := []struct {
		name       string
		mockFn     func(mock *mockTestClient)
		fileMockFn func(fs afero.Fs)
		want       MultiApplyResults[*testResponse]
		wantErr    error
	}{
		{
			name: "apply single entity, create it",
			mockFn: func(mock *mockTestClient) {
				mock.On("Create", &testCreate{ID: "1", Name: "one"}).Return(&testResponse{ID: "1", Name: "one"}, nil)
			},
			fileMockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, mustMarshal(t, &testCreate{
					ID:   "1",
					Name: "one",
				}), 0755))
			},
			want: MultiApplyResults[*testResponse]{
				{
					Action: MultiApplyCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
			},
		},
		{
			name: "apply two entities, create both",
			mockFn: func(mock *mockTestClient) {
				mock.On("Create", &testCreate{ID: "1", Name: "one"}).Return(&testResponse{ID: "1", Name: "one"}, nil)
				mock.On("Create", &testCreate{ID: "2", Name: "two"}).Return(&testResponse{ID: "2", Name: "two"}, nil)
			},
			fileMockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(`---
id: "1"
name: one
---
id: "2"
name: two
`), 0755))
			},
			want: MultiApplyResults[*testResponse]{
				{
					Action: MultiApplyCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
				{
					Action: MultiApplyCreated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
		},
		{
			name: "apply two entities, update one",
			mockFn: func(mock *mockTestClient) {
				mock.On("Create", &testCreate{ID: "1", Name: "one"}).Return(&testResponse{ID: "1", Name: "one"}, nil)
				mock.On("Create", &testCreate{ID: "2", Name: "two"}).Return(nil, AlreadyExistsError()).Once()
				mock.On("Update", &testUpdate{ID: "2", Name: "two"}).Return(&testResponse{ID: "2", Name: "two"}, nil).Once()
			},
			fileMockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(`---
id: "1"
name: one
---
id: "2"
name: two
`), 0755))
			},
			want: MultiApplyResults[*testResponse]{
				{
					Action: MultiApplyCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
				{
					Action: MultiApplyUpdated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			client := newMockTestClient(t)
			fs := afero.NewMemMapFs()

			cli := GenericCLI[*testCreate, *testUpdate, *testResponse]{
				crud:   testCRUD{client: client},
				fs:     fs,
				parser: MultiDocumentYAML[*testResponse]{fs: fs},
			}

			if tt.mockFn != nil {
				tt.mockFn(client)
			}

			if tt.fileMockFn != nil {
				tt.fileMockFn(fs)
			}

			got, err := cli.ApplyFromFile(testFile)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func mustMarshal(t *testing.T, d any) []byte {
	b, err := json.Marshal(d)
	require.NoError(t, err)
	return b
}
