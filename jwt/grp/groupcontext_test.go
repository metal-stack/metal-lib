package grp

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestGrprconfig(t *testing.T) {

	_, err := NewGrpr(Config{})
	require.Error(t, err)

	_, err = NewGrpr(Config{
		ProviderTenant: "Tn",
	})
	require.NoError(t, err)

	_, err = NewGrpr(Config{
		ProviderTenant: "tnnt",
	})
	require.NoError(t, err)
}

var grpr = MustNewGrpr(Config{
	ProviderTenant: "Tn",
})

type ExpectedErrorTest struct {
	groupString   string
	expectedError error
}

var invalidGroupTests = []ExpectedErrorTest{
	{
		groupString:   "",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "tnnt_aas-cluster-namespace-role-stuff",
		expectedError: errInvalidFormat,
	},
}

func TestParsegroupInvalid(t *testing.T) {
	for _, test := range invalidGroupTests {

		t.Run(test.groupString, func(t *testing.T) {

			_, err := grpr.ParseGroupName(test.groupString)

			require.Error(t, err, test.groupString)
			require.Equal(t, test.expectedError, err, test.groupString)
		})
	}
}

type ExpectedGroupTest struct {
	groupString         string
	result              *Group
	prefixedGroupString string
	fullGroupString     string
}

var validGroupTests = []ExpectedGroupTest{
	{
		groupString:         "kaas-clustername-namespace-admin",
		result:              &Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "", Namespace: "namespace", Role: "admin"},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-clustername-namespace-admin",
	},
	{
		groupString:         "kaas-ddd#clustername-namespace-admin",
		result:              &Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "ddd", Namespace: "namespace", Role: "admin"},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-ddd#clustername-namespace-admin",
	},
	{
		groupString:         "kaas-ddd#all-all-admin",
		result:              &Group{AppPrefix: "kaas", ClusterName: "all", ClusterTenant: "ddd", Namespace: "all", Role: "admin"},
		prefixedGroupString: "oidc:all-admin",
		fullGroupString:     "kaas-ddd#all-all-admin",
	},
	{
		groupString:         "kaas-cluster-namespace-role",
		result:              &Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "kaas-ddd#cluster-namespace-role",
		result:              &Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "namespace", Role: "role"},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-ddd#cluster-namespace-role",
	},
	{
		groupString:         "kaas-ddd#cluster--role",
		result:              &Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "", Role: "role"},
		prefixedGroupString: "oidc:-role",
		fullGroupString:     "kaas-ddd#cluster--role",
	},
	{
		groupString:         "kaas-cluster-namespace-role",
		result:              &Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "kaas-cluster-namespace-role",
		result:              &Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "KAAS-cluster-NameSpace-ROLE",
		result:              &Group{AppPrefix: "KAAS", ClusterName: "cluster", ClusterTenant: "", Namespace: "NameSpace", Role: "ROLE"},
		prefixedGroupString: "oidc:NameSpace-ROLE",
		fullGroupString:     "KAAS-cluster-NameSpace-ROLE",
	},
}

func TestParseGroupValid(t *testing.T) {
	for _, test := range validGroupTests {

		t.Run(test.groupString, func(t *testing.T) {
			result, err := grpr.ParseGroupName(test.groupString)

			require.NoError(t, err, test.groupString)
			require.Equal(t, test.result, result, test.groupString)
		})
	}
}

func TestToPrefixedGroupValid(t *testing.T) {
	for _, test := range validGroupTests {
		result, err := grpr.ParseGroupName(test.groupString)

		require.NoError(t, err, test.groupString)
		require.Equal(t, test.prefixedGroupString, result.ToPrefixedGroupString("oidc:"))
	}
}

func TestToFullGroupValid(t *testing.T) {
	for _, test := range validGroupTests {
		result, err := grpr.ParseGroupName(test.groupString)

		require.NoError(t, err, test.groupString)
		require.Equal(t, test.fullGroupString, result.ToFullGroupString())
	}
}

