package auth

import (
	"bytes"
	"fmt"
	"github.com/icza/dyno"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
)

//
// kubeconfig provides methods to read and write kuberentes kubectl-config-files.
// It must not depend on the k8s.io-client due to the dependency-hell surrounding it.
// So we tried to be generic and structure agnostic, so that we can read a config,
// modify just the parts we need and write it back and do not loose anything,
// we don't want to know and care about.
// It would be much easier to un/marshall structs, but then we would be in danger
// of loosing fields that we don't have in our version.
//

const (
	cloudContextName = "cloudctl"
	oidcAuthProvider = "oidc"
)

// UserIDExtractor extractor to make the source of the "userid" customizable
type UserIDExtractor func(tokenInfo TokenInfo) string

func ExtractName(tokenInfo TokenInfo) string {
	return tokenInfo.TokenClaims.Name
}

func ExtractEMail(tokenInfo TokenInfo) string {
	return tokenInfo.TokenClaims.Email
}

// UpdateKubeConfig saves the given tokenInfo in the kubeConfig. The given path to kubeconfig is preferred,
// otherwise the location of the kubeconfig is determined from env KUBECONFIG or default location.
//
// we modify/append a user with auth-provider from given tokenInfo.
// we modify/append a context with name metalctl that references the user.
//
// returns filename the config got written to or error if any
//
func UpdateKubeConfig(kubeConfig string, tokenInfo TokenInfo, userIDExtractor UserIDExtractor) (string, error) {

	if userIDExtractor == nil {
		return "", errors.New("userIdExtractor must not be nil")
	}

	cfg, outputFilename, isDefault, err := LoadKubeConfig(kubeConfig)
	if err != nil {
		// file does not exist, we create it from scratch
		outputFilename = kubeConfig
		err = CreateFromTemplate(&cfg)
		if err != nil {
			return "", err
		}
	}

	usersSlice, err := dyno.GetSlice(cfg, "users")
	if err != nil {
		return "", err
	}
	if usersSlice == nil {
		return "", errors.New("users slice not found")
	}

	userName := userIDExtractor(tokenInfo)

	config := map[string]string{
		"client-id":                 tokenInfo.ClientID,
		"client-secret":             tokenInfo.ClientSecret,
		"id-token":                  tokenInfo.IDToken,
		"refresh-token":             tokenInfo.RefreshToken,
		"idp-issuer-url":            tokenInfo.TokenClaims.Iss,
		"idp-certificate-authority": tokenInfo.IssuerCA,
	}

	err = AddUserConfigMap(cfg, userName, config)
	if err != nil {
		return "", err
	}

	err = AddContext(cfg, cloudContextName, "", userName)
	if err != nil {
		return "", err
	}

	// use configured yaml
	yamlBytes, err := EncodeKubeconfig(cfg)
	if err != nil {
		return "", err
	}

	// if the location of the kubeconfig is not specified explicitly, we create the default path
	if isDefault {
		err = ensureDirectory(outputFilename)
		if err != nil {
			return "", err
		}
	}

	err = ioutil.WriteFile(outputFilename, yamlBytes.Bytes(), 0600)
	if err != nil {
		return "", err
	}

	return outputFilename, nil
}

//AddUserConfigMap adds the given user-auth-configMap to the kubecfg or replaces an already existing user
func AddUserConfigMap(kubecfg map[interface{}]interface{}, userName string, configMap map[string]string) error {

	authProviderMap := map[string]interface{}{
		"config": configMap,
		"name":   "oidc",
	}

	userMap := map[string]interface{}{
		"auth-provider": authProviderMap,
	}

	user := map[string]interface{}{
		"name": userName,
		"user": userMap,
	}

	usersSlice, err := dyno.GetSlice(kubecfg, "users")
	if err != nil {
		return err
	}
	if usersSlice == nil {
		return errors.New("users slice not found")
	}

	// check if user already exists
	_, index, err := findMapListMap(kubecfg, "users", "name", userName)
	if err != nil {
		// not found, append
		err = dyno.Append(kubecfg, user, "users")
		if err != nil {
			return err
		}
	} else {
		// replace user
		usersSlice[index] = user
	}
	return nil
}

//AddUser adds the given user-authconfig to the kubecfg or replaces an already existing user
func AddUser(kubecfg map[interface{}]interface{}, authCtx AuthContext) error {

	userName := authCtx.User

	configMap := map[string]string{
		"client-id":     authCtx.ClientID,
		"client-secret": authCtx.ClientSecret,
		"id-token":      authCtx.IDToken,
		//		"refresh-token":             authCtx.RefreshToken,
		"idp-issuer-url":            authCtx.IssuerURL,
		"idp-certificate-authority": authCtx.IssuerCA,
	}

	return AddUserConfigMap(kubecfg, userName, configMap)
}

