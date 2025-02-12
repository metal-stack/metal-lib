package auditing

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
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
		component string
		log       *slog.Logger

		async        bool
		asyncRetry   int
		asyncBackoff time.Duration
		asyncTimeout time.Duration

		client *http.Client

		endpoint   string
		hecToken   string
		sourceType string
		index      string
		host       string
	}

	splunkEvent struct {
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
		Event Entry `json:"event,omitempty"`
	}
)

// NewSplunk returns a new auditing backend for splunk. It supports the HTTP event collector interface.
func NewSplunk(c Config, sc SplunkConfig) (Auditing, error) {
	if c.Component == "" {
		component, err := defaultComponent()
		if err != nil {
			return nil, err
		}

		c.Component = component
	}

	var (
		endpoint     = "http://localhost:8088"
		sourceType   = "_json"
		asyncBackoff = 200 * time.Millisecond
		asyncTimeout = 5 * time.Second
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

	if c.Async && c.AsyncTimeout > 0 {
		asyncTimeout = c.AsyncTimeout
	}

	if c.Async && c.AsyncBackoff > 0 {
		asyncBackoff = c.AsyncBackoff
	}

	a := &splunkAuditing{
		component:    c.Component,
		log:          c.Log.WithGroup("auditing"),
		async:        c.Async,
		asyncRetry:   c.AsyncRetry,
		asyncBackoff: asyncBackoff,
		asyncTimeout: asyncTimeout,
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

func (a *splunkAuditing) Flush() error {
	return nil
}

func (a *splunkAuditing) Index(entry Entry) error {
	if entry.Timestamp.IsZero() {
		return errors.New("timestamp is not set")
	}

	splunkEvent := &splunkEvent{
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

	if !a.async {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/services/collector", bytes.NewBuffer(e))
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", "Splunk "+a.hecToken)

		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("error indexing audit entry in splunk: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}

	go func() {
		count := 0

		for {
			if count > a.asyncRetry {
				a.log.Error("maximum amount of retries reached for sending event to splunk, giving up", "retries", a.asyncRetry, "entry-id", entry.Id)
				return
			}

			count++

			ctx, cancel := context.WithTimeout(context.Background(), a.asyncTimeout)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/services/collector", bytes.NewBuffer(e))
			if err != nil {
				cancel()
				a.log.Error("error creating request", "error", err)
				return
			}

			req.Header.Add("Authorization", "Splunk "+a.hecToken)

			resp, err := a.client.Do(req)
			cancel()
			if err != nil {
				a.log.Error("error indexing audit entry in splunk, retrying in 200ms", "error", err)
				time.Sleep(a.asyncBackoff)
				continue
			}
			defer resp.Body.Close()

			return
		}
	}()

	return nil
}

func (a *splunkAuditing) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	return nil, fmt.Errorf("search not implemented for splunk audit backend")
}
