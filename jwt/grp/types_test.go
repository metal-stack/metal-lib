package grp

import (
	"reflect"
	"testing"
)

func TestNewGroup(t *testing.T) {
	type args struct {
		app           string
		clusterTenant string
		cluster       string
		namespace     string
		role          string
	}
	tests := []struct {
		name string
		args args
		want *Group
	}{
		{
			name: "plain",
			args: args{
				app:           "kaas",
				clusterTenant: "all",
				cluster:       "mycluster",
				namespace:     "myns",
				role:          "myrole",
			},
			want: &Group{
				AppPrefix:     "kaas",
				ClusterTenant: "all",
				ClusterName:   "mycluster",
				Namespace:     "myns",
				Role:          "myrole",
			},
		},
		{
			name: "encode",
			args: args{
				app:           "kaas",
				clusterTenant: "all",
				cluster:       "my-cluster",
				namespace:     "my-ns",
				role:          "myrole",
			},
			want: &Group{
				AppPrefix:     "kaas",
				ClusterTenant: "all",
				ClusterName:   "my$cluster",
				Namespace:     "my$ns",
				Role:          "myrole",
			},
		},
	}

	grpr := MustNewGrpr(Config{ProviderTenant: "tnnt"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpr.NewGroup(tt.args.app, tt.args.clusterTenant, tt.args.cluster, tt.args.namespace, tt.args.role); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
