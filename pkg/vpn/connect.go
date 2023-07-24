package vpn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"tailscale.com/tsnet"
)

type VPN struct {
	Conn     net.Conn
	server   *tsnet.Server
	tempDir  string
	TargetIP string
}

func Connect(target, controllerURL, authkey string) (*VPN, error) {
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
	ctx := context.Background()
	var firewallVPNIP netip.Addr
	err = retry.Do(
		func() error {
			fmt.Printf(".")
			status, err := lc.Status(ctx)
			if err != nil {
				return err
			}
			if status.Self.Online {
				for _, peer := range status.Peer {
					if strings.HasPrefix(peer.HostName, target) {
						firewallVPNIP = peer.TailscaleIPs[0]
						fmt.Printf(" connected to %s (ip %s) took: %s\n", target, firewallVPNIP, time.Since(start))
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
	return &VPN{Conn: conn, server: s, tempDir: tempDir, TargetIP: firewallVPNIP.String()}, err
}

func (v *VPN) Close() error {
	var errs []error

	errs = append(errs, v.server.Close())
	errs = append(errs, os.RemoveAll(v.tempDir))

	return errors.Join(errs...)
}
