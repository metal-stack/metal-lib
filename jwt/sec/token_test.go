package sec

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-cmp/cmp"
	token "github.com/metal-stack/metal-lib/jwt/jwt"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

func TestParseTokenUnvalidated(t *testing.T) {

	grps := []string{
		"tnnt_kaas-all-all-admin", "tnnt_maas-all-all-admin", "tnnt_k8s-test-all-clusteradmin", "tnnt_k8s-qa$poc-all-clusteradmin", "tnnt_k8s-ddd$poc-all-clusteradmin", "tnnt_k8s-prod$poc-all-clusteradmin",
	}

	issAtUnix := int64(1557381999)
	issuedAt := time.Unix(issAtUnix, 0)
	expAtUnix := int64(1557410799)
	expiredAt := time.Unix(expAtUnix, 0)
	token, err := token.GenerateToken("tnnt", grps, issuedAt, expiredAt)
	require.NoError(t, err)

	type args struct {
		token string
	}
	tests := []struct {
		name       string
		args       args
		wantUser   *security.User
		wantClaims *security.Claims
		wantErr    bool
	}{
		{
			name: "achim",
			args: args{
				token: token,
			},
			wantUser: &security.User{
				EMail:  "achim.admin@tenant.de",
				Name:   "achim",
				Groups: ToResourceAccess("kaas-all-all-admin", "maas-all-all-admin", "k8s-test-all-clusteradmin", "k8s-qa$poc-all-clusteradmin", "k8s-ddd$poc-all-clusteradmin", "k8s-prod$poc-all-clusteradmin"),
				Tenant: "tnnt",
			},
			wantClaims: &security.Claims{
				StandardClaims: jwt.StandardClaims{
					Audience:  "",
					ExpiresAt: expAtUnix,
					Id:        "",
					IssuedAt:  issAtUnix,
					Issuer:    "https://dex.test.fi-ts.io/dex",
					NotBefore: 0,
					Subject:   "achim",
				},
				Audience: []interface{}{"theAudience"},
				Groups:   []string{"tnnt_kaas-all-all-admin", "tnnt_maas-all-all-admin", "tnnt_k8s-test-all-clusteradmin", "tnnt_k8s-qa$poc-all-clusteradmin", "tnnt_k8s-ddd$poc-all-clusteradmin", "tnnt_k8s-prod$poc-all-clusteradmin"},
				EMail:    "achim.admin@tenant.de",
				Name:     "achim",
				FederatedClaims: map[string]string{
					"connector_id": "tnnt_ldap_openldap",
					"user_id":      "cn=achim.admin,ou=People,dc=tenant,dc=de",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := plugin.ParseTokenUnvalidated(tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTokenUnvalidated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantUser) {
				t.Errorf("ParseTokenUnvalidated() got User = %v, want %v", got, tt.wantUser)
			}
			if !reflect.DeepEqual(got1, tt.wantClaims) {
				fmt.Println(cmp.Diff(got1, tt.wantClaims))
				t.Errorf("ParseTokenUnvalidated() got1 Claims = %v, want %v", got1, tt.wantClaims)
			}
		})
	}
}
