package jwt

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSimpleToken(t *testing.T) {

	alg := jose.RS256

	publicKey, privateKey, err := security.CreateWebkeyPair(alg, "sig", 0)
	require.NoError(t, err, "error creating keypair")

	cl := jwt.Claims{
		Subject:   "subject",
		Issuer:    "issuer",
		Expiry:    jwt.NewNumericDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		NotBefore: jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Audience:  jwt.Audience{"leela", "fry"},
	}

	signer := security.MustMakeSigner(alg, privateKey)

	token, err := CreateToken(signer, cl)
	require.NoError(t, err, "error creating token")
	assert.NotEmpty(t, token)

	parsedClaims := &jwt.Claims{}
	webToken, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{alg})
	require.NoError(t, err)
	err = webToken.Claims(publicKey, parsedClaims)
	require.NoError(t, err, "error parsing claims")
	require.Equal(t, "subject", parsedClaims.Subject)
	require.Equal(t, "issuer", parsedClaims.Issuer)
}

func TestGenerateFullToken(t *testing.T) {

	alg := jose.RS256

	publicKey, privateKey, err := security.CreateWebkeyPair(alg, "sig", 0)
	require.NoError(t, err, "error creating keypair")

	cl := jwt.Claims{
		Issuer:   "https://dex.test.metal-stack.io/dex",
		Subject:  "achim",
		Audience: jwt.Audience{"cli-id1", "cli-id2"},
		Expiry:   jwt.NewNumericDate(time.Unix(1557410799, 0)),
		IssuedAt: jwt.NewNumericDate(time.Unix(1557381999, 0)),
	}

	fed := map[string]string{
		"connector_id": "tenant_ldap_openldap",
		"user_id":      "cn=achim.admin,ou=People,dc=tenant,dc=de",
	}

	privateClaims := ExtendedClaims{
		Groups: []string{
			"k8s_kaas-admin",
			"k8s_kaas-edit",
			"k8s_kaas-view",
			"k8s_development__cluster-admin",
			"k8s_production__cluster-admin",
			"k8s_staging__cluster-admin",
		},
		EMail:           "achim.admin@tenant.de",
		Name:            "achim",
		FederatedClaims: fed,
	}

	signer := security.MustMakeSigner(alg, privateKey)

	token, err := CreateToken(signer, cl, privateClaims)
	require.NoError(t, err, "error creating token")
	assert.NotEmpty(t, token)

	_, err = publicKey.MarshalJSON()
	require.NoError(t, err)

	webToken, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{alg})
	require.NoError(t, err)

	parsedClaims := &jwt.Claims{}
	extendedClaims := &ExtendedClaims{}
	err = webToken.Claims(publicKey, parsedClaims, extendedClaims)
	require.NoError(t, err, "error parsing claims")
	assert.Equal(t, "achim", parsedClaims.Subject)
	assert.Equal(t, "achim.admin@tenant.de", extendedClaims.EMail)
	assert.Equal(t, "tenant_ldap_openldap", extendedClaims.FederatedClaims["connector_id"])
	assert.Equal(t, "cn=achim.admin,ou=People,dc=tenant,dc=de", extendedClaims.FederatedClaims["user_id"])
	assert.Len(t, extendedClaims.Groups, 6)
}
