package vpn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

type ConnectOpt any

type connectOptOutputWriter struct {
	out io.Writer
}

func ConnectOptOutputWriter(out io.Writer) ConnectOpt {
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
		case connectOptOutputWriter:
			out = o.out
		default:
			return nil, fmt.Errorf("unknown connect opt: %T", opt)
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
		Ephemeral:  true,
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

	var firewallVPNIPs []netip.Addr

	err = retry.Do(
		func() error {
			_, _ = fmt.Fprintf(out, ".")
			status, err := lc.Status(ctx)
			if err != nil {
				return err
			}

			if status.BackendState != "Running" {
				_, _ = fmt.Fprintf(out, "backend is not yet running, but: %s\n", status.BackendState)
				return fmt.Errorf("backend state did not reach running, only %q", status.BackendState)
			}

			if status.Self.Online {
				for _, peer := range status.Peer {
					if strings.HasPrefix(peer.HostName, target) {
						firewallVPNIPs = peer.TailscaleIPs
						_, _ = fmt.Fprintf(out, " connected to %s (ips %s) took: %s\n", target, firewallVPNIPs, time.Since(start))

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

	// prefer ipv6 over ipv4 addresses for connection
	sort.SliceStable(firewallVPNIPs, func(i, j int) bool {
		if firewallVPNIPs[i].Is6() && firewallVPNIPs[j].Is4() {
			return true
		}
		return false
	})

	var errs []error

	for _, vpnIP := range firewallVPNIPs {
		vpn, err := func() (*vpn, error) {
			connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ip := vpnIP.String()

			_, _ = fmt.Fprintf(out, " attempting to ping %s\n", ip)

			_, err := lc.Ping(connectCtx, vpnIP, tailcfg.PingPeerAPI)
			if err != nil {
				return nil, fmt.Errorf("unable to ping ip %q: %w", ip, err)
			}

			var conn net.Conn

			err = retry.Do(
				func() error {
					dialCtx, cancel := context.WithTimeout(connectCtx, 10*time.Second)
					defer cancel()

					_, _ = fmt.Fprintf(out, " attempting to dial to %s\n", ip)

					conn, err = lc.DialTCP(dialCtx, ip, 22)
					if err != nil {
						return fmt.Errorf("unable to dial %q: %w", ip, err)
					}

					return nil
				},
				retry.Context(connectCtx),
			)
			if err != nil {
				return nil, err
			}

			return &vpn{Conn: conn, server: s, tempDir: tempDir, TargetIP: ip}, err
		}()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		return vpn, nil
	}

	if err := lc.Logout(context.Background()); err != nil {
		errs = append(errs, fmt.Errorf("unable to logout with tailscale server: %w", err))
	}

	return nil, fmt.Errorf("not able to dial to any of the found peer ips: %w", errors.Join(errs...))
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
