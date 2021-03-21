package auth

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

const testCloudContextName = "cloudctl"
const testCloudContextNameDev = "cloudctl-dev"
const testCloudContextNameProd = "cloudctl-prod"

func Test_ExtractUsername(t *testing.T) {
	type tst struct {
		t    TokenInfo
		want string
	}
	tests := []tst{
		{
			t: TokenInfo{
				TokenClaims: Claims{
					Name:              "Erich",
					PreferredUsername: "",
					Roles:             nil,
				},
				IssuerConfig: IssuerConfig{},
			},
			want: "Erich",
		},
		{
			t: TokenInfo{
				TokenClaims: Claims{
					Name:              "Erich",
					PreferredUsername: "xyz123",
				},
				IssuerConfig: IssuerConfig{},
			},
			want: "xyz123",
		},
		{
			t: TokenInfo{
				TokenClaims: Claims{
					PreferredUsername: "xyz123",
				},
				IssuerConfig: IssuerConfig{},
			},
			want: "xyz123",
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ExtractName(tt.t))
	}
}

func Test_GetCurrentUser(t *testing.T) {

	tests := []test{
		{
			filename:    "./testdata/config",
			contextName: testCloudContextName,
			validate: expectSuccess(
				TestAuthContext{
					User:             "myUserId",
					Ctx:              testCloudContextName,
					AuthProviderName: "oidc",
					AuthProviderOidc: true,
					IDToken:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
				}),
		},
		{
			filename:    "./testdata/config-bare",
			contextName: testCloudContextName,
			validate: expectSuccess(
				TestAuthContext{
					User:             "myUserId",
					Ctx:              testCloudContextName,
					AuthProviderName: "oidc",
					AuthProviderOidc: true,
					IDToken:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
				}),
		},
		{
			filename:    "./testdata/config-bare-with-suffix",
			contextName: testCloudContextNameDev,
			validate: expectSuccess(
				TestAuthContext{
					User:             "myUserIdDev",
					Ctx:              testCloudContextNameDev,
					AuthProviderName: "oidc",
					AuthProviderOidc: true,
					IDToken:          "Dev-ID-Token",
				}),
		},
		{
			filename:    "./testdata/config-bare-with-suffix",
			contextName: testCloudContextNameProd,
			validate: expectSuccess(
				TestAuthContext{
					User:             "myUserId",
					Ctx:              testCloudContextNameProd,
					AuthProviderName: "oidc",
					AuthProviderOidc: true,
					IDToken:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
				}),
		},
		{
			filename:    "./testdata/config-no-oidc",
			contextName: testCloudContextName,
			validate:    expectError("missing key: auth-provider (path element idx: 1)"),
		},
		{
			filename:    "./testdata/config-notexists",
			contextName: testCloudContextName,
			validate:    expectError("error loading kube-config: stat ./testdata/config-notexists: no such file or directory"),
		},
		{
			filename:    "./testdata/config-empty",
			contextName: testCloudContextName,
			validate:    expectError("error loading kube-config - config is empty"),
		},
	}

	for _, currentTest := range tests {
		t.Run(currentTest.filename, func(t *testing.T) {

			authCtx, err := GetAuthContext(currentTest.filename, currentTest.contextName)
			validateErr := currentTest.validate(t, authCtx, err)
			if validateErr != nil {
				t.Errorf("test failed with unexpected error: %v", validateErr)
			}
		})
	}
}

type TestAuthContext AuthContext

func (tac *TestAuthContext) compare(t *testing.T, authCtx AuthContext) {
	assert.Equal(t, tac.User, authCtx.User)
	assert.Equal(t, tac.Ctx, authCtx.Ctx)
	assert.Equal(t, tac.AuthProviderName, authCtx.AuthProviderName)
	assert.Equal(t, tac.AuthProviderOidc, authCtx.AuthProviderOidc)
	assert.Equal(t, tac.IDToken, authCtx.IDToken)
}

type test struct {
	contextName string
	filename    string
	validate    validateFn
}

type validateFn func(t *testing.T, ctx AuthContext, err error) error

type successData struct {
	expected TestAuthContext
}

func expectSuccess(expected TestAuthContext) validateFn {
	s := successData{
		expected: expected,
	}

	return s.validateSuccess
}

