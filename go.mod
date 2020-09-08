module github.com/metal-stack/metal-lib

go 1.14

require (
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful-openapi/v2 v2.2.1
	github.com/emicklei/go-restful/v3 v3.3.1
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2
	github.com/icza/dyno v0.0.0-20200205103839-49cb13720835
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/metal-stack/security v0.3.0
	github.com/metal-stack/v v1.0.2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nsqio/go-nsq v1.0.8
	github.com/nsqio/nsq v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/pretty v1.0.2 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)

replace github.com/metal-stack/security => ../security
