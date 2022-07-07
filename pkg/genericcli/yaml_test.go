package genericcli

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type testYAML struct {
	ID     string
	Labels []string
}

var (
	testYAMLRaw = `---
id: a
labels:
  - a
---
id: b
labels:
  - b
`
	errorComparer = cmp.Comparer(func(x, y error) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil && y != nil {
			return false
		}
		if x != nil && y == nil {
			return false
		}
		return x.Error() == y.Error()
	})
)

func Test_ReadAll(t *testing.T) {
	const testFile = "/test.yaml"

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		want    []*testYAML
		wantErr error
	}{
		{
			name: "parsing empty file",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(""), 0755))
			},
			want: nil,
		},
		{
			name: "parsing multi-document yaml",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(testYAMLRaw), 0755))
			},
			want: []*testYAML{
				{
					ID:     "a",
					Labels: []string{"a"},
				},
				{
					ID:     "b",
					Labels: []string{"b"},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := MultiDocumentYAML[*testYAML]{
				fs: afero.NewMemMapFs(),
			}

			if tt.mockFn != nil {
				tt.mockFn(m.fs)
			}

			got, err := m.ReadAll(testFile)

			if diff := cmp.Diff(tt.wantErr, err, errorComparer); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_ReadIndex(t *testing.T) {
	const testFile = "/test.yaml"

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		index   int
		want    *testYAML
		wantErr error
	}{
		{
			name: "request zero index",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(testYAMLRaw), 0755))
			},
			index: 0,
			want: &testYAML{
				ID:     "a",
				Labels: []string{"a"},
			},
		},
		{
			name: "request one index",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(testYAMLRaw), 0755))
			},
			index: 1,
			want: &testYAML{
				ID:     "b",
				Labels: []string{"b"},
			},
		},
		{
			name: "not existing index",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, []byte(testYAMLRaw), 0755))
			},
			index:   2,
			wantErr: fmt.Errorf("index not found in document: 2"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := MultiDocumentYAML[*testYAML]{
				fs: afero.NewMemMapFs(),
			}

			if tt.mockFn != nil {
				tt.mockFn(m.fs)
			}

			got, err := m.ReadIndex(testFile, tt.index)

			if diff := cmp.Diff(tt.wantErr, err, errorComparer); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