//AddContext adds or replaces the given context with given clusterName and userName.
func AddContext(cfg map[interface{}]interface{}, contextName string, clusterName string, userName string) error {
	type Context struct {
		Context map[string]interface{}
		Name    string
	}

	// check & create context
	ctxData := make(map[string]interface{})
	ctxData["cluster"] = clusterName
	ctxData["user"] = userName

	context := Context{
		Name:    contextName,
		Context: ctxData,
	}

	//check if "contexts" exists
	_, err := dyno.Get(cfg, "contexts")
	if err != nil {
		// not found, create contexts completely
		cfg["contexts"] = []Context{
			context,
		}
	} else {
		// "contexts" exist, now find named context within "contexts"
		_, index, err := findMapListMap(cfg, "contexts", "name", contextName)
		if err != nil {
			// context "metalctl" not found
			err = dyno.Append(cfg, context, "contexts")
			if err != nil {
				return err
			}
		} else {
			// update context "metalctl"
			ctxList, _ := dyno.GetSlice(cfg, "contexts")
			ctxList[index] = context
		}
	}

	return nil
}

//SetCurrentContext sets the current context to the given name
func SetCurrentContext(cfg map[interface{}]interface{}, contextName string) {
	cfg["current-context"] = contextName
}

//GetClusterNames returns all clusternames
func GetClusterNames(cfg map[interface{}]interface{}) ([]string, error) {

	clusterNames := []string{}
	clusters, err := dyno.GetSlice(cfg, "clusters")
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		m, err := dyno.GetMapS(clusters[i])
		if err != nil {
			return nil, err
		}
		cn, ok := m["name"].(string)
		if ok && cn != "" {
			clusterNames = append(clusterNames, cn)
		}
	}
	return clusterNames, nil
}

//EncodeKubeconfig serializes the given kubeconfig
func EncodeKubeconfig(kubeconfig map[interface{}]interface{}) (bytes.Buffer, error) {
	var yamlBytes bytes.Buffer
	e := yaml.NewEncoder(&yamlBytes)
	e.SetIndent(2)
	err := e.Encode(&kubeconfig)
	return yamlBytes, err
}

// ensureDirectory checks all directories in fqFile exist and creates if necessary
func ensureDirectory(fqFile string) error {
	kcPath := path.Dir(fqFile)
	if _, err := os.Stat(kcPath); os.IsNotExist(err) {
		return os.MkdirAll(kcPath, 0700)
	}
	return nil
}

//AuthContext models the data in the kubeconfig user/auth-provider/config/oidc-config-map
type AuthContext struct {
	// Name of the context for metalctl auth
	Ctx string
	// Name of the user in the active context
	User string
	// Name of the authProvider in the active context
	AuthProviderName string
	// Flag if the AuthProvider is oidc, i.e. valid for our usecases
	AuthProviderOidc bool

	// IDToken, only if AuthProviderOidc is true
	IDToken string

	// RefreshToken
	RefreshToken string

	IssuerConfig
}

// finds the listKey from the given map, gets the list of maps, finds the map with matchKey == matchValue, returns map, index, error
func findMapListMap(cfg map[interface{}]interface{}, listKey string, matchKey string, matchValue string) (map[string]interface{}, int, error) {

	ctxSlice, err := dyno.GetSlice(cfg, listKey)
	if err != nil {
		return nil, 0, err
	}

	for i := 0; i < len(ctxSlice); i++ {
		currentContextItemMap, err := dyno.GetMapS(ctxSlice[i])
		if err != nil {
			break
		}

		if currentContextItemMap[matchKey] == matchValue {
			return currentContextItemMap, i, nil
		}
	}

	return nil, 0, errors.Errorf("no %s, %s=%s found", listKey, matchKey, matchValue)
}

