package httperrors

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHTTPErrorResponse_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		text    []byte
		want    *HTTPErrorResponse
		wantErr error
	}{
		{
			name: "unmarshals empty json",
			text: []byte("{}"),
			want: &HTTPErrorResponse{},
		},
		{
			name: "unmarshals json response",
			text: []byte(`{"statuscode": 300, "message":"test"}`),
			want: &HTTPErrorResponse{
				StatusCode: 300,
				Message:    "test",
			},
		},
		{
			name:    "errors on invalid json",
			text:    []byte("nojson"),
			wantErr: fmt.Errorf("endpoint did not return a json response:\nnojson"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTPErrorResponse{}
			err := h.UnmarshalText(tt.text)
			if tt.wantErr != nil {
				var errText string
				if err != nil {
					errText = err.Error()
				}
				if tt.wantErr.Error() != errText {
					t.Errorf("HTTPErrorResponse.UnmarshalText() want = %s, got = %s", errText, tt.wantErr.Error())
				}
				return
			}
			if diff := cmp.Diff(h, tt.want); diff != "" {
				t.Errorf("HTTPErrorResponse.UnmarshalText() diff = %s", diff)
			}
		})
	}
}
