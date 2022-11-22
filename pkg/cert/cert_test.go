package cert

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func Test_GenerateCert(t *testing.T) {

	caKey, caCert, err := GenerateCert(CSR{
		CommonName:    "Test CA",
		Organization:  "Test Organization",
		LifetimeYears: 10,
		CertType:      CACert,
	}, nil, nil)
	if err != nil {
		panic(err)
	}

	caKey1, caCert1, err := GenerateCert(CSR{
		CommonName:    "Test CA1",
		Organization:  "Test Organization",
		LifetimeYears: 10,
		CertType:      CACert,
	}, nil, nil)
	if err != nil {
		panic(err)
	}

	clientKey, clientCert, err := GenerateCert(CSR{
		Organization:  "Test Organization",
		CommonName:    "test-client",
		DNSNames:      []string{"client.example.com"},
		LifetimeYears: 1,
		CertType:      ClientCert,
	}, caKey, caCert)
	if err != nil {
		panic(err)
	}

	serverKey, serverCert, err := GenerateCert(CSR{
		Organization:  "Test Organization",
		CommonName:    "test-server",
		DNSNames:      []string{"server.example.com"},
		LifetimeYears: 1,
		CertType:      ServerCert,
	}, caKey, caCert)
	if err != nil {
		panic(err)
	}

	selfSingedKey, selfSignedCert, err := GenerateCert(CSR{
		Organization:  "Test Organization",
		CommonName:    "test-server",
		DNSNames:      []string{"server.example.com"},
		LifetimeYears: 1,
		CertType:      SelfSignedCert,
	}, nil, nil)
	if err != nil {
		panic(err)
	}

	type args struct {
		key      *string
		cert     *string
		caKey    *string
		caCert   *string
		dnsName  string
		keyUsage x509.ExtKeyUsage
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "check that client cert is signed correctly",
			args: args{
				key:      clientKey,
				cert:     clientCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "client.example.com",
				keyUsage: x509.ExtKeyUsageClientAuth,
			},
			wantErr: false,
		},
		{
			name: "verify of client cert should fail for server usage",
			args: args{
				key:      clientKey,
				cert:     clientCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "client.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: true,
		},
		{
			name: "verify of client cert should fail for wrong dns name",
			args: args{
				key:      clientKey,
				cert:     clientCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "dummy.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: true,
		},
		{
			name: "verify of client cert should fail for wrong key",
			args: args{
				key:      serverKey,
				cert:     clientCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "client.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: true,
		},
		{
			name: "verify of client cert should fail wrong ca key",
			args: args{
				key:      clientKey,
				cert:     clientCert,
				caKey:    caKey1,
				caCert:   caCert,
				dnsName:  "client.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: true,
		},
		{
			name: "verify of client cert should fail wrong ca",
			args: args{
				key:      clientKey,
				cert:     clientCert,
				caKey:    caKey1,
				caCert:   caCert1,
				dnsName:  "client.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: true,
		},
		{
			name: "check that server cert is signed correctly",
			args: args{
				key:      serverKey,
				cert:     serverCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "server.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: false,
		},
		{
			name: "verify of server cert should fail for client usage",
			args: args{
				key:      serverKey,
				cert:     serverCert,
				caKey:    caKey,
				caCert:   caCert,
				dnsName:  "server.example.com",
				keyUsage: x509.ExtKeyUsageClientAuth,
			},
			wantErr: true,
		},
		{
			name: "check that self singed cert is signed correctly",
			args: args{
				key:      selfSingedKey,
				cert:     selfSignedCert,
				caKey:    selfSingedKey,
				caCert:   selfSignedCert,
				dnsName:  "server.example.com",
				keyUsage: x509.ExtKeyUsageServerAuth,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM([]byte(*tt.args.caCert))
			if !ok {
				t.Errorf("failed to parse CA certificate: %v", err)
				return
			}

			block, _ := pem.Decode([]byte(*tt.args.cert))
			if block == nil {
				t.Errorf("failed to parse certificate PEM: %v", err)
				return
			}
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Errorf("failed to parse certificate: %v", err)
				return
			}

			opts := x509.VerifyOptions{
				DNSName:   tt.args.dnsName,
				Roots:     roots,
				KeyUsages: []x509.ExtKeyUsage{tt.args.keyUsage},
			}

			_, err = c.Verify(opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
