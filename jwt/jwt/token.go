package jwt

import (
	"github.com/metal-stack/security"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"time"
)

type ExtendedClaims struct {
	Groups          []string          `json:"groups"`
	EMail           string            `json:"email"`
	Name            string            `json:"name"`
	FederatedClaims map[string]string `json:"federated_claims"`
}

// CreateToken creates a jwt token with the given claims
func CreateToken(signer jose.Signer, cl interface{}, privateClaims ...interface{}) (string, error) {
	builder := jwt.Signed(signer).Claims(cl)
	for i := range privateClaims {
		builder = builder.Claims(privateClaims[i])
	}
	raw, err := builder.CompactSerialize()
	if err != nil {
		return "", err
	}
	return raw, nil
}

func GenerateToken(tenant string, grps []string, issuedAt, expiresAt time.Time) (string, error) {
	alg := jose.RS256

	_, privateKey, err := security.CreateWebkeyPair(alg, "sig", 0)
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Issuer:   "https://dex.test.metal-stack.io/dex",
		Subject:  "achim",
		Audience: jwt.Audience{"theAudience"},
		Expiry:   jwt.NewNumericDate(expiresAt),
		IssuedAt: jwt.NewNumericDate(issuedAt),
	}

	fed := map[string]string{
		"connector_id": tenant + "_ldap_openldap",
		"user_id":      "cn=achim.admin,ou=People,dc=tenant,dc=de",
	}

	privateClaims := ExtendedClaims{
		Groups:          grps,
		EMail:           "achim.admin@tenant.de",
		Name:            "achim",
		FederatedClaims: fed,
	}

	signer := security.MustMakeSigner(alg, privateKey)

	token, err := CreateToken(signer, cl, privateClaims)

	return token, err
}
