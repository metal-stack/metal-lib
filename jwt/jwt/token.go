package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"log"
	"time"
)

type ExtendedClaims struct {
	Groups          []string          `json:"groups"`
	EMail           string            `json:"email"`
	Name            string            `json:"name"`
	FederatedClaims map[string]string `json:"federated_claims"`
}

// CreateToken creates a jwt token with the given claims
func CreateToken(signer jose.Signer, cl jwt.Claims, privateClaims ...interface{}) (string, error) {
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

// MustMakeSigner creates a Signer and panics if an error occurs
func MustMakeSigner(alg jose.SignatureAlgorithm, k interface{}) jose.Signer {
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: k}, nil)
	if err != nil {
		panic("failed to create signer:" + err.Error())
	}
	return sig
}

// CreateWebkeyPair creates a JSONWebKey-Pair.
// alg is one of jose signature-algorithm constants, e.g. jose.RS256.
// use is "sig" for signature or "enc" for encryption, see https://tools.ietf.org/html/rfc7517#page-6
func CreateWebkeyPair(alg jose.SignatureAlgorithm, use string) (jose.JSONWebKey, jose.JSONWebKey, error) {
	kid := uuid.New().String()

	var publicKey crypto.PrivateKey
	var privateKey crypto.PublicKey
	var err error

	publicKey, privateKey, err = GenerateSigningKey(jose.SignatureAlgorithm(alg), 4096)
	if err != nil {
		return jose.JSONWebKey{}, jose.JSONWebKey{}, err
	}

	salg := string(alg)
	publicWebKey := jose.JSONWebKey{Key: publicKey, KeyID: kid, Algorithm: salg, Use: use}
	privateWebKey := jose.JSONWebKey{Key: privateKey, KeyID: kid, Algorithm: salg, Use: use}

	if privateWebKey.IsPublic() || !publicWebKey.IsPublic() || !privateWebKey.Valid() || !publicWebKey.Valid() {
		log.Fatalf("invalid keys were generated")
	}

	return publicWebKey, privateWebKey, nil
}

// GenerateSigningKey generates a keypair for corresponding SignatureAlgorithm.
func GenerateSigningKey(alg jose.SignatureAlgorithm, bits int) (crypto.PublicKey, crypto.PrivateKey, error) {
	switch alg {
	case jose.ES256, jose.ES384, jose.ES512, jose.EdDSA:
		keylen := map[jose.SignatureAlgorithm]int{
			jose.ES256: 256,
			jose.ES384: 384,
			jose.ES512: 521, // sic!
			jose.EdDSA: 256,
		}
		if bits != 0 && bits != keylen[alg] {
			return nil, nil, errors.New("invalid elliptic curve key size, this algorithm does not support arbitrary size")
		}
	case jose.RS256, jose.RS384, jose.RS512, jose.PS256, jose.PS384, jose.PS512:
		if bits == 0 {
			bits = 2048
		}
		if bits < 2048 {
			return nil, nil, errors.New("invalid key size for RSA key, 2048 or more is required")
		}
	}
	switch alg {
	case jose.ES256:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, err
	case jose.ES384:
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, err
	case jose.ES512:
		key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, err
	case jose.EdDSA:
		pub, key, err := ed25519.GenerateKey(rand.Reader)
		return pub, key, err
	case jose.RS256, jose.RS384, jose.RS512, jose.PS256, jose.PS384, jose.PS512:
		key, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, err
	default:
		return nil, nil, fmt.Errorf("unknown algorithm %s for signing key", alg)
	}
}

func GenerateToken(tenant string, grps []string, issuedAt, expiresAt time.Time) (string, error) {
	alg := jose.RS256

	_, privateKey, err := CreateWebkeyPair(alg, "sig")
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

	signer := MustMakeSigner(alg, privateKey)

	token, err := CreateToken(signer, cl, privateClaims)

	return token, err
}