// determines the current context and user
func CurrentAuthContext(kubeConfig string) (AuthContext, error) {

	empty := AuthContext{}

	cfg, _, _, err := LoadKubeConfig(kubeConfig)
	if err != nil {
		return empty, err
	}

	// get context "metalctl" to determine user

	cloudContext, _, err := findMapListMap(cfg, "contexts", "name", cloudContextName)
	if err != nil {
		return empty, err
	}
	if cloudContext == nil {
		return empty, errors.Errorf("cannot determine user from kube-config, context '%s' does not exist", cloudContextName)
	}
	empty.Ctx = cloudContextName

	// determine username from context
	contextMap, err := dyno.GetMapS(cloudContext, "context")
	if err != nil {
		return empty, err
	}
	userName := fmt.Sprintf("%v", contextMap["user"])
	empty.User = userName

	// get user
	userMap, _, err := findMapListMap(cfg, "users", "name", userName)
	if err != nil {
		return empty, err
	}

	authProviderMap, err := dyno.GetMapS(userMap, "user", "auth-provider")
	if err != nil {
		return empty, err
	}

	// read auth-data
	authProviderName, err := dyno.GetString(authProviderMap, "name")
	if err != nil {
		return empty, err
	}

	isOidc := authProviderName == oidcAuthProvider
	if isOidc {
		token, err := dyno.GetString(authProviderMap, "config", "id-token")
		if err != nil {
			return empty, err
		}
		issuerURL, err := dyno.GetString(authProviderMap, "config", "idp-issuer-url")
		if err != nil {
			return empty, err
		}
		issuerCA, err := dyno.GetString(authProviderMap, "config", "idp-certificate-authority")
		if err != nil {
			return empty, err
		}
		clientId, err := dyno.GetString(authProviderMap, "config", "client-id")
		if err != nil {
			return empty, err
		}
		clientSecret, err := dyno.GetString(authProviderMap, "config", "client-secret")
		if err != nil {
			return empty, err
		}

		return AuthContext{
			Ctx:              cloudContextName,
			User:             userName,
			AuthProviderName: authProviderName,
			AuthProviderOidc: isOidc,
			IDToken:          token,

			IssuerConfig: IssuerConfig{
				IssuerURL:    issuerURL,
				IssuerCA:     issuerCA,
				ClientID:     clientId,
				ClientSecret: clientSecret,
			},
		}, nil
	}

	return empty, errors.New("cannot determine user from kube-config, no current context set")
}

// LoadKubeConfig loads the kube-config from the given location, if kubeConfig is "" the default location will be used.
// If kubeconfig is explicitly given and no file exists at the location, an error is returned.
// If the default location is used and no file exists, the contents of the kubeconfigTemplate are returned.
// returns map, filename, isDefaultLocation and error
func LoadKubeConfig(kubeConfig string) (content map[interface{}]interface{}, filename string, isDefaultLocation bool, e error) {

	var err error
	var cfg map[interface{}]interface{}
	var outputFilename string

	isDefault := false

	if kubeConfig != "" {
		if _, err = os.Stat(kubeConfig); os.IsNotExist(err) {
			// no file, use default
			return nil, "", false, errors.Wrap(err, "error loading kube-config")
		} else {
			// read exactly the specified file
			cfg, err = readFile(kubeConfig)
			if err != nil {
				return nil, "", false, err
			}
			if len(cfg) == 0 {
				return nil, "", false, errors.New("error loading kube-config - config is empty")
			}
		}

		outputFilename = kubeConfig

	} else {
		// try path from env
		envPaths := fromEnv()
		if len(envPaths) > 1 {
			return nil, "", false, errors.Errorf("there are multiple files in env %s, don't know which one to update - please use cmdline-option", RecommendedConfigPathEnvVar)
		}

		if len(envPaths) == 1 {
			filename := envPaths[0]

			if _, err = os.Stat(filename); os.IsNotExist(err) {
				// no file, use default
				err = nil
			} else {
				// read exactly the specified file
				cfg, err = readFile(filename)
				if err != nil {
					return nil, "", false, err
				}
			}

			outputFilename = filename

		} else {
			// use default location
			filename := RecommendedHomeFile
			isDefault = true

			if _, err = os.Stat(filename); os.IsNotExist(err) {
				// no file, use default
				err = nil
			} else {
				// read exactly the specified file
				cfg, err = readFile(filename)
				if err != nil {
					return nil, "", isDefault, err
				}
			}

			outputFilename = filename
		}
	}

	if len(cfg) == 0 {
		err = CreateFromTemplate(&cfg)
		if err != nil {
			return nil, "", isDefault, err
		}
	}

	return cfg, outputFilename, isDefault, err
}

// reads the given yaml-file and unmarshalls the contents (top level map)
// if the file does not exits or if the file is empty, an empty map is returned
func readFile(filename string) (map[interface{}]interface{}, error) {
	// we expect a map top level
	cfg := make(map[interface{}]interface{})

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s", filename)
	}

	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "error un-marshalling %s", filename)
	}

	return cfg, err
}

// minimal kube config
const kubeconfigTemplate = `apiVersion: v1
kind: Config
clusters: []
contexts: []
current-context: ""
preferences: {}
users: []
`

// CreateFromTemplate returns a minimal kubeconfig
func CreateFromTemplate(cfg *map[interface{}]interface{}) error {

	return yaml.Unmarshal([]byte(kubeconfigTemplate), cfg)
}
