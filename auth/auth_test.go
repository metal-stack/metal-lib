package auth

import (
	"testing"
)

func Test_IssuerValidation(t *testing.T) {
	err := OIDCFlow(Config{})
	if err == nil {
		t.Fatal("Expected err")
	}
}

func Test_ClientIdValidation(t *testing.T) {
	err := OIDCFlow(Config{
		IssuerURL: "https://dex:4711",
	})
	if err == nil {
		t.Fatal("Expected err")
	}
}

func Test_ClientSecretValidation(t *testing.T) {
	err := OIDCFlow(Config{
		IssuerURL: "https://dex:4711",
		ClientID:  "123",
	})
	if err == nil {
		t.Fatal("Expected err")
	}
}

func Test_TokenHandlerValidation(t *testing.T) {
	err := OIDCFlow(Config{
		IssuerURL:    "https://dex:4711",
		ClientID:     "123",
		ClientSecret: "231",
	})
	if err == nil {
		t.Fatal("Expected err")
	}
}
