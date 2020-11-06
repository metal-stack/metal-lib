package sign

import (
	"crypto"

	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
)

// Sign signs data with an RSA Private Key
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
