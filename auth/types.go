package auth

//IssuerConfig holds the config for openID connect issuer
type IssuerConfig struct {
	// Client-ID
	ClientID string
	// ClientSecret
	ClientSecret string
	// Issuer-URL
	IssuerURL string
	// IssuerCA if any
	IssuerCA string
}