func (s *successData) validateSuccess(t *testing.T, authCtx AuthContext, err error) error {

	if err != nil {
		return err
	}

	s.expected.compare(t, authCtx)

	return nil
}

func expectError(errorMsg string) validateFn {
	e := errorData{
		errorMessage: errorMsg,
	}

	return e.validateError
}

type errorData struct {
	errorMessage string
}

func (e *errorData) validateError(t *testing.T, ctx AuthContext, err error) error {

	if err == nil {
		return fmt.Errorf("expected error '%s', got none", e.errorMessage)
	}

	if err.Error() != e.errorMessage {
		return fmt.Errorf("expected error '%s', got '%s'", e.errorMessage, err.Error())
	}

	return nil
}

var demoToken = TokenInfo{
	IssuerConfig: IssuerConfig{
		ClientID:     "clientId_abcd",
		ClientSecret: "clientSecret_123123",
		IssuerURL:    "the_issuer",
		IssuerCA:     "/my/ca",
	},
	TokenClaims: Claims{
		Issuer:            "the_issuer",
		Subject:           "the_sub",
		EMail:             "email@provider.de",
		Name:              "user001",
		PreferredUsername: "",
		Roles:             nil,
	},
	IDToken:      "abcd4711",
	RefreshToken: "refresh234",
}

var demoToken2 = TokenInfo{
	IssuerConfig: IssuerConfig{
		ClientID:     "clientId_abcd",
		ClientSecret: "clientSecret_123123",
		IssuerURL:    "the_issuer",
		IssuerCA:     "/my/ca",
	},
	TokenClaims: Claims{
		Issuer:  "the_issuer",
		Subject: "the_sub",
		EMail:   "other-email@other-provider.de",
		Name:    "user002",
	},
	IDToken:      "cdefg",
	RefreshToken: "refresh987",
}

func TestUpdateUserNewFile(t *testing.T) {

	asserter := require.New(t)

	tmpfileName := filepath.Join(os.TempDir(), fmt.Sprintf("this_file_must_not_exist_%d", rand.Int63()))

	// delete file, just to be sure
	_ = os.Remove(tmpfileName)

	// "Update" -> create new file
	ti := demoToken
	_, err := UpdateKubeConfig(tmpfileName, ti, ExtractEMail)
	if err != nil {
		t.Fatalf("error updating kube-config: %v", err)
	}

	defer os.Remove(tmpfileName)

	// check it is written
	asserter.FileExists(tmpfileName, "expected file to exist")

	// check contents
	diffFiles(t, "./testdata/createdDemoConfig", tmpfileName)

	authContext, err := CurrentAuthContext(tmpfileName)
	if err != nil {
		t.Fatalf("error reading back user: %v", err)
	}

	asserter.Equal(authContext.User, demoToken.TokenClaims.EMail, "User")
	asserter.Equal(authContext.IDToken, demoToken.IDToken, "IDToken")
	asserter.Equal(authContext.AuthProviderName, "oidc", "AuthProvider")
	asserter.Equal(authContext.Ctx, testCloudContextName, "Context")
	asserter.Equal(authContext.ClientID, demoToken.ClientID, "ClientID")
	asserter.Equal(authContext.ClientSecret, demoToken.ClientSecret, "ClientSecret")
	asserter.Equal(authContext.IssuerURL, demoToken.IssuerURL, "Issuer")
	asserter.Equal(authContext.IssuerCA, demoToken.IssuerCA, "IssuerCA")

}

func TestUpdateUserWithNameExtractorNewFile(t *testing.T) {

	asserter := require.New(t)

	tmpfileName := filepath.Join(os.TempDir(), fmt.Sprintf("this_file_must_not_exist_%d", rand.Int63()))

	// delete file, just to be sure
	_ = os.Remove(tmpfileName)

	// "Update" -> create new file
	ti := demoToken
	_, err := UpdateKubeConfig(tmpfileName, ti, ExtractName)
	if err != nil {
		t.Fatalf("error updating kube-config: %v", err)
	}

	defer os.Remove(tmpfileName)

	// check it is written
	asserter.FileExists(tmpfileName, "expected file to ")

	// check contents
	diffFiles(t, "./testdata/createdDemoConfigName", tmpfileName)

	authContext, err := CurrentAuthContext(tmpfileName)
	if err != nil {
		t.Fatalf("error reading back user: %v", err)
	}

	asserter.Equal(authContext.User, demoToken.TokenClaims.Username(), "User")
	asserter.Equal(authContext.IDToken, demoToken.IDToken, "IDToken")
	asserter.Equal(authContext.ClientID, demoToken.ClientID, "ClientID")
	asserter.Equal(authContext.ClientSecret, demoToken.ClientSecret, "ClientSecret")
	asserter.Equal(authContext.IssuerURL, demoToken.IssuerURL, "Issuer")
	asserter.Equal(authContext.IssuerCA, demoToken.IssuerCA, "IssuerCA")
	asserter.Equal(authContext.AuthProviderName, "oidc", "AuthProvider")
	asserter.Equal(authContext.Ctx, testCloudContextName, "Context")
}

