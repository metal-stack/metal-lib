package auth

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

const testCloudContextName123 = "ctx123"

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

func Test_NewUpdateKubeConfigHandler(t *testing.T) {
	tokenInfo := TokenInfo{
		IDToken:      "123",
		RefreshToken: "456",
		TokenClaims:  Claims{},
		IssuerConfig: IssuerConfig{},
	}

	file, err := ioutil.TempFile("", "prefix")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	var b bytes.Buffer
	thf := NewUpdateKubeConfigHandler(file.Name(), &b)
	err = thf(tokenInfo)
	assert.NoError(t, err)

	_, err = GetAuthContext(file.Name(), "xyz")
	assert.EqualError(t, err, "no contexts, name=xyz found")

	authCtx, err := GetAuthContext(file.Name(), cloudContext)
	assert.NoError(t, err)
	assert.Equal(t, "123", authCtx.IDToken)
}

func Test_NewUpdateKubeConfigHandlerWithContext(t *testing.T) {
	tokenInfo := TokenInfo{
		IDToken:      "123",
		RefreshToken: "456",
		TokenClaims:  Claims{},
		IssuerConfig: IssuerConfig{},
	}

	file, err := ioutil.TempFile("", "prefix")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	var b bytes.Buffer
	thf := NewUpdateKubeConfigHandler(file.Name(), &b, WithContextName("ctx123"))
	err = thf(tokenInfo)
	assert.NoError(t, err)

	_, err = GetAuthContext(file.Name(), "cloudctl-xyz")
	assert.EqualError(t, err, "no contexts, name=cloudctl-xyz found")

	authCtx, err := GetAuthContext(file.Name(), testCloudContextName123)
	assert.NoError(t, err)
	assert.Equal(t, "123", authCtx.IDToken)
}
