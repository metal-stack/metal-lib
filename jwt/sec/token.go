package sec

import (
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/metal-stack/metal-lib/auth"
	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/security"
)

var signatureAlgorithms = []jose.SignatureAlgorithm{
	jose.RS256,
	jose.RS384,
	jose.RS512,
	jose.ES256,
	jose.ES384,
	jose.ES512,
	jose.PS256,
	jose.PS384,
	jose.PS512,
	jose.EdDSA,
}

// ParseTokenUnvalidated extracts information from the given jwt token without validating it
func (p *Plugin) ParseTokenUnvalidated(token string) (*security.User, *security.Claims, error) {

	parsedClaims := &security.Claims{}
	webToken, err := jwt.ParseSigned(token, signatureAlgorithms)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token: %w", err)
	}

	err = webToken.UnsafeClaimsWithoutVerification(parsedClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token claims: %w", err)
	}

	user, err := p.ExtractUserProcessGroups(parsedClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting user: %w", err)
	}

	return user, parsedClaims, nil
}

// ParseTokenUnvalidated extracts information from the given jwt token without validating it.
// FederatedClaims are optional and
// ResourceAccess are constructed from Roles and Groups claims.
func ParseTokenUnvalidatedUnfiltered(token string) (*security.User, *auth.Claims, error) {

	parsedClaims := &auth.Claims{}
	webToken, err := jwt.ParseSigned(token, signatureAlgorithms)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token: %w", err)
	}

	err = webToken.UnsafeClaimsWithoutVerification(parsedClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token claims: %w", err)
	}

	// check federated claims
	tenant := ""
	var res []security.ResourceAccess
	if parsedClaims.FederatedClaims != nil {
		// "old" token with groups-claim
		cid := parsedClaims.FederatedClaims["connector_id"]
		if cid != "" {
			tenant, _, err = grp.ParseConnectorId(cid)
			if err == nil {
				for _, g := range parsedClaims.Groups {
					res = append(res, security.ResourceAccess(g))
				}
			}
		}
	} else {
		// "new" token, add roles claims
		for _, g := range parsedClaims.Roles {
			res = append(res, security.ResourceAccess(g))
		}
	}

	user := &security.User{
		Issuer:  parsedClaims.Issuer,
		Subject: parsedClaims.Subject,
		Name:    parsedClaims.Username(),
		EMail:   parsedClaims.EMail,
		Groups:  res,
		Tenant:  tenant,
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error extracting user: %w", err)
	}

	return user, parsedClaims, nil
}