func TestLoadExistingConfigWithOIDC(t *testing.T) {

	authContext, err := CurrentAuthContext("./testdata/UEMCgivenConfig")

	require.NoError(t, err)

	require.Equal(t, authContext.User, demoToken.TokenClaims.EMail, "User")
	require.Equal(t, authContext.IDToken, demoToken.IDToken, "IDToken")
	require.Equal(t, authContext.ClientID, demoToken.ClientID, "ClientID")
	require.Equal(t, authContext.ClientSecret, demoToken.ClientSecret, "ClientSecret")
	require.Equal(t, authContext.IssuerURL, demoToken.IssuerURL, "Issuer")
	require.Equal(t, authContext.IssuerCA, demoToken.IssuerCA, "IssuerCA")
	require.Equal(t, authContext.AuthProviderName, "oidc", "AuthProvider")
	require.Equal(t, authContext.Ctx, testCloudContextName, "Context")
}

func TestUpdateUserExistingConfig(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEUgivenConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	_, err := UpdateKubeConfig(tmpfile.Name(), demoToken, ExtractEMail)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/UEUexpectedConfig", tmpfile.Name())
}

func TestUpdateIncompleteConfig(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/configIncomplete")
	defer os.Remove(tmpfile.Name()) // clean up

	_, err := UpdateKubeConfig(tmpfile.Name(), demoToken, ExtractEMail)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/configIncompleteExpected", tmpfile.Name())
}

func TestUpdateExistingCloudctlConfig(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEMCgivenConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	_, err := UpdateKubeConfig(tmpfile.Name(), demoToken2, ExtractEMail)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/UEMCexpectedConfig", tmpfile.Name())

	_, err = UpdateKubeConfig(tmpfile.Name(), demoToken2, ExtractEMail)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/UEMCexpectedConfig", tmpfile.Name())
}

func TestUpdateExistingProdConfig(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEMCgivenProdConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	_, err := UpdateKubeConfigContext(tmpfile.Name(), demoToken2, ExtractEMail, testCloudContextNameProd)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/UEMCexpectedProdConfig", tmpfile.Name())

	_, err = UpdateKubeConfigContext(tmpfile.Name(), demoToken2, ExtractEMail, testCloudContextNameProd)
	if err != nil {
		t.Fatalf("error updating config: %v", err)
	}

	diffFiles(t, "./testdata/UEMCexpectedProdConfig", tmpfile.Name())
}

func TestManipulateEncodeKubeconfig(t *testing.T) {

	// load full kubeconfig
	cfg, _, _, err := LoadKubeConfig("./testdata/UEUgivenConfig")
	require.NoError(t, err)

	err = AddUser(cfg, AuthContext{
		Ctx:              "user",
		User:             "username",
		AuthProviderName: "authprovider",
		AuthProviderOidc: true,
		IDToken:          "1234",
		RefreshToken:     "5678",
		IssuerConfig: IssuerConfig{
			ClientID:     "clientdId123",
			ClientSecret: "clientSecret345",
			IssuerURL:    "https://issuer",
			IssuerCA:     "/ca.cert",
		},
	})
	require.NoError(t, err)

	clusters, err := GetClusterNames(cfg)
	require.NoError(t, err)
	require.Equal(t, 1, len(clusters))

	err = AddContext(cfg, "myContext", clusters[0], "username")
	require.NoError(t, err)
	SetCurrentContext(cfg, "myContext")

	// encode result
	buf, err := EncodeKubeconfig(cfg)
	require.NoError(t, err)

	want, err := ioutil.ReadFile("./testdata/UEUManipulatedExpectedConfig")
	require.NoError(t, err)

	require.Empty(t, cmp.Diff(want, buf.Bytes()))
}

