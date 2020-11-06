package sign

import (
	"crypto"
	"errors"

	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
)

// Sign signs data with an RSA Private Key and; returns the base64 encoded signature
func Sign(privateKey *rsa.PrivateKey, data []byte) (string, error) {
	signatureBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, SHA256(data))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signatureBytes), nil
}

// VerifySignature verifies the given signature with the RSA public key and the given data
func VerifySignature(pubKey *rsa.PublicKey, sig string, data []byte) (bool, error) {
	der, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false, err
	}

	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, SHA256(data), der)
	if err != nil {
		return false, err
	}
	return true, nil
}

// SHA256 generates the SHA256 checksum for the given bytes
func SHA256(in []byte) []byte {
	h := sha256.Sum256(in)
	return h[:]
}

// ExtractPubKey extracts the RSA PublicKey of the given x509.Certificate.
func ExtractPubKey(cert *x509.Certificate) (*rsa.PublicKey, error) {
	switch cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return cert.PublicKey.(*rsa.PublicKey), nil
	default:
		return nil, errors.New("certificate contains no rsa public key")
	}
}

// DecodeCertificate takes a byte slice, decodes it from the PEM format, converts it to an x509.Certificate
// object, and returns it. In case an error occurs, it returns the error.
func DecodeCertificate(bytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("could not decode the PEM-encoded certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

// DecodePrivateKey takes a byte slice, decodes it from the PEM format, converts it to an rsa.PrivateKey
// object, and returns it. In case an error occurs, it returns the error.
func DecodePrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// DecodePublicKey takes a byte slice, decodes it from the PEM format, converts it to an rsa.PublicKey
func DecodePublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA public key")
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}
