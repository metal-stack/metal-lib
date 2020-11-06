package sign

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	// Generate new KeyPair
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	publicKey := &privateKey.PublicKey

	// Encode as PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(publicKey),
	})

	// Decode PEM
	privateKey, err = DecodePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		panic(err)
	}

	publicKey, err = DecodePublicKey([]byte(publicKeyPEM))
	if err != nil {
		panic(err)
	}

	data := []byte("test")

	type args struct {
		privateKey       *rsa.PrivateKey
		publicKey        *rsa.PublicKey
		dataSigning      []byte
		dataVerification []byte
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "check that signing and verification works if data is the same on both sides",
			args: args{
				privateKey:       privateKey,
				publicKey:        publicKey,
				dataSigning:      data,
				dataVerification: data,
			},
			want: true,
		},
		{
			name: "verification should fail if data was tampered",
			args: args{
				privateKey:       privateKey,
				publicKey:        publicKey,
				dataSigning:      data,
				dataVerification: []byte("tampered data"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature, err := Sign(tt.args.privateKey, tt.args.dataSigning)
			if err != nil {
				t.Errorf("signing failed: %w", err)
				return
			}

			got, err := VerifySignature(tt.args.publicKey, signature, tt.args.dataVerification)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}
