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
					FirstScope:  "*",
					SecondScope: "*",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "tenant1",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "exact match",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "mycluster",
					SecondScope: "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:   "kaas",
					FirstScope:  "mycluster",
					SecondScope: "mynamespace",
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
					FirstScope:  "mycluster",
					SecondScope: "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard firstScope",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "*",
					SecondScope: "mynamespace",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard secondScope",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "mycluster",
					SecondScope: "*",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "match wildcard role",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "mycluster",
					SecondScope: "mynamespace",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "different app prefix",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "maas",
					FirstScope:  "mycluster",
					SecondScope: "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: false,
		},
		{
			name: "different firstScope",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "someothercluster",
					SecondScope: "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: false,
		},
		{
			name: "different secondScope",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "kaas",
					FirstScope:  "mycluster",
					SecondScope: "mynamespace2",
					Role:        "myrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: false,
		},
		{
			name: "different role",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "*",
					FirstScope:  "*",
					SecondScope: "*",
					Role:        "myotherrole",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "kaas",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: false,
		},
		{
			name: "realistic example matches with firstScope all",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					FirstScope:  "xyz",
					SecondScope: "mynamespace",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "k8s",
					OnBehalfTenant: "tenant2",
					FirstScope:     "all",
					SecondScope:    "mynamespace",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "realistic example matches with secondScope all",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					FirstScope:  "mycluster",
					SecondScope: "xyz",
					Role:        "*",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "k8s",
					OnBehalfTenant: "tenant2",
					FirstScope:     "mycluster",
					SecondScope:    "all",
					Role:           "myrole",
				},
			},
			want: true,
		},
		{
			name: "realistic case-insensitive example",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					FirstScope:  "mycluster",
					SecondScope: "somenamesoace",
					Role:        "role",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "k8S",
					OnBehalfTenant: "TENANT2",
					FirstScope:     "MYCLUSTER",
					SecondScope:    "ALL",
					Role:           "ROLE",
				},
			},
			want: true,
		},
		{
			name: "realistic example no match due to role",
			fields: fields{
				groupExpr: GroupExpression{
					AppPrefix:   "k8s",
					FirstScope:  "mycluster",
					SecondScope: "somenamesoace",
					Role:        "role",
				},
			},
			args: args{
				group: Group{
					AppPrefix:      "k8S",
					OnBehalfTenant: "tenant2",
					FirstScope:     "ALL",
					SecondScope:    "ALL",
					Role:           "MYROLE",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			g := tt.fields.groupExpr
			if got := g.Matches(tt.args.group); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
