package sec

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/auth"
	libjwt "github.com/metal-stack/metal-lib/jwt/jwt"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/require"
)

func TestParseTokenUnvalidated(t *testing.T) {

	grps := []string{
		"tnnt_kaas-all-all-admin", "tnnt_maas-all-all-admin", "tnnt_k8s-test-all-clusteradmin", "tnnt_k8s-qa$poc-all-clusteradmin", "tnnt_k8s-ddd$poc-all-clusteradmin", "tnnt_k8s-prod$poc-all-clusteradmin",
	}

	issuedAt := jwt.NewNumericDate(time.Unix(1557381999, 0))
	expiredAt := jwt.NewNumericDate(time.Unix(1557410799, 0))

	token, err := libjwt.GenerateToken("tnnt", grps, issuedAt.Time, expiredAt.Time)
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
				RegisteredClaims: jwt.RegisteredClaims{
					Audience:  nil,
					ExpiresAt: expiredAt,
					ID:        "",
					IssuedAt:  issuedAt,
					Issuer:    "https://dex.test.metal-stack.io/dex",
					NotBefore: nil,
					Subject:   "achim",
				},
				Audience: "theAudience",
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
		tt := tt
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

func TestParseTokenUnvalidatedUnfiltered(t *testing.T) {

	grps := []string{"tnnt_kaas-all-all-admin", "tnnt_maas-all-all-admin", "tnnt_k8s-test-all-clusteradmin", "tnnt_k8s-qa$poc-all-clusteradmin", "tnnt_k8s-ddd#ddd$poc-all-clusteradmin", "tnnt_k8s-prod$poc-all-clusteradmin"}
	var grpsRA []security.ResourceAccess
	for _, g := range grps {
		grpsRA = append(grpsRA, security.ResourceAccess(g))
	}

	issAtUnix := int64(1557381999)
	issuedAt := time.Unix(issAtUnix, 0)
	expAtUnix := int64(1557410799)
	expiredAt := time.Unix(expAtUnix, 0)

	oldToken, err := libjwt.GenerateToken("tnnt", grps, issuedAt, expiredAt)
	require.NoError(t, err)

	newTokenCfg := security.DefaultTokenCfg()
	newToken, _, _, err := security.CreateTokenAndKeys(newTokenCfg)
	require.NoError(t, err)

	newTokenCfgNameOnly := security.DefaultTokenCfg()
	newTokenCfgNameOnly.PreferredName = ""
	newTokenNameOnly, _, _, err := security.CreateTokenAndKeys(newTokenCfgNameOnly)
	require.NoError(t, err)

	type args struct {
		token string
	}
	tests := []struct {
		name       string
		args       args
		wantUser   *security.User
		wantClaims *auth.Claims
		wantErr    bool
	}{
		{
			name: "old oldToken",
			args: args{
				token: oldToken,
			},
			wantUser: &security.User{
				Issuer:  "https://dex.test.metal-stack.io/dex",
				Subject: "achim",
				EMail:   "achim.admin@tenant.de",
				Name:    "achim",
				Groups:  grpsRA,
				Tenant:  "tnnt",
			},
			wantClaims: &auth.Claims{
				ExpiresAt: expAtUnix,
				Id:        "",
				IssuedAt:  issAtUnix,
				Issuer:    "https://dex.test.metal-stack.io/dex",
				NotBefore: 0,
				Subject:   "achim",
				Audience:  "theAudience",
				Groups:    []string{"tnnt_kaas-all-all-admin", "tnnt_maas-all-all-admin", "tnnt_k8s-test-all-clusteradmin", "tnnt_k8s-qa$poc-all-clusteradmin", "tnnt_k8s-ddd#ddd$poc-all-clusteradmin", "tnnt_k8s-prod$poc-all-clusteradmin"},
				EMail:     "achim.admin@tenant.de",
				Name:      "achim",
				FederatedClaims: map[string]string{
					"connector_id": "tnnt_ldap_openldap",
					"user_id":      "cn=achim.admin,ou=People,dc=tenant,dc=de",
				},
				Roles: nil,
			},
			wantErr: false,
		},
		{
			name: "new Token",
			args: args{
				token: newToken,
			},
			wantUser: &security.User{
				Issuer:  newTokenCfg.IssuerUrl,
				Subject: newTokenCfg.Subject,
				EMail:   newTokenCfg.Email,
				Name:    newTokenCfg.PreferredName,
				Groups:  []security.ResourceAccess{security.ResourceAccess("Tn_k8s-all-all-cadm")},
				Tenant:  "",
			},
			wantClaims: &auth.Claims{
				ExpiresAt:         newTokenCfg.ExpiresAt.Unix(),
				Id:                newTokenCfg.Id,
				IssuedAt:          newTokenCfg.IssuedAt.Unix(),
				Issuer:            newTokenCfg.IssuerUrl,
				NotBefore:         newTokenCfg.IssuedAt.Unix(),
				Subject:           newTokenCfg.Subject,
				Audience:          newTokenCfg.Audience[0],
				Groups:            nil,
				EMail:             newTokenCfg.Email,
				Name:              newTokenCfg.Name,
				PreferredUsername: newTokenCfg.PreferredName,
				Roles:             []string{"Tn_k8s-all-all-cadm"},
			},
			wantErr: false,
		},
		{
			name: "new Token, name only",
			args: args{
				token: newTokenNameOnly,
			},
			wantUser: &security.User{
				Issuer:  newTokenCfgNameOnly.IssuerUrl,
				Subject: newTokenCfgNameOnly.Subject,
				EMail:   newTokenCfgNameOnly.Email,
				Name:    newTokenCfgNameOnly.Name,
				Groups:  []security.ResourceAccess{security.ResourceAccess("Tn_k8s-all-all-cadm")},
				Tenant:  "",
			},
			wantClaims: &auth.Claims{
				ExpiresAt:         newTokenCfgNameOnly.ExpiresAt.Unix(),
				Id:                newTokenCfgNameOnly.Id,
				IssuedAt:          newTokenCfgNameOnly.IssuedAt.Unix(),
				Issuer:            newTokenCfgNameOnly.IssuerUrl,
				NotBefore:         newTokenCfgNameOnly.IssuedAt.Unix(),
				Subject:           newTokenCfgNameOnly.Subject,
				Audience:          newTokenCfgNameOnly.Audience[0],
				Groups:            nil,
				EMail:             newTokenCfgNameOnly.Email,
				Name:              newTokenCfgNameOnly.Name,
				PreferredUsername: "",
				Roles:             []string{"Tn_k8s-all-all-cadm"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotClaims, err := ParseTokenUnvalidatedUnfiltered(tt.args.token)
			if (err != nil) != tt.wantErr {
				//nolint:errorlint
				t.Errorf("ParseTokenUnvalidatedUnfiltered() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUser, tt.wantUser) {
				diff := cmp.Diff(tt.wantUser, gotUser)
				t.Errorf("ParseTokenUnvalidatedUnfiltered() gotUser = %v, want %v, diff %s", gotUser, tt.wantUser, diff)
			}
			if !reflect.DeepEqual(gotClaims, tt.wantClaims) {
				fmt.Println(cmp.Diff(gotClaims, tt.wantClaims))
				t.Errorf("ParseTokenUnvalidated() got1 Claims = %v, want %v", gotClaims, tt.wantClaims)
			}
		})
	}
}
