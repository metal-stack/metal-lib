package grp

import (
	"reflect"
	"testing"
)

func TestNewGroup(t *testing.T) {
	type args struct {
		app            string
		onBehalfTenant string
		firstScope     string
		secondScope    string
		role           string
	}
	tests := []struct {
		name string
		args args
		want *Group
	}{
		{
			name: "plain",
			args: args{
				app:            "kaas",
				onBehalfTenant: "all",
				firstScope:     "mycluster",
				secondScope:    "myns",
				role:           "myrole",
			},
			want: &Group{
				AppPrefix:      "kaas",
				OnBehalfTenant: "all",
				FirstScope:     "mycluster",
				SecondScope:    "myns",
				Role:           "myrole",
			},
		},
		{
			name: "encode",
			args: args{
				app:            "kaas",
				onBehalfTenant: "all",
				firstScope:     "my-cluster",
				secondScope:    "my-ns",
				role:           "myrole",
			},
			want: &Group{
				AppPrefix:      "kaas",
				OnBehalfTenant: "all",
				FirstScope:     "my$cluster",
				SecondScope:    "my$ns",
				Role:           "myrole",
			},
		},
	}

	grpr := MustNewGrpr(Config{ProviderTenant: "tnnt"})

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := grpr.NewGroup(tt.args.app, tt.args.onBehalfTenant, tt.args.firstScope, tt.args.secondScope, tt.args.role); !reflect.DeepEqual(got, tt.want) {
				//nolint:errorlint
				t.Errorf("NewGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
