package genericcli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"buf.build/go/protoyaml"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers/proto_test"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

type editorSetup = func(t *testing.T) (editorPath string, cleanup func())

type editTestCase struct {
	name       string
	setupFn    editorSetup
	mockFn     func(mock *mockTestClient)
	args       []string
	wantErr    func(t *testing.T, err error)
	wantResult *testResponse
}

func Test_Edit(t *testing.T) {
	tests := []editTestCase{
		{
			name: "edit succeeds with changes",
			setupFn: func(t *testing.T) (string, func()) {
				return fakeEditorChange(t, &testUpdate{ID: "foo", Name: "two"})
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
				mock.On("Update", &testUpdate{ID: "foo", Name: "two"}).Return(&testResponse{ID: "foo", Name: "two"}, nil)
			},
			args:       []string{"foo"},
			wantResult: &testResponse{ID: "foo", Name: "two"},
		},
		{
			name: "no changes returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return fakeEditorChange(t, &testUpdate{ID: "foo", Name: "one"})
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
			},
			args:       []string{"foo"},
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "no changes were made") },
			wantResult: nil,
		},
		{
			name: "editor failure returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return makeFailEditor(t)
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
			},
			args:       []string{"foo"},
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "exit status 1") },
			wantResult: nil,
		},
		{
			name: "get fails returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return makeFailEditor(t)
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(nil, fmt.Errorf("not found"))
			},
			args:       []string{"foo"},
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "not found") },
			wantResult: nil,
		},
		{
			name:       "wrong number of args returns error",
			mockFn:     func(mock *mockTestClient) {},
			args:       []string{"foo", "bar"},
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "2 were provided") },
			wantResult: nil,
		},
		{
			name: "CRUD update fails returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return fakeEditorChange(t, &testUpdate{ID: "foo", Name: "two"})
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
				mock.On("Update", &testUpdate{ID: "foo", Name: "two"}).Return(nil, fmt.Errorf("update failed"))
			},
			args:       []string{"foo"},
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "error updating entity: update failed") },
			wantResult: nil,
		},
		{
			name: "no args returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return makeFailEditor(t)
			},
			mockFn:     func(mock *mockTestClient) {},
			args:       nil,
			wantErr:    func(t *testing.T, err error) { require.ErrorContains(t, err, "none was provided") },
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockTestClient(t)
			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			if tt.setupFn != nil {
				editorPath, cleanup := tt.setupFn(t)
				defer cleanup()
				t.Setenv("EDITOR", editorPath)
			}

			cli := &MultiArgGenericCLI[*testCreate, *testUpdate, *testResponse]{
				crud:   testCRUD{client: mock},
				fs:     afero.NewOsFs(),
				parser: MultiDocumentYAML[*testResponse]{fs: afero.NewOsFs()},
			}

			got, err := cli.Edit(1, tt.args)

			if tt.wantErr != nil {
				tt.wantErr(t, err)
			} else {
				require.NoError(t, err)
			}

			if diff := cmp.Diff(tt.wantResult, got, testcommon.IgnoreUnexported()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

type editAndPrintTestCase struct {
	name    string
	setupFn editorSetup
	mockFn  func(mock *mockTestClient)
	args    []string
	wantErr func(t *testing.T, err error)
}

func Test_EditAndPrint(t *testing.T) {
	tests := []editAndPrintTestCase{
		{
			name: "edit and print succeeds with changes",
			setupFn: func(t *testing.T) (string, func()) {
				return fakeEditorChange(t, &testUpdate{ID: "foo", Name: "two"})
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
				mock.On("Update", &testUpdate{ID: "foo", Name: "two"}).Return(&testResponse{ID: "foo", Name: "two"}, nil)
			},
			args: []string{"foo"},
		},
		{
			name: "edit with no changes returns error",
			setupFn: func(t *testing.T) (string, func()) {
				return fakeEditorChange(t, &testUpdate{ID: "foo", Name: "one"})
			},
			mockFn: func(mock *mockTestClient) {
				mock.On("Get", "foo").Return(&testResponse{ID: "foo", Name: "one"}, nil)
			},
			args:    []string{"foo"},
			wantErr: func(t *testing.T, err error) { require.ErrorContains(t, err, "no changes were made") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockTestClient(t)
			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			out := new(bytes.Buffer)
			p := printers.NewYAMLPrinter().WithOut(out)

			if tt.setupFn != nil {
				editorPath, cleanup := tt.setupFn(t)
				defer cleanup()
				t.Setenv("EDITOR", editorPath)
			}

			cli := &MultiArgGenericCLI[*testCreate, *testUpdate, *testResponse]{
				crud:   testCRUD{client: mock},
				fs:     afero.NewOsFs(),
				parser: MultiDocumentYAML[*testResponse]{fs: afero.NewOsFs()},
			}

			err := cli.EditAndPrint(1, tt.args, p)

			if tt.wantErr != nil {
				tt.wantErr(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func fakeEditorChange(t *testing.T, afterUpdate any) (editorPath string, cleanup func()) {
	dir, err := os.MkdirTemp("", "edit-editor")
	require.NoError(t, err)

	var changedYAML []byte
	if msg, ok := afterUpdate.(proto.Message); ok {
		changedYAML, err = protoyaml.Marshal(msg)
	} else {
		changedYAML, err = yaml.Marshal(&afterUpdate)
	}
	require.NoError(t, err)

	changedRef := filepath.Join(dir, "changed.yaml")
	err = os.WriteFile(changedRef, changedYAML, 0644)
	require.NoError(t, err)

	// Shell script that copies changed content to the temp file passed as $1
	scriptPath := filepath.Join(dir, "editor.sh")
	script := fmt.Sprintf("#!/bin/sh\ncp %s \"$1\"\n", changedRef)
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	cleanup = func() { _ = os.RemoveAll(dir) }

	return scriptPath, cleanup
}

func makeFailEditor(t *testing.T) (editorPath string, cleanup func()) {
	dir, err := os.MkdirTemp("", "edit-editor")
	require.NoError(t, err)

	scriptPath := filepath.Join(dir, "editor.sh")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1\n"), 0755)
	require.NoError(t, err)

	return scriptPath, func() { _ = os.RemoveAll(dir) }
}

func Test_Edit_WithProto(t *testing.T) {
	foo := &proto_test.Foo{
		Text:  "original",
		State: proto_test.State_STATE_PENDING,
	}

	changedFoo := &proto_test.Foo{
		Text:  "changed",
		State: proto_test.State_STATE_ACTIVE,
	}

	t.Run("edit proto succeeds with changes", func(t *testing.T) {
		editorPath, cleanup := fakeEditorChange(t, changedFoo)
		defer cleanup()
		t.Setenv("EDITOR", editorPath)

		crud := &proto_test.TestCRUD{Foo: foo}
		cli := &MultiArgGenericCLI[*proto_test.Foo, *proto_test.Foo, *proto_test.Foo]{
			crud:   crud,
			fs:     afero.NewOsFs(),
			parser: MultiDocumentYAML[*proto_test.Foo]{fs: afero.NewOsFs()},
		}

		got, err := cli.Edit(1, []string{"some-id"})
		require.NoError(t, err)
		require.True(t, proto.Equal(changedFoo, got))
	})

	t.Run("edit proto with no changes returns error", func(t *testing.T) {
		editorPath, cleanup := fakeEditorChange(t, foo)
		defer cleanup()
		t.Setenv("EDITOR", editorPath)

		crud := &proto_test.TestCRUD{Foo: foo}
		cli := &MultiArgGenericCLI[*proto_test.Foo, *proto_test.Foo, *proto_test.Foo]{
			crud:   crud,
			fs:     afero.NewOsFs(),
			parser: MultiDocumentYAML[*proto_test.Foo]{fs: afero.NewOsFs()},
		}

		_, err := cli.Edit(1, []string{"some-id"})
		require.ErrorContains(t, err, "no changes were made")
	})
}
