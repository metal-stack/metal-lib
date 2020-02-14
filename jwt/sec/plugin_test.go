package sec

import (
	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

// setup for most of the tests in this package
var grpr = grp.MustNewGrpr(grp.Config{
	ProviderTenant: "tnnt",
})

var plugin = NewPlugin(grpr)

type testGroupsOnBehalf struct {
	tenant string
	groups []security.ResourceAccess
}

func TestExtractUserProcessGroups(t *testing.T) {
	type args struct {
		plugin *Plugin
		claims *security.Claims
	}
	tests := []struct {
		name               string
		args               args
		wantUser           *security.User
		wantGroupsOnBehalf []testGroupsOnBehalf
		wantErr            bool
	}{
		{
			name: "NoFederatedClaim",
			args: args{
				claims: &security.Claims{
					Audience: "audience",
					Groups:   []string{},
					EMail:    "hans@demo.de",
					Name:     "hans",
				},
			},
			wantErr: true,
		},
		{
			name: "NoConnectorId",
			args: args{
				claims: &security.Claims{
					Audience:        "audience",
					Groups:          []string{},
					EMail:           "hans@demo.de",
					Name:            "hans",
					FederatedClaims: map[string]string{},
				},
			},
			wantErr: true,
		},
		{
			name: "UnparsableConnectorId",
			args: args{
				claims: &security.Claims{
					Audience: "audience",
					Groups:   []string{},
					EMail:    "hans@demo.de",
					Name:     "hans",
					FederatedClaims: map[string]string{
						"connector_id": "ldap",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "UnixLDAP",
			args: args{
				claims: &security.Claims{
					Audience: "audience",
					Groups: []string{
						"tnnt_k8s-all-all-group1",
						"tnnt_maas-all-all-maasgroup1",
						"tnnt_kaas-ddd#all-all-kaasgroup1",
						"other_kaas-all-all-group1",
						"other_kaas-ddd#all-all-group1",
						"malfrmd-kaas-all-all",
						"malfrmd_kaas-all-all",
						"malformed",
					},
					EMail: "hans@demo.de",
					Name:  "hans",
					FederatedClaims: map[string]string{
						"connector_id": "tnnt_ldap",
					},
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "hans",
				Groups: []security.ResourceAccess{
					security.ResourceAccess("k8s-all-all-group1"),
					security.ResourceAccess("maas-all-all-maasgroup1"),
					security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
				},
				Tenant: "tnnt",
			},
			wantGroupsOnBehalf: []testGroupsOnBehalf{
				{
					tenant: "ddd",
					groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-kaasgroup1")},
				},
			},
			wantErr: false,
		},
		{
			name: "ActiveDirectory",
			args: args{
				plugin: NewPlugin(grp.MustNewGrpr(grp.Config{ProviderTenant: "Tn"})),
				claims: &security.Claims{
					Audience: "audience",
					Groups: []string{
						"TnRg_Srv_Appk8s-ddd#all-all-group1_Full",
						"TnRg_Srv_Appmaas-all-all-maasgroup1_Full",
						"TnRg_Srv_Appkaas-ddd#all-all-kaasgroup1_Full",
						"DxRg_Srv_Appmaas-all-all-maasgroup2_Full",
						"DxRg_Srv_Appmaas-ddd#all-all-maasgroup2_Full",
						"FxRg_Srv_Appmaas-all-all-maasgroup3_Full",
						"other_Srv_Appkaas-all-all-group1_Edit",
						"malfrmd-kaas-all-all",
						"malfrmd_kaas-all-all",
						"malformed",
					},
					EMail: "hans@demo.de",
					Name:  "hans",
					FederatedClaims: map[string]string{
						"connector_id": "Tn_ad",
					},
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "hans",
				Groups: []security.ResourceAccess{
					security.ResourceAccess("k8s-ddd#all-all-group1"),
					security.ResourceAccess("maas-all-all-maasgroup1"),
					security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
				},
				Tenant: "Tn",
			},
			wantGroupsOnBehalf: []testGroupsOnBehalf{
				{
					tenant: "ddd",
					groups: []security.ResourceAccess{security.ResourceAccess("k8s-all-all-group1"), security.ResourceAccess("kaas-all-all-kaasgroup1")},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			plg := plugin
			if tt.args.plugin != nil {
				plg = tt.args.plugin
			}
			gotUser, err := plg.ExtractUserProcessGroups(tt.args.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractUserProcessGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotUser, tt.wantUser) {
				t.Errorf("ExtractUserProcessGroups() gotUser = %v, want %v", gotUser, tt.wantUser)
			}

			for i := range tt.wantGroupsOnBehalf {
				gob := tt.wantGroupsOnBehalf[i]
				if gotGroupsOnBehalf := plugin.GroupsOnBehalf(gotUser, gob.tenant); !reflect.DeepEqual(gotGroupsOnBehalf, gob.groups) {
					t.Errorf("groupsOnBehalf() = %v, want %v", gotGroupsOnBehalf, gob.groups)
				}
			}
		})
	}
}

func TestHasOneOfGroups(t *testing.T) {
	type args struct {
		user   *security.User
		tenant string
		groups []security.ResourceAccess
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not allowed",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-admin"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin")},
					Tenant: "tnnt",
				},
				tenant: "ddd",
				groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-admin"), security.ResourceAccess("kaas-all-all-edit"), security.ResourceAccess("kaas-all-all-something")},
			},
			want: false,
		},
		{
			name: "allowed",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-edit"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin")},
					Tenant: "tnnt",
				},
				tenant: "ddd",
				groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-view")},
			},
			want: true,
		},
		{
			name: "allowed list",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-edit"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin")},
					Tenant: "tnnt",
				},
				tenant: "ddd",
				groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-admin"), security.ResourceAccess("kaas-all-all-view")},
			},
			want: true,
		},
		{
			name: "allowed list",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-edit"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin")},
					Tenant: "tnnt",
				},
				tenant: "kkk",
				groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-admin")},
			},
			want: true,
		},
		{
			name: "DENY: not provider tenant wants to act on behalf of another tenant",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-edit"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin")},
					Tenant: "tnnt",
				},
				tenant: "kkk",
				groups: []security.ResourceAccess{security.ResourceAccess("kaas-all-all-view")},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plugin.HasOneOfGroups(tt.args.user, tt.args.tenant, tt.args.groups...); got != tt.want {
				t.Errorf("HasOneOfGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasGroupExpression(t *testing.T) {
	type args struct {
		user       *security.User
		tenant     string
		expression grp.GroupExpression
	}
	var tests = []struct {
		name string
		args args
		want bool
	}{
		{
			name: "all",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-all-all-edit"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin"),
						security.ResourceAccess("invalid-grp"),
					},
					Tenant: "tnnt",
				},
				tenant: "tnnt",
				expression: grp.GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "*",
				},
			},
			want: true,
		},
		{
			name: "explicit",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-mycluster-mynamespace-view"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin"),
					},
					Tenant: "tnnt",
				},
				tenant: "tnnt",
				expression: grp.GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "*",
				},
			},
			want: true,
		},
		{
			name: "explicit, no match",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-mycluster-mynamespace2-view"),
						security.ResourceAccess("kaas-mycluster2-mynamespace-view"),
					},
					Tenant: "tnnt",
				},
				tenant: "tnnt",
				expression: grp.GroupExpression{
					AppPrefix:   "kaas",
					ClusterName: "mycluster",
					Namespace:   "mynamespace",
					Role:        "*",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plugin.HasGroupExpression(tt.args.user, tt.args.tenant, tt.args.expression); got != tt.want {
				t.Errorf("HasGroupExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeResourceAccess(t *testing.T) {
	type args struct {
		ras [][]security.ResourceAccess
	}
	tests := []struct {
		name string
		args args
		want []security.ResourceAccess
	}{
		{
			name: "empty",
			args: args{
				ras: [][]security.ResourceAccess{},
			},
			want: nil, // slice nil value
		},
		{
			name: "single",
			args: args{
				ras: [][]security.ResourceAccess{ToResourceAccess("a")},
			},
			want: ToResourceAccess("a"),
		},
		{
			name: "two",
			args: args{
				ras: [][]security.ResourceAccess{
					ToResourceAccess("a"),
					ToResourceAccess("b"),
				},
			},
			want: ToResourceAccess("a", "b"),
		},
		{
			name: "multi",
			args: args{
				ras: [][]security.ResourceAccess{
					ToResourceAccess("a1", "a2"),
					ToResourceAccess("b"),
					ToResourceAccess("b"), // duplicate
					ToResourceAccess("c1", "c2", "c3"),
				},
			},
			want: ToResourceAccess("a1", "a2", "b", "b", "c1", "c2", "c3"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeResourceAccess(tt.args.ras...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeResourceAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantsOnBehalf(t *testing.T) {
	type args struct {
		user   *security.User
		groups []security.ResourceAccess
	}
	tests := []struct {
		name        string
		args        args
		wantTenants []string
		wantAll     bool
		wantErr     bool
	}{
		{
			name: "tnnt only",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-all-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-all-maasgroup1"),
			},
			wantTenants: []string{"tnnt"},
			wantAll:     false,
			wantErr:     false,
		},
		{
			name: "tnnt & ddd",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-all-all-maasgroup1"),
						security.ResourceAccess("maas-ddd#all-all-maasgroup1"),
						security.ResourceAccess("maas-kkk#test-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-all-maasgroup1"),
			},
			wantTenants: []string{"tnnt", "ddd"},
			wantAll:     false,
			wantErr:     false,
		},
		{
			name: "tnnt & ddd from multiple groups",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-all-all-maasgroup1"),
						security.ResourceAccess("maas-ddd#all-all-maasgroup2"),
						security.ResourceAccess("maas-kkk#test-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-all-maasgroup1", "maas-all-all-maasgroup2"),
			},
			wantTenants: []string{"tnnt", "ddd"},
			wantAll:     false,
			wantErr:     false,
		},
		{
			name: "tnnt & ddd cluster",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-mycluster-all-maasgroup1"),
						security.ResourceAccess("maas-ddd#mycluster-all-maasgroup1"),
						security.ResourceAccess("maas-kkk#test-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-mycluster-all-maasgroup1"),
			},
			wantTenants: []string{"tnnt", "ddd"},
			wantAll:     false,
			wantErr:     false,
		},
		{
			name: "tnnt & ddd namespace",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-all-myns-maasgroup1"),
						security.ResourceAccess("maas-ddd#all-myns-maasgroup1"),
						security.ResourceAccess("maas-kkk#test-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-myns-maasgroup1"),
			},
			wantTenants: []string{"tnnt", "ddd"},
			wantAll:     false,
			wantErr:     false,
		},
		{
			name: "all",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-kkk#all-all-maasgroup2"),
						security.ResourceAccess("maas-all#all-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-all-maasgroup1", "maas-all-all-maasgroup2"),
			},
			wantTenants: []string{},
			wantAll:     true,
			wantErr:     false,
		},
		{
			name: "all mixed with explicit tenants",
			args: args{
				user: &security.User{
					EMail: "hans@demo.de",
					Name:  "hans",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("k8s-ddd#all-all-group1"),
						security.ResourceAccess("maas-all#all-all-maasgroup1"),
						security.ResourceAccess("maas-all-all-maasgroup1"),
						security.ResourceAccess("maas-ddd#all-all-maasgroup1"),
						security.ResourceAccess("kaas-ddd#all-all-kaasgroup1"),
					},
					Tenant: "tnnt",
				},
				groups: ToResourceAccess("maas-all-all-maasgroup1"),
			},
			wantTenants: []string{},
			wantAll:     true,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTenants, gotAll, err := plugin.TenantsOnBehalf(tt.args.user, tt.args.groups)
			if (err != nil) != tt.wantErr {
				t.Errorf("TenantsOnBehalf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.ElementsMatch(t, gotTenants, tt.wantTenants) {
				t.Errorf("TenantsOnBehalf() gotTenants = %v, want %v", gotTenants, tt.wantTenants)
			}
			if gotAll != tt.wantAll {
				t.Errorf("TenantsOnBehalf() gotAll = %v, want %v", gotAll, tt.wantAll)
			}
		})
	}
}
