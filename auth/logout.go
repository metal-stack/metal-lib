package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

type LogoutParams struct {
	IssuerURL string
	Logger    *zap.SugaredLogger
}

func (l *LogoutParams) Validate() error {
	if l.IssuerURL == "" {
		return errors.New("error validating config: IssuerURL is required")
	}
	if l.Logger == nil {
		return errors.New("error validating config: Logger is required")
	}

	return nil
}

func Logout(params *LogoutParams) error {
	err := params.Validate()
	if err != nil {
		return err
	}

	var (
		log          = params.Logger
		completeChan = make(chan bool)

		wellKnownURL    = strings.TrimSuffix(params.IssuerURL, "/") + "/.well-known/openid-configuration"
		wellKnownConfig struct {
			EndSessionEndpoint string `json:"end_session_endpoint"`
		}

		mux     = http.NewServeMux()
		handler = logoutHandler{
			log:          log,
			completeChan: completeChan,
		}
	)

	mux.HandleFunc("/", handler.handleLogout)

	err = fetchJSON(wellKnownURL, &wellKnownConfig)
	if err != nil {
		return err
	}

	if wellKnownConfig.EndSessionEndpoint == "" {
		return fmt.Errorf("no endpoint for ending oidc session discovered, unsupported by oidc provider")
	}

	endSessionURL, err := url.Parse(wellKnownConfig.EndSessionEndpoint)
	if err != nil {
		return fmt.Errorf("cannot parse end session url: %w", err)
	}

	listener, listenAddr, err := newRandomPortListener()
	if err != nil {
		return err
	}

	values := endSessionURL.Query()
	values.Add("redirect_uri", listenAddr)
	endSessionURL.RawQuery = values.Encode()

	server := &http.Server{
		Handler: mux,
	}

	log.Debug("Opening Browser for Logout")
	fmt.Printf("Opening Browser for Authentication. If this does not work, please point your browser to %s\n", listenAddr)

	go func() {
		log.Debugw("opening browser", "addr", listenAddr, "end-session-url", endSessionURL.String())
		err := openBrowser(endSessionURL.String())
		if err != nil {
			log.Errorw("open browser", "error", err)
		}
	}()
	go func() {
		<-completeChan
		err = server.Shutdown(context.Background())
		if err != nil {
			log.Errorw("shutdown", "error", err)
		}
	}()

	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

type logoutHandler struct {
	log          *zap.SugaredLogger
	completeChan chan bool
}

func (l *logoutHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	logoutPage := `<html>
<head>` + commonStyle + `
</head>
<body>
	<h3>Logout successful</h3>
	<h4>OIDC session successfully logged out. Token is not revoked and is valid until expiration.</h4>
</body>
</html>`
	_, err := w.Write([]byte(logoutPage))
	if err != nil {
		l.log.Debug("logout failed")
	} else {
		l.log.Debug("logout succeeded")
	}

	l.completeChan <- true
}
