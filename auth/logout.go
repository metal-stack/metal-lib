package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

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

func Logout(config *LogoutParams) error {
	err := config.Validate()
	if err != nil {
		return err
	}

	var (
		log          = config.Logger
		completeChan = make(chan bool)

		wellKnownURL    = strings.TrimSuffix(config.IssuerURL, "/") + "/.well-known/openid-configuration"
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
		return fmt.Errorf("no end session endpoint discovered, unsupported by oidc provider")
	}

	endSessionURL, err := url.Parse(wellKnownConfig.EndSessionEndpoint)
	if err != nil {
		return fmt.Errorf("cannot parse end session url: %w", err)
	}

	// use next free port for callback
	/* #nosec */
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listenAddr := fmt.Sprintf("http://localhost:%d", port)

	values := endSessionURL.Query()
	values.Add("redirect_uri", listenAddr)
	endSessionURL.RawQuery = values.Encode()

	server := &http.Server{
		Handler: mux,
	}

	log.Debug("Opening Browser for Logout")
	fmt.Printf("Opening Browser for Authentication. If this does not work, please point your browser to %s\n", listenAddr)

	go func() {
		log.Debugw("opening browser", "addr", listenAddr, "port", port, "end-session-url", endSessionURL.String())
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

func fetchJSON(url string, data any) error {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "metal-lib")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching url: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("retrieved bad status code (%s): %s", resp.Status, body)
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}

	return nil
}
