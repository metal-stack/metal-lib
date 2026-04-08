package splunk

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/metal-stack/metal-lib/auditing/api"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
)

const (
	SplunkBackendName = "splunk"

	spunkIndexTimeout = 5 * time.Second
	// See on their docs in troubleshoot-http-event-collector (Possible_error_codes)
	splunkHealthyCode = 17
)

type (
	SplunkConfig struct {
		Endpoint   string
		HECToken   string
		SourceType string
		Index      string
		Host       string
		TlsConfig  *tls.Config
	}

	splunkAuditing struct {
		component    string
		log          *slog.Logger
		indexTimeout time.Duration

		client *http.Client

		endpoint   string
		hecToken   string
		sourceType string
		index      string
		host       string
	}

	SplunkEvent struct {
		// Time is the event time. The default time format is UNIX time format.
		Time int64 `json:"time,omitempty"`
		// Host value to assign to the event data. This key is typically the hostname of the client from which you're sending data.
		Host string `json:"host,omitempty"`
		// Source value to assign to the event data. For example, if you're sending data from an app you're developing, set this key to the name of the app.
		Source string `json:"source,omitempty"`
		// Sourcetype value to assign to the event data.
		SourceType string `json:"sourcetype,omitempty"`
		// Index by which the event data is to be indexed.
		Index string `json:"index,omitempty"`
		// Event is the actual event data in whatever format you want: a string, a number, another JSON object, and so on.
		Event api.Entry `json:"event,omitempty"`
	}

	splunkRequestEndpoint struct {
		path   string
		method string
		body   []byte
	}
)

// NewSplunk returns a new auditing backend for splunk. It supports the HTTP event collector interface.
func NewSplunk(c api.Config, sc SplunkConfig) (api.Auditing, error) {
	if c.Component == "" {
		component, err := api.DefaultComponent()
		if err != nil {
			return nil, err
		}

		c.Component = component
	}
	if c.IndexTimeout == 0 {
		c.IndexTimeout = spunkIndexTimeout
	}

	var (
		endpoint   = "http://localhost:8088"
		sourceType = "_json"
	)

	if sc.Endpoint != "" {
		endpoint = sc.Endpoint
	}

	if sc.HECToken == "" {
		return nil, fmt.Errorf("HEC token must be configured")
	}

	if sc.SourceType != "" {
		sourceType = sc.SourceType
	}

	if sc.Endpoint != "" {
		endpoint = sc.Endpoint
	}

	a := &splunkAuditing{
		component:    c.Component,
		log:          c.Log.WithGroup("auditing").With("audit-backend", SplunkBackendName),
		indexTimeout: c.IndexTimeout,
		client:       &http.Client{Transport: &http.Transport{TLSClientConfig: sc.TlsConfig}},
		endpoint:     endpoint,
		hecToken:     sc.HECToken,
		sourceType:   sourceType,
		index:        sc.Index,
		host:         sc.Host,
	}

	a.log.Info("initialized splunk client")

	return a, nil
}

func (a *splunkAuditing) Index(entry api.Entry) error {
	if entry.Timestamp.IsZero() {
		return errors.New("timestamp is not set")
	}

	splunkEvent := &SplunkEvent{
		Time:       entry.Timestamp.Unix(),
		Host:       a.host,
		Source:     a.component,
		SourceType: a.sourceType,
		Index:      a.index,
		Event:      entry,
	}

	e, err := json.Marshal(splunkEvent)
	if err != nil {
		return fmt.Errorf("error marshaling splunk event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.indexTimeout)
	defer cancel()

	_, err = a.splunkRequest(ctx, splunkRequestEndpoint{
		path:   "/services/collector",
		method: http.MethodPost,
		body:   e,
	})
	if err != nil {
		return fmt.Errorf("error indexing audit entry in splunk: %w", err)
	}

	return nil
}

func (a *splunkAuditing) Search(ctx context.Context, filter api.EntryFilter) ([]api.Entry, error) {
	return nil, fmt.Errorf("search not implemented for splunk audit backend")
}

func (a *splunkAuditing) ServiceName() string {
	return SplunkBackendName
}

func (a *splunkAuditing) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	resp, err := a.splunkRequest(ctx, splunkRequestEndpoint{
		path:   "/services/collector/health",
		method: http.MethodGet,
		body:   nil,
	})
	if err != nil {
		return healthstatus.HealthResult{}, fmt.Errorf("audit backend %q is unhealthy, collector is unhealthy: %w", SplunkBackendName, err)
	}

	type healthResp struct {
		Text string `json:"text"`
		Code int    `json:"code"`
	}

	health := healthResp{}

	if err := json.Unmarshal(resp, &health); err != nil {
		return healthstatus.HealthResult{}, fmt.Errorf("unable to unmarshal health response: %w", err)
	}

	if health.Code != splunkHealthyCode {
		return healthstatus.HealthResult{}, fmt.Errorf("audit backend %q is degraded: %s", SplunkBackendName, health.Text)
	}

	return healthstatus.HealthResult{
		Message: fmt.Sprintf("audit backend %q is healthy: %s", SplunkBackendName, health.Text),
		Status:  healthstatus.HealthStatusHealthy,
	}, nil
}

func (a *splunkAuditing) splunkRequest(ctx context.Context, ep splunkRequestEndpoint) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, a.indexTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, ep.method, a.endpoint+ep.path, bytes.NewBuffer(ep.body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Splunk "+a.hecToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during splunk request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if code := resp.StatusCode; code >= http.StatusBadRequest {
		return nil, fmt.Errorf("splunk endpoint %q did not return ok (%d)", ep.path, code)
	}

	return bytes, nil
}