func TestReduceAndEncodeKubeconfig(t *testing.T) {

	// load full kubeconfig
	cfg, _, _, err := LoadKubeConfig("./testdata/UEMCgivenConfig")
	require.NoError(t, err)

	// create empty kubeconfig
	resultCfg := make(map[interface{}]interface{})
	err = CreateFromTemplate(&resultCfg)
	require.NoError(t, err)

	// copy over clusters only
	resultCfg["clusters"] = cfg["clusters"]

	// encode result
	buf, err := EncodeKubeconfig(resultCfg)
	require.NoError(t, err)

	want, err := ioutil.ReadFile("./testdata/UEUReducedExpectedConfig")
	require.NoError(t, err)

	require.Empty(t, cmp.Diff(want, buf.Bytes()))
}

func TestKubeconfigFromEnv(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEMCgivenConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	os.Setenv(RecommendedConfigPathEnvVar, tmpfile.Name())
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	_, filename, isDefault, err := LoadKubeConfig("")
	require.Nil(t, err)
	require.Equal(t, tmpfile.Name(), filename)
	require.False(t, isDefault)
}

func TestAuthContextFromEnv(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEMCgivenConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	os.Setenv(RecommendedConfigPathEnvVar, tmpfile.Name())
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	authCtx, err := GetAuthContext("", testCloudContextName)
	require.Nil(t, err)
	require.Equal(t, testCloudContextName, authCtx.Ctx)
	require.Equal(t, "email@provider.de", authCtx.User)
}

func TestKubeconfigDefault(t *testing.T) {

	// TODO we can't control the default location without mocking the fileaccess
	// it would be good to test the "path will be created if default location does not exist" feature
	_, _, isDefault, _ := LoadKubeConfig("")
	require.True(t, isDefault)
}

func TestKubeconfigFromEnvDoesNotExist(t *testing.T) {

	os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig")
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	authCtx, filename, isDefault, err := LoadKubeConfig("")
	require.Nil(t, err)
	require.Equal(t, "/tmp/path/to/kubeconfig", filename)
	require.NotNil(t, authCtx)
	require.False(t, isDefault)
}

func TestAuthContextFromEnvDoesNotExist(t *testing.T) {

	tmpfile := writeTemplate(t, "./testdata/UEMCgivenConfig")
	defer os.Remove(tmpfile.Name()) // clean up

	os.Setenv(RecommendedConfigPathEnvVar, tmpfile.Name())
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	_, err := CurrentAuthContext("")
	require.Nil(t, err)
}

func TestKubeconfigFromEnvMultiplePaths(t *testing.T) {

	os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/another/path")
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	_, filename, isDefault, err := LoadKubeConfig("")
	require.EqualError(t, err, "there are multiple files in env KUBECONFIG, don't know which one to update - please use cmdline-option")
	require.Equal(t, "", filename)
	require.False(t, isDefault)
}

func writeTemplate(t *testing.T, templateName string) (f *os.File) {
	tmpfile, err := ioutil.TempFile("", "test-template")
	if err != nil {
		t.Fatalf("error creating empty template: %v", err)
	}

	var template []byte
	template, err = ioutil.ReadFile(templateName)
	if err != nil {
		t.Fatalf("error reading template: %v", err)
	}

	if _, err := tmpfile.Write(template); err != nil {
		t.Fatalf("error writing template: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("error closing template: %v", err)
	}

	return tmpfile
}

// diff given file contents and report diff-errors as t.Error
func diffFiles(t *testing.T, expectedFileName string, gotFileName string) {

	var err error

	var gotBytes []byte
	gotBytes, err = ioutil.ReadFile(gotFileName)
	if err != nil {
		t.Fatalf("error reading created file: %v", err)
	}

	var expectedBytes []byte
	expectedBytes, err = ioutil.ReadFile(expectedFileName)
	if err != nil {
		t.Fatalf("error reading expected data file: %v", err)
	}

	if diff := cmp.Diff(expectedBytes, gotBytes); diff != "" {
		t.Errorf("output differs (-want +got)\n%s", diff)
		t.Log(string(gotBytes))
	}
}
