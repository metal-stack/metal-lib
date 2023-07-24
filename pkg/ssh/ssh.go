package ssh

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Client struct {
	*ssh.Client
}
type Env struct {
	Key   string
	Value string
}

func NewClientWithConnection(user, host string, privateKey []byte, conn net.Conn) (*Client, error) {
	sshConfig, err := getSSHConfig(user, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	sshConn, sshChan, req, err := ssh.NewClientConn(conn, host, sshConfig)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(sshConn, sshChan, req)
	if err != nil {
		return nil, err
	}
	return &Client{client}, nil
}

func NewClient(user, host string, privateKey []byte, port int) (*Client, error) {
	fmt.Printf("ssh to %s@%s:%d\n", user, host, port)
	sshConfig, err := getSSHConfig(user, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}
	sshServerAddress := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", sshServerAddress, sshConfig)
	return &Client{client}, err
}

func (c *Client) Connect(env *Env) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if env != nil {
		err = session.Setenv(env.Key, env.Value)
		if err != nil {
			return err
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
				fmt.Printf("error restoring ssh terminal:%v\n", err)
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
