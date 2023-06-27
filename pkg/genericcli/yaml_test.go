package genericcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testYAML struct {
	ID     string   `json:"id"`
	Labels []string `json:"labels"`
}

var (
	testYAMLRaw = `

---
id: a
labels:
  - a
---
id: b
labels:
  - b
`
)

func Test_ReadAll(t *testing.T) {
	const testFile = "/test.yaml"

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		want    []testYAML
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
			want: []testYAML{
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
			m := MultiDocumentYAML[testYAML]{
				fs: afero.NewMemMapFs(),
			}

			if tt.mockFn != nil {
				tt.mockFn(m.fs)
			}

			got, err := m.ReadAll(testFile)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_ReadOneProtoWithUnderscore(t *testing.T) {
	// the json field has an underscore which is not understood by the
	// usual yaml v3 library from golang. this test ensures that we
	// can also handle proto messages with json fields with underscores properly.

	type testProto struct {
		TimeWithUnderscore *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=time,json=time_at,proto3" json:"time_at,omitempty"`
	}

	const testFile = "/test.yaml"

	now := time.Now()
	testObject := testProto{
		TimeWithUnderscore: timestamppb.New(now),
	}

	mustMarshal := func(t *testing.T, d any) []byte {
		b, err := json.MarshalIndent(d, "", "    ")
		require.NoError(t, err)
		return b
	}

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		want    *testProto
		wantErr error
	}{
		{
			name: "parsing yaml into proto message",
			mockFn: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, testFile, mustMarshal(t, testObject), 0755))
			},
			want: &testObject,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := MultiDocumentYAML[*testProto]{
				fs: afero.NewMemMapFs(),
			}

			if tt.mockFn != nil {
				tt.mockFn(m.fs)
			}

			got, err := m.ReadOne(testFile)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got, testcommon.IgnoreUnexported()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_ReadAllWithPtrSlice(t *testing.T) {
	const testFile = "/test.yaml"

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		want    []*testYAML
		wantErr error
	}{
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

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
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

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_YamlIsEqual(t *testing.T) {
	tests := []struct {
		name    string
		x       []byte
		y       []byte
		want    bool
		wantErr error
	}{
		{
			name: "yaml is equal",
			x:    []byte(`a: b`),
			y:    []byte(`a: b`),
			want: true,
		},
		{
			name: "yaml is equal indepedent of trailing spaces",
			x:    []byte(`a: b`),
			y:    []byte(`  a: b   `),
			want: true,
		},
		{
			name: "yaml is unequal",
			x:    []byte(`a: b`),
			y:    []byte(`a: c`),
			want: false,
		},
		{
			name:    "yaml is invalid",
			x:       []byte(`a: b`),
			y:       []byte(`a: b: c`),
			want:    false,
			wantErr: errors.New("error converting YAML to JSON: yaml: mapping values are not allowed in this context"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := YamlIsEqual(tt.x, tt.y)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if got != tt.want {
				t.Errorf("yamlIsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
