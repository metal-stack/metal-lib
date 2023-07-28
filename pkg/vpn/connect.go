package vpn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"tailscale.com/tsnet"
)

type ConnectOpt interface{}

type connectOptOutputWriter struct {
	out io.Writer
}

func ConnectionOptOutputWriter(out io.Writer) ConnectOpt {
	return connectOptOutputWriter{out: out}
}

type vpn struct {
	net.Conn
	server   *tsnet.Server
	tempDir  string
	TargetIP string
}

// Connect to the given target host with tailscale, controllerURL specifies the URL where the coordination server lives
// authKey is the key to authenticate to the vpn.
func Connect(ctx context.Context, target, controllerURL, authkey string, opts ...ConnectOpt) (*vpn, error) {
	var out io.Writer
	out = os.Stdout

	for _, opt := range opts {
		switch o := opt.(type) {
		case *connectOptOutputWriter:
			out = o.out
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	randomSuffix, _, _ := strings.Cut(uuid.NewString(), "-")
	hostname = fmt.Sprintf("%s-%s", hostname, randomSuffix)
	tempDir, err := os.MkdirTemp("", hostname)
	if err != nil {
		return nil, err
	}
	s := &tsnet.Server{
		Hostname:   hostname,
		ControlURL: controllerURL,
		AuthKey:    authkey,
		Dir:        tempDir,
	}

	// now disable logging, maybe altogether later
	if os.Getenv("DEBUG") == "" {
		s.Logf = func(format string, args ...any) {}
	}

	start := time.Now()
	lc, err := s.LocalClient()
	if err != nil {
		return nil, err
	}
	var firewallVPNIP netip.Addr
	err = retry.Do(
		func() error {
			fmt.Fprintf(out, ".")
			status, err := lc.Status(ctx)
			if err != nil {
				return err
			}
			if status.Self.Online {
				for _, peer := range status.Peer {
					if strings.HasPrefix(peer.HostName, target) {
						firewallVPNIP = peer.TailscaleIPs[0]
						fmt.Fprintf(out, " connected to %s (ip %s) took: %s\n", target, firewallVPNIP, time.Since(start))
						return nil
					}
				}
			}
			return fmt.Errorf("did not get online")
		},
		retry.Attempts(50),
	)
	if err != nil {
		return nil, err
	}
	// disable logging after successful connect
	s.Logf = func(format string, args ...any) {}

	conn, err := lc.DialTCP(ctx, firewallVPNIP.String(), 22)
	return &vpn{Conn: conn, server: s, tempDir: tempDir, TargetIP: firewallVPNIP.String()}, err
}

// Close all open connections after vpn was used.
func (v *vpn) Close() error {
	var errs []error

	err := v.server.Close()
	if err != nil {
		errs = append(errs, err)
	}
	err = os.RemoveAll(v.tempDir)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