var invalidGroupADTests = []ExpectedErrorTest{
	{
		groupString:   "",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace_Full",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace-role-stuff",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "TnPgX_Srv_Appkaas-cluster-namespace-role-stuff_Full",
		expectedError: errInvalidFormat,
	},
}

func TestParseADInvalid(t *testing.T) {
	runGroupContextExpectedErrorTests(t, invalidGroupADTests, grpr.ParseADGroup)
}

type ExpectedGroupContextTest struct {
	groupString         string
	result              *GroupContext
	prefixedGroupString string
	fullGroupString     string
}

var validGroupADTests = []ExpectedGroupContextTest{
	{
		groupString:         "TnPg_Srv_Appkaas-clustername-namespace-admin_full",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "", Namespace: "namespace", Role: "admin"}},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-clustername-namespace-admin",
	},
	{
		groupString:         "TnPg_Srv_Appkaas-ddd#clustername-namespace-admin_full",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "ddd", Namespace: "namespace", Role: "admin"}},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-ddd#clustername-namespace-admin",
	},
	{
		groupString:         "TnPg_Srv_Appkaas-cluster-namespace-role_Edit",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "TnPg_Srv_Appkaas-ddd#cluster-namespace-role_Full",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-ddd#cluster-namespace-role",
	},
	{
		groupString:         "TnPg_Srv_Appkaas-ddd#cluster--role_Mod",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "", Role: "role"}},
		prefixedGroupString: "oidc:-role",
		fullGroupString:     "kaas-ddd#cluster--role",
	},
	{
		groupString:         "TnPg_Srv_Appkaas-cluster-namespace-role_Edit",
		result:              &GroupContext{TenantPrefix: "tn", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "DpRg_Srv_Appkaas-cluster-namespace-role_Edit",
		result:              &GroupContext{TenantPrefix: "dp", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "DpRg_Srv_Appkaas-clu$ter-namespace-role_Edit",
		result:              &GroupContext{TenantPrefix: "dp", Group: Group{AppPrefix: "kaas", ClusterName: "clu$ter", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-clu$ter-namespace-role",
	},
}

func TestParseADValid(t *testing.T) {
	runGroupContextExpectedResultTests(t, validGroupADTests, grpr.ParseADGroup)
}

var invalidGroupUXTests = []ExpectedErrorTest{
	{
		groupString:   "",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "kaas-cluster-namespace-role-stuff",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "tnnt_aas-cluster-namespace-role-stuff",
		expectedError: errInvalidFormat,
	},
	{
		groupString:   "tnnt_aas-cluster-namespace-role_stuff",
		expectedError: errInvalidFormat,
	},
}

func TestParseUXInvalid(t *testing.T) {
	runGroupContextExpectedErrorTests(t, invalidGroupUXTests, grpr.ParseUnixLDAPGroup)
}

var validGroupUXTests = []ExpectedGroupContextTest{
	{
		groupString:         "tnnt_kaas-clustername-namespace-admin",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "", Namespace: "namespace", Role: "admin"}},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-clustername-namespace-admin",
	},
	{
		groupString:         "tnnt_kaas-ddd#clustername-namespace-admin",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "clustername", ClusterTenant: "ddd", Namespace: "namespace", Role: "admin"}},
		prefixedGroupString: "oidc:namespace-admin",
		fullGroupString:     "kaas-ddd#clustername-namespace-admin",
	},
	{
		groupString:         "tnnt_kaas-cluster-namespace-role",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "tnnt_kaas-ddd#cluster-namespace-role",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-ddd#cluster-namespace-role",
	},
	{
		groupString:         "tnnt_kaas-ddd#cluster--role",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "ddd", Namespace: "", Role: "role"}},
		prefixedGroupString: "oidc:-role",
		fullGroupString:     "kaas-ddd#cluster--role",
	},
	{
		groupString:         "tnnt_kaas-cluster-namespace-role",
		result:              &GroupContext{TenantPrefix: "tnnt", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "ddd_kaas-cluster-namespace-role",
		result:              &GroupContext{TenantPrefix: "ddd", Group: Group{AppPrefix: "kaas", ClusterName: "cluster", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-cluster-namespace-role",
	},
	{
		groupString:         "ddd_kaas-clu$ter-namespace-role",
		result:              &GroupContext{TenantPrefix: "ddd", Group: Group{AppPrefix: "kaas", ClusterName: "clu$ter", ClusterTenant: "", Namespace: "namespace", Role: "role"}},
		prefixedGroupString: "oidc:namespace-role",
		fullGroupString:     "kaas-clu$ter-namespace-role",
	},
}

func TestParseUXValid(t *testing.T) {
	runGroupContextExpectedResultTests(t, validGroupUXTests, grpr.ParseUnixLDAPGroup)
}

func runGroupContextExpectedErrorTests(t *testing.T, tests []ExpectedErrorTest, parseFn GroupContextParseFunc) {
	for _, test := range tests {

		t.Run(test.groupString, func(t *testing.T) {

			_, err := parseFn(test.groupString)

			require.Error(t, err, test.groupString)
			require.Equal(t, test.expectedError, err, test.groupString)
		})
	}
}

func runGroupContextExpectedResultTests(t *testing.T, tests []ExpectedGroupContextTest, parseFn GroupContextParseFunc) {
	for _, test := range tests {

		t.Run(test.groupString, func(t *testing.T) {
			result, err := parseFn(test.groupString)

			require.NoError(t, err, test.groupString)
			require.Equal(t, test.result, result, test.groupString)
			require.Equal(t, test.prefixedGroupString, result.ToPrefixedGroupString("oidc:"))
			require.Equal(t, test.fullGroupString, result.ToFullGroupString())
		})
	}
}

func TestGroupEncodeName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no",
			args: args{name: "group"},
			want: "group",
		},
		{
			name: "sep",
			args: args{name: "composite-group"},
			want: "composite$group",
		},
		{
			name: "multi sep",
			args: args{name: "multi-composite-group"},
			want: "multi$composite$group",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpr.GroupEncodeName(tt.args.name); got != tt.want {
				t.Errorf("GroupEncodeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupEncodeNames(t *testing.T) {
	type args struct {
		names []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "mixed",
			args: args{names: []string{"", "group", "comp-group", "multi-comp-group"}},
			want: []string{"", "group", "comp$group", "multi$comp$group"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpr.GroupEncodeNames(tt.args.names); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupEncodeNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromStringToFullGroupStringRoundtrip(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "kaas-all-all-admin",
			wantErr: false,
		},
		{
			name:    "kaas---admin",
			wantErr: false,
		},
		{
			name:    "kaas-ddd#--admin",
			wantErr: false,
		},
		{
			name:    "sadasd",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpr.ParseGroupName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				require.Equal(t, tt.name, got.ToFullGroupString())
			}
		})
	}
}

func TestGroup_ToOnBehalfGroupString(t *testing.T) {
	type fields struct {
		AppPrefix     string
		ClusterTenant string
		ClusterName   string
		Namespace     string
		Role          string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "w/o clustertenant",
			fields: fields{AppPrefix: "kaas", ClusterTenant: "", ClusterName: "bnk", Namespace: "project1", Role: "admin"},
			want:   "kaas-bnk-project1-admin",
		},
		{
			name:   "clustertenant",
			fields: fields{AppPrefix: "kaas", ClusterTenant: "ddd", ClusterName: "bnk", Namespace: "project2", Role: "admin"},
			want:   "kaas-bnk-project2-admin",
		},
		{
			name:   "clustertenant spec char",
			fields: fields{AppPrefix: "kaas", ClusterTenant: "my$tenant", ClusterName: "bnk", Namespace: "project3", Role: "view"},
			want:   "kaas-bnk-project3-view",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{
				AppPrefix:     tt.fields.AppPrefix,
				ClusterTenant: tt.fields.ClusterTenant,
				ClusterName:   tt.fields.ClusterName,
				Namespace:     tt.fields.Namespace,
				Role:          tt.fields.Role,
			}
			if got := g.ToCanonicalGroupString(); got != tt.want {
				t.Errorf("ToCanonicalGroupString() = %v, want %v", got, tt.want)
			}
		})
	}
}
