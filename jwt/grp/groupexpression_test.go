package grp

import (
	"testing"
)

func TestGroupExpression_Matches(t *testing.T) {
	type fields struct {
		groupExpr GroupExpression
	}
	type args struct {
		group Group
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "all wildcards",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "*",
					ClusterName: "*",
					Namespace:   "*",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "tenant1",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "exact match",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "myrole",
				},
			},
			want: true,
		},
		{
			name: "wildcard appprefix",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "*",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard clustername",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "*",
					Namespace:   "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard namespace",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "*",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard role",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "different app prefix",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "maas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: false,
		},
		{
			name: "different cluster",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "someothercluster",
					Namespace:   "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: false,
		},
		{
			name: "different namespace",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: false,
		},
		{
			name: "different role",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "*",
					ClusterName: "*",
					Namespace:   "*",
					Role:        "myotherrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "kaas",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: false,
		},
		{
			name: "realisitc example matches with cluster all",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					ClusterName: "mycluster",
					Namespace:   "*",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "k8s",
					ClusterTenant: "tenant2",
					ClusterName:   "all",
					Namespace:     "mynamespace",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "realisitc example matches with cluster all",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					ClusterName: "mycluster",
					Namespace:   "*",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "k8s",
					ClusterTenant: "tenant2",
					ClusterName:   "mycluster",
					Namespace:     "all",
					Role:          "myrole",
				},
			},
			want: true,
		},
		{
			name: "realisitc example no match",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					ClusterName: "mycluster",
					Namespace:   "somenamesoace",
					Role:        "role",
				},
			},
			args: args{
				group: Group{
					AppPrefix:     "k8s",
					ClusterTenant: "tenant2",
					ClusterName:   "all",
					Namespace:     "all",
					Role:          "myrole",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.fields.groupExpr
			if got := g.Matches(tt.args.group); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
