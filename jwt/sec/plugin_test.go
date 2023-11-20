package sec

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/assert"
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
		name     string
		args     args
		wantUser *security.User
		wantErr  bool
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
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
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
		})
	}
}

func TestGenericOIDCExtractUserProcessGroups(t *testing.T) {
	type args struct {
		plugin       *Plugin
		issuerConfig *security.IssuerConfig
		claims       *security.GenericOIDCClaims
	}
	tests := []struct {
		name               string
		args               args
		wantUser           *security.User
		wantGroupsOnBehalf []testGroupsOnBehalf
		wantErr            error
	}{
		{
			name: "Minimal no directory type",
			args: args{
				issuerConfig: &security.IssuerConfig{
					Annotations: map[string]string{
						OidcDirectory: "xx",
					},
					Tenant:   "tn",
					Issuer:   "https://issuer.example.com",
					ClientID: "client123",
				},
				claims: &security.GenericOIDCClaims{
					Claims: jwt.Claims{
						Audience: jwt.Audience{"audience"},
					},
					Roles:             []string{},
					EMail:             "hans@demo.de",
					Name:              "Hans Meiser",
					PreferredUsername: "xyz4711",
				},
			},
			wantErr: errors.New("invalid directoryType xx"),
		},
		{
			name: "Minimal ldap",
			args: args{
				issuerConfig: &security.IssuerConfig{
					Annotations: map[string]string{
						OidcDirectory: "ldap",
					},
					Tenant:   "tnnt",
					Issuer:   "https://issuer.example.com",
					ClientID: "client123",
				},
				claims: &security.GenericOIDCClaims{
					Claims: jwt.Claims{
						Audience: jwt.Audience{"audience"},
					},
					Roles:             []string{"tnnt_kaas-all-all-admin"},
					EMail:             "hans@demo.de",
					Name:              "Hans Meiser",
					PreferredUsername: "xyz4711",
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "xyz4711",
				Groups: []security.ResourceAccess{
					security.ResourceAccess("kaas-all-all-admin"),
				},
				Tenant: "tnnt",
			},
			wantErr: nil,
		},
		{
			name: "Minimal ad",
			args: args{
				issuerConfig: &security.IssuerConfig{
					Annotations: map[string]string{
						OidcDirectory: "ad",
					},
					Tenant:   "Tn",
					Issuer:   "https://issuer.example.com",
					ClientID: "client123",
				},
				claims: &security.GenericOIDCClaims{
					Claims: jwt.Claims{
						Audience: jwt.Audience{"audience"},
					},
					Roles:             []string{"TnRg_Srv_Appkaas-all-all-admin_Full"},
					EMail:             "hans@demo.de",
					Name:              "Hans Meiser",
					PreferredUsername: "xyz4711",
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "xyz4711",
				Groups: []security.ResourceAccess{
					security.ResourceAccess("kaas-all-all-admin"),
				},
				Tenant: "Tn",
			},
			wantErr: nil,
		},
		{
			name: "LDAP",
			args: args{
				issuerConfig: &security.IssuerConfig{
					Annotations: map[string]string{
						OidcDirectory: "ldap",
					},
					Tenant:   "tnnt",
					Issuer:   "https://issuer.example.com",
					ClientID: "client123",
				},
				claims: &security.GenericOIDCClaims{
					Claims: jwt.Claims{
						Audience: jwt.Audience{"audience"},
					},
					Roles: []string{
						"tnnt_k8s-all-all-group1",
						"tnnt_maas-all-all-maasgroup1",
						"tnnt_kaas-ddd#all-all-kaasgroup1",
						"other_kaas-all-all-group1",
						"other_kaas-ddd#all-all-group1",
						"malfrmd-kaas-all-all",
						"malfrmd_kaas-all-all",
						"malformed",
					},
					EMail:             "hans@demo.de",
					Name:              "Hans Meiser",
					PreferredUsername: "xyz4711",
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "xyz4711",
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
			wantErr: nil,
		},
		{
			name: "ActiveDirectory",
			args: args{
				plugin: NewPlugin(grp.MustNewGrpr(grp.Config{ProviderTenant: "Tn"})),
				issuerConfig: &security.IssuerConfig{
					Annotations: map[string]string{
						OidcDirectory: "ad",
					},
					Tenant:   "Tn",
					Issuer:   "https://issuer.example.com",
					ClientID: "client123",
				},
				claims: &security.GenericOIDCClaims{
					Claims: jwt.Claims{
						Audience: jwt.Audience{"audience"},
					},
					Roles: []string{
						"TnRg_Srv_Appk8s-all#all-all-role_Full",
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
					EMail:             "hans@demo.de",
					Name:              "Hans Meiser",
					PreferredUsername: "xyz4711",
				},
			},
			wantUser: &security.User{
				EMail: "hans@demo.de",
				Name:  "xyz4711",
				Groups: []security.ResourceAccess{
					security.ResourceAccess("k8s-all#all-all-role"),
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
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			plg := plugin
			if tt.args.plugin != nil {
				plg = tt.args.plugin
			}
			gotUser, err := plg.GenericOIDCExtractUserProcessGroups(tt.args.issuerConfig, tt.args.claims)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("ExtractUserProcessGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotUser, tt.wantUser) {
				diff := cmp.Diff(tt.wantUser, gotUser)
				t.Errorf("ExtractUserProcessGroups() gotUser = %v, want %v, diff %s", gotUser, tt.wantUser, diff)
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

func TestHasGroupExpression(t *testing.T) {
	type expr struct {
		expr grp.GroupExpression
		want bool
	}
	type args struct {
		user           *security.User
		resourceTenant string
		expression     []expr
	}
	var tests = []struct {
		name string
		args args
	}{
		{
			name: "all-all in groups",
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
				resourceTenant: "tnnt",
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "mynamespace2",
							Role:        "*",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "*",
							FirstScope:  "*",
							SecondScope: "*",
							Role:        "*",
						},
						want: true,
					},
					{ // cadm not in groups
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "cadm",
						},
						want: false,
					},
					{ // maas not in groups
						expr: grp.GroupExpression{
							AppPrefix:   "maas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "admin",
						},
						want: false,
					},
				},
			},
		},
		{
			name: "match any tenant",
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
				resourceTenant: "*", // wildcard  matches any tenant
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "admin",
						},
						want: true,
					},
					{ // cadm not in groups
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "cadm",
						},
						want: false,
					},
					{ // maas not in groups
						expr: grp.GroupExpression{
							AppPrefix:   "maas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "cadm",
						},
						want: false,
					},
				},
			},
		},
		{
			name: "wrong tenant",
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
				resourceTenant: "xyz",
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: false,
					},
					{ // match any, but the tenant does not match
						expr: grp.GroupExpression{
							AppPrefix:   "*",
							FirstScope:  "*",
							SecondScope: "*",
							Role:        "*",
						},
						want: false,
					},
				},
			},
		},
		{
			name: "explicit",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas_mycluster2-mynamespace-view"), // illegal gets skipped
						security.ResourceAccess("kaas_mycluster2-mynamespace"),      // illegal gets skipped
						security.ResourceAccess("kaas-mycluster-mynamespace-view"),
						security.ResourceAccess("kaas-ddd#all-all-view"),
						security.ResourceAccess("kaas-kkk#all-all-admin"),
					},
					Tenant: "tnnt",
				},
				resourceTenant: "tnnt",
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "view",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "*",
							SecondScope: "mynamespace",
							Role:        "view",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "*",
							Role:        "view",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: true,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: false,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace2",
							Role:        "*",
						},
						want: false,
					},
				},
			},
		},
		{
			name: "no resource tenant",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-mycluster-mynamespace-view"),
						security.ResourceAccess("kaas-xyz#mycluster2-mynamespace-view"),
					},
					Tenant: "tnnt",
				},
				resourceTenant: "", // no tenant given, there is no default
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: false,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: false,
					},
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "*",
							Role:        "*",
						},
						want: false,
					},
				},
			},
		},
		{
			name: "on behalf",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-mycluster-mynamespace2-view"),
						security.ResourceAccess("kaas-xyz#mycluster2-mynamespace-view"),
					},
					Tenant: "tnnt",
				},
				resourceTenant: "xyz",
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: true,
					},
				},
			},
		},
		{
			name: "on behalf all",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-mycluster-mynamespace2-view"),
						security.ResourceAccess("kaas-all#mycluster2-mynamespace-view"),
					},
					Tenant: "tnnt",
				},
				resourceTenant: "xyz",
				expression: []expr{
					{
						expr: grp.GroupExpression{
							AppPrefix:   "kaas",
							FirstScope:  "mycluster2",
							SecondScope: "mynamespace",
							Role:        "*",
						},
						want: true,
					},
				},
			},
		},
		{
			name: "encoded group and expression",
			args: args{
				user: &security.User{
					EMail: "",
					Name:  "",
					Groups: []security.ResourceAccess{
						security.ResourceAccess("kaas-my$cluster-my$namespace2-view"),
						security.ResourceAccess("kaas-all#my$cluster2-my$namespace-admin"),
					},
					Tenant: "tnnt",
				},
				resourceTenant: "tnnt",
				expression: []expr{
					{
						expr: func() grp.GroupExpression {
							g, _ := grp.NewGrpr(grp.Config{ProviderTenant: "x"})
							p := NewPlugin(g)
							return p.NewGroupExpression("kaas", "my-cluster", "my-namespace2", "view")
						}(),
						want: true,
					},
					{
						expr: func() grp.GroupExpression {
							g, _ := grp.NewGrpr(grp.Config{ProviderTenant: "x"})
							p := NewPlugin(g)
							return p.NewGroupExpression("kaas", "my-cluster2", "my-namespace", "edit")
						}(),
						want: false,
					},
					{ // provider tenant is not checked in this "on behalf" scenario, the match is only on api, group scopes and role
						expr: func() grp.GroupExpression {
							g, _ := grp.NewGrpr(grp.Config{ProviderTenant: "x"})
							p := NewPlugin(g)
							return p.NewGroupExpression("kaas", "my-cluster2", "my-namespace", "admin")
						}(),
						want: true,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		for _, exp := range tt.args.expression {
			exp := exp
			t.Run(fmt.Sprintf("%s:%v", tt.name, exp.expr), func(t *testing.T) {
				if got := plugin.HasGroupExpression(tt.args.user, tt.args.resourceTenant, exp.expr); got != exp.want {
					t.Errorf("HasGroupExpression(%v) = %v, want %v", exp.expr, got, exp.want)
				}
			})
		}
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
		tt := tt
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
		tt := tt
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
