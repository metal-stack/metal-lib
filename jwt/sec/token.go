package sec

import (
	"fmt"
	"github.com/metal-stack/security"
	"gopkg.in/square/go-jose.v2/jwt"
)

// ParseTokenUnvalidated extracts information from the given jwt token without validating it
func (p *Plugin) ParseTokenUnvalidated(token string) (*security.User, *security.Claims, error) {

	parsedClaims := &security.Claims{}
	webToken, err := jwt.ParseSigned(token)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token: %s", err)
	}

	err = webToken.UnsafeClaimsWithoutVerification(parsedClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing token claims: %s", err)
	}

	user, err := p.ExtractUserProcessGroups(parsedClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting user: %s", err)
	}

	return user, parsedClaims, nil
}
