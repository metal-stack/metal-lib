package genericcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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
		name           string
		mockFn         func(mock *mockTestClient)
		fileMockFn     func(fs afero.Fs)
		want           BulkResults[*testResponse]
		wantOutput     string
		wantBulkOutput string
		wantErr        error
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
			want: BulkResults[*testResponse]{
				{
					Action: BulkCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
			},
			wantOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
`,
			wantBulkOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
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
			want: BulkResults[*testResponse]{
				{
					Action: BulkCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
				{
					Action: BulkCreated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
			wantOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
| ID | NAME |
|----|------|
| 2  | two  |
`,
			wantBulkOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
| 2  | two  |
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
			want: BulkResults[*testResponse]{
				{
					Action: BulkCreated,
					Result: &testResponse{
						ID:   "1",
						Name: "one",
					},
				},
				{
					Action: BulkUpdated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
			wantOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
| ID | NAME |
|----|------|
| 2  | two  |
`,
			wantBulkOutput: `
| ID | NAME |
|----|------|
| 1  | one  |
| 2  | two  |
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
			want: BulkResults[*testResponse]{
				{
					Action: BulkErrorOnCreate,
					Error:  fmt.Errorf("error creating entity: creation error for id 1"),
				},
				{
					Action: BulkUpdated,
					Result: &testResponse{
						ID:   "2",
						Name: "two",
					},
				},
			},
			wantOutput: `
error creating entity: creation error for id 1
| ID | NAME |
|----|------|
| 2  | two  |
`,
			wantBulkOutput: `
| ID | NAME |
|----|------|
| 2  | two  |
`,
			wantErr: fmt.Errorf("error creating entity: creation error for id 1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := newMockCLI(t, tt.mockFn, tt.fileMockFn)
			got, err := cli.ApplyFromFile(testFile)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreInterfaces(struct{ printers.Printer }{}), cmpopts.IgnoreTypes(time.Duration(0)), testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}

			for _, ttt := range []struct {
				name string
				bulk bool
				want string
			}{
				{
					name: "intermediate output",
					bulk: false,
					want: tt.wantOutput,
				},
				{
					name: "bulk output",
					bulk: true,
					want: tt.wantBulkOutput,
				},
			} {
				t.Run(ttt.name, func(t *testing.T) {
					cli = newMockCLI(t, tt.mockFn, tt.fileMockFn)
					buffer := new(bytes.Buffer)
					printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
						Out:      buffer,
						Markdown: true,
						ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
							switch d := data.(type) {
							case *testResponse:
								return []string{"ID", "Name"}, [][]string{{d.ID, d.Name}}, nil
							case []*testResponse:
								var rows [][]string
								for i := range d {
									rows = append(rows, []string{d[i].ID, d[i].Name})
								}
								return []string{"ID", "Name"}, rows, nil
							default:
								return nil, nil, fmt.Errorf("unknown format: %T", d)
							}
						},
					}).WithOut(buffer)
					cli.bulkPrint = ttt.bulk

					err = cli.ApplyFromFileAndPrint(testFile, printer)
					wantErr := tt.wantErr
					if !ttt.bulk && tt.wantErr != nil {
						wantErr = fmt.Errorf("errors occurred during the process")
					}
					if diff := cmp.Diff(wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
						t.Errorf("error diff (+got -want):\n %s", diff)
					}

					if diff := cmp.Diff(strings.TrimSpace(ttt.want), strings.TrimSpace(buffer.String())); diff != "" {
						t.Errorf("diff (+got -want):\n %s", diff)
						t.Logf("expecting: \n%s", ttt.want)
						t.Logf("got: \n%s", buffer.String())
					}
				})
			}
		})
	}
}

func newMockCLI(t *testing.T, mockFn func(mock *mockTestClient), fileMockFn func(fs afero.Fs)) *MultiArgGenericCLI[*testCreate, *testUpdate, *testResponse] {
	client := newMockTestClient(t)
	fs := afero.NewMemMapFs()

	cli := MultiArgGenericCLI[*testCreate, *testUpdate, *testResponse]{
		crud:   testCRUD{client: client},
		fs:     fs,
		parser: MultiDocumentYAML[*testResponse]{fs: fs},
	}

	if mockFn != nil {
		mockFn(client)
	}

	if fileMockFn != nil {
		fileMockFn(fs)
	}

	return &cli
}

func mustMarshal(t *testing.T, d any) []byte {
	b, err := json.Marshal(d)
	require.NoError(t, err)
	return b
}
