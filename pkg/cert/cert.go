package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"
)

const (
	CACert         = 0
	SelfSignedCert = 1
	ClientCert     = 2
	ServerCert     = 3
)

type CSR struct {
	CommonName    string
	DNSNames      []string
	Organization  string
	LifetimeYears int
	CertType      int
}

// GenerateCert returns a new Key/Cert pair from the given CSR
// Usage:
//
//	for CertType == ClientCert or CertType == ServerCert provide a caKey and caCert
//	for CertType == CACert or CertType == SelfSignedCert set caKey and caCert to nil
func GenerateCert(csr CSR, caKey *string, caCert *string) (key *string, cert *string, err error) {
	var (
		caCertRaw   *x509.Certificate
		caKeyRaw    *rsa.PrivateKey
		keyUsage    x509.KeyUsage
		extKeyUsage []x509.ExtKeyUsage
	)

	if (csr.CertType == ServerCert || csr.CertType == ClientCert) && (caKey == nil || caCert == nil) {
		return nil, nil, errors.New("requesting certificate without CA key/cert pair")
	}
	if (csr.CertType == CACert || csr.CertType == SelfSignedCert) && (caKey != nil || caCert != nil) {
		return nil, nil, errors.New("requesting selfsinged (CA) certificate, but CA key/cert pair is given")
	}

	// generate key
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// generate certificate
	randSerial, err := randomSerialNumber()
	if err != nil {
		return nil, nil, err
	}

	switch csr.CertType {
	case CACert:
		keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	case SelfSignedCert:
		keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageServerAuth)
	case ServerCert:
		keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageServerAuth)
	case ClientCert:
		keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	isCA := csr.CertType == CACert || csr.CertType == SelfSignedCert
	newCert := &x509.Certificate{
		SerialNumber: randSerial,
		Subject: pkix.Name{
			Organization: []string{csr.Organization},
			CommonName:   csr.CommonName,
		},
		DNSNames:              csr.DNSNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(csr.LifetimeYears, 0, 0),
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		IsCA:                  isCA,
		BasicConstraintsValid: isCA,
	}

	if isCA {
		caKeyRaw = privkey
		caCertRaw = newCert
	} else {
		// load CA
		certDERBlock, _ := pem.Decode([]byte(*caCert))
		if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
			return nil, nil, errors.New("failed to read the CA certificate: unexpected content")
		}
		caCertRaw, err = x509.ParseCertificate(certDERBlock.Bytes)
		if err != nil {
			return nil, nil, errors.New("failed to parse the CA certificate")
		}

		caKeyDERBlock, _ := pem.Decode([]byte(*caKey))
		fmt.Printf("caKeyDERBlock: %#v\n", caKeyDERBlock)

		if caKeyDERBlock == nil || caKeyDERBlock.Type != "RSA PRIVATE KEY" {
			return nil, nil, errors.New("failed to read the CA key: unexpected content")
		}
		caKeyRaw, err = x509.ParsePKCS1PrivateKey(caKeyDERBlock.Bytes)
		if err != nil {
			return nil, nil, errors.New("failed to parse the CA key")
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, newCert, caCertRaw, &privkey.PublicKey, caKeyRaw)
	if err != nil {
		return nil, nil, err
	}

	// pem encode
	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, errors.New("failed to encode the certificate")
	}
	certString := certPEM.String()

	keyPEM := new(bytes.Buffer)
	err = pem.Encode(keyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privkey),
	})
	if err != nil {
		return nil, nil, errors.New("failed to encode the key")
	}

	keyString := keyPEM.String()

	return &keyString, &certString, nil
}

func randomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	return serialNumber, err
}
