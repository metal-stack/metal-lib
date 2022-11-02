package genericcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
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
		wantOutput string
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
			wantOutput: `
| ID | NAME |
|----|------|
|  1 | one  |
`,
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
			wantOutput: `
| ID | NAME |
|----|------|
|  1 | one  |
| ID | NAME |
|----|------|
|  2 | two  |
`,
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
			wantOutput: `
| ID | NAME |
|----|------|
|  1 | one  |
| ID | NAME |
|----|------|
|  2 | two  |
`,
		},
		{
			name: "apply two entities, first one fails, second gets created",
			mockFn: func(mock *mockTestClient) {
				mock.On("Create", &testCreate{ID: "1", Name: "one"}).Return(nil, fmt.Errorf("creation error for id 1"))
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
					Action: MultiApplyErrorOnCreate,
					Error:  fmt.Errorf("error creating entity: creation error for id 1"),
				},
				{
					Action: MultiApplyUpdated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
			wantErr: fmt.Errorf("errors occurred during apply"),
			wantOutput: `
error creating entity: creation error for id 1
| ID | NAME |
|----|------|
|  2 | two  |
`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			client := newMockTestClient(t)
			fs := afero.NewMemMapFs()
			buffer := new(bytes.Buffer)
			printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
				Out:      buffer,
				Markdown: true,
				ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
					switch d := data.(type) {
					case *testResponse:
						return []string{"ID", "Name"}, [][]string{{d.ID, d.Name}}, nil
					default:
						return nil, nil, fmt.Errorf("unknown format: %T", d)
					}
				},
			}).WithOut(buffer)

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

			got, err := cli.ApplyFromFile(testFile, printer)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreInterfaces(struct{ printers.Printer }{}), testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(strings.TrimSpace(tt.wantOutput), strings.TrimSpace(buffer.String())); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
				t.Logf("expecting: \n%s", tt.wantOutput)
				t.Logf("got: \n%s", buffer.String())
			}
		})
	}
}

func mustMarshal(t *testing.T, d any) []byte {
	b, err := json.Marshal(d)
	require.NoError(t, err)
	return b
}
