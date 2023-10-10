package jwt

import (
	"time"

	"github.com/metal-stack/security"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
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
		Issuer:    "https://dex.test.metal-stack.io/dex",
		Subject:   "achim",
		Audience:  jwt.Audience{"theAudience"},
		Expiry:    jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		NotBefore: nil,
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

// DexTokenGenerator provides MustCreateTokenAndKeys as the dex-variant-impl of the security.TokenProvider to create tokens for testing.
type DexTokenGenerator struct {
	FederatedClaims map[string]string
}

// MustCreateTokenAndKeys creates a keyset and token, panics on error
func (d *DexTokenGenerator) MustCreateTokenAndKeys(cfg *security.TokenCfg) (token string, pubKey jose.JSONWebKey, privKey jose.JSONWebKey) {
	token, pubKey, privKey, err := d.CreateTokenAndKeys(cfg)
	if err != nil {
		panic(err)
	}
	return token, pubKey, privKey
}

// CreateTokenAndKeys creates a keyset and token
func (d *DexTokenGenerator) CreateTokenAndKeys(cfg *security.TokenCfg) (token string, pubKey jose.JSONWebKey, privKey jose.JSONWebKey, err error) {
	pubKey, privKey, err = security.CreateWebkeyPair(cfg.Alg, "sig", cfg.KeyBitlength)
	if err != nil {
		return "", jose.JSONWebKey{}, jose.JSONWebKey{}, err
	}

	cl := jwt.Claims{
		Issuer:    cfg.IssuerUrl,
		Subject:   cfg.Subject,
		Audience:  cfg.Audience,
		Expiry:    jwt.NewNumericDate(cfg.ExpiresAt),
		NotBefore: jwt.NewNumericDate(cfg.IssuedAt),
		IssuedAt:  jwt.NewNumericDate(cfg.IssuedAt),
		ID:        cfg.Id,
	}

	pcl := ExtendedClaims{
		Name:            cfg.Name,
		EMail:           cfg.Email,
		Groups:          cfg.Roles,
		FederatedClaims: d.FederatedClaims,
	}

	signer := security.MustMakeSigner(cfg.Alg, privKey)

	token, err = CreateToken(signer, cl, pcl)
	if err != nil {
		return "", jose.JSONWebKey{}, jose.JSONWebKey{}, err
	}

	return token, pubKey, privKey, nil
}
