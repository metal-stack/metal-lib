package ssh

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type ConnectOpt any

type connectOptOutputWriter struct {
	out io.Writer
}

func ConnectOptOutputWriter(out io.Writer) ConnectOpt {
	return connectOptOutputWriter{out: out}
}

type Client struct {
	*ssh.Client
	out io.Writer
}
type Env map[string]string

// NewClientWithConnection connects via ssh to host with the given user and authenticates with the privateKey.
// a already created net.Conn must be provided.
// see vpn.Connect howto create such a connection via tailscale VPN
//
// Call client.Connect() to actually get the ssh session
func NewClientWithConnection(user, host string, privateKey []byte, conn net.Conn, opts ...ConnectOpt) (*Client, error) {
	sshConfig, err := getSSHConfig(user, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

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

	sshConn, sshChan, req, err := ssh.NewClientConn(conn, host, sshConfig)
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(sshConn, sshChan, req)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: client,
		out:    out,
	}, nil
}

// NewClient connects via ssh to host with the given user and authenticates with the privateKey.
//
// Call client.Connect() to actually get the ssh session
func NewClient(user, host string, privateKey []byte, port int, opts ...ConnectOpt) (*Client, error) {
	var out io.Writer
	out = os.Stdout

	for _, opt := range opts {
		switch o := opt.(type) {
		case *connectOptOutputWriter:
			out = o.out
		}
	}

	fmt.Fprintf(out, "ssh to %s@%s:%d\n", user, host, port)

	sshConfig, err := getSSHConfig(user, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	sshServerAddress := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", sshServerAddress, sshConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: client,
		out:    out,
	}, nil
}

// Connect once a ssh.Client was created, you can connect to it, this call blocks until session is terminated.
func (c *Client) Connect(env *Env) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if env != nil {
		var errs []error
		for key, value := range *env {
			err := session.Setenv(key, value)
			if err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin
	// Set up terminal modes
	// https://net-ssh.github.io/net-ssh/classes/Net/SSH/Connection/Term.html
	// https://www.ietf.org/rfc/rfc4254.txt
	// https://godoc.org/golang.org/x/crypto/ssh
	// THIS IS THE TITLE
	// https://pythonhosted.org/ANSIColors-balises/ANSIColors.html
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,      // enable echoing
		ssh.TTY_OP_ISPEED: 115200, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 115200, // output speed = 14.4kbaud
	}

	fileDescriptor := int(os.Stdin.Fd())

	if term.IsTerminal(fileDescriptor) {
		originalState, err := term.MakeRaw(fileDescriptor)
		if err != nil {
			return err
		}
		defer func() {
			err = term.Restore(fileDescriptor, originalState)
			if err != nil {
				fmt.Fprintf(c.out, "error restoring ssh terminal:%v\n", err)
			}
		}()

		termWidth, termHeight, err := term.GetSize(fileDescriptor)
		if err != nil {
			return err
		}

		err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
		if err != nil {
			return err
		}
	}

	err = session.Shell()
	if err != nil {
		return err
	}

	// You should now be connected via SSH with a fully-interactive terminal
	// This call blocks until the user exits the session (e.g. via CTRL + D)
	return session.Wait()
}

func getSSHConfig(user string, privateKey []byte) (*ssh.ClientConfig, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		//nolint:gosec
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
}
