package ssh

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Endpoint struct {
	Host   string
	Port   int
	Socket string
}

func (e Endpoint) Address() string {
	if e.Socket != "" {
		return fmt.Sprintf("unix://%s", e.Socket)
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func (e Endpoint) String() string {
	if e.Socket != "" {
		return e.Socket
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

type SSHAuthSock struct {
	Path string
}

func (sa SSHAuthSock) Enabled() bool {
	return sa.Path != ""
}

func (sa SSHAuthSock) Authenticate() (methods []ssh.AuthMethod, err error) {
	conn, err := net.Dial("unix", sa.Path)
	if err != nil {
		return nil, err
	}
	agentClient := agent.NewClient(conn)
	methods = append(methods, ssh.PublicKeysCallback(agentClient.Signers))
	return
}

type SSHPassword struct {
	Password string
}

func (pw SSHPassword) Enabled() bool {
	return pw.Password != ""
}

func (pw SSHPassword) Authenticate() (methods []ssh.AuthMethod, err error) {
	methods = append(methods, ssh.Password(pw.Password))
	return
}

type SSHPrivateKey struct {
	PrivateKey  string
	Password    string
	Certificate string
}

func (pk SSHPrivateKey) Enabled() bool {
	return pk.PrivateKey != ""
}

func (pk SSHPrivateKey) Authenticate() (methods []ssh.AuthMethod, err error) {
	var signer ssh.Signer
	if pk.Password != "" {
		log.Println("[DEBUG] using private key without password for authentication")
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(pk.PrivateKey), []byte(pk.Password))
	} else {
		log.Println("[DEBUG] using private key with password for authentication")
		signer, err = ssh.ParsePrivateKey([]byte(pk.PrivateKey))
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key:\n%v", err)
	}
	methods = append(methods, ssh.PublicKeys(signer))
	if pk.Certificate != "" {
		log.Println("[DEBUG] using client certificate for authentication")
		pcert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pk.Certificate))
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate %q: %s", pk.Certificate, err)
		}
		certSigner, err := ssh.NewCertSigner(pcert.(*ssh.Certificate), signer)
		if err != nil {
			return nil, fmt.Errorf("failed to create cert signer %q: %s", certSigner, err)
		}
		methods = append(methods, ssh.PublicKeys(certSigner))
	}
	return
}

type SSHAuth interface {
	Enabled() bool
	Authenticate() (methods []ssh.AuthMethod, err error)
}

type SSHTunnel struct {
	Local  Endpoint
	Remote Endpoint
	Server Endpoint
	Auth   []SSHAuth
	User   string
}

func (st *SSHTunnel) Start() (*net.Listener, error) {
	log.Println("[DEBUG] Creating SSH Tunnel")

	sshConf := &ssh.ClientConfig{
		User:            st.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	for _, auth := range st.Auth {
		if auth.Enabled() {
			if methods, err := auth.Authenticate(); err != nil {
				return nil, err
			} else {
				sshConf.Auth = append(sshConf.Auth, methods...)
			}
		}
	}

	protocol := "tcp"
	if st.Local.Socket != "" {
		protocol = "unix"
	}

	localListener, err := net.Listen(protocol, st.Local.String())
	if err != nil {
		return nil, err
	}

	if st.Local.Socket == "" {
		netParts := strings.Split(localListener.Addr().String(), ":")
		st.Local.Port, _ = strconv.Atoi(netParts[1])
	}

	sshClientConn, err := ssh.Dial("tcp", st.Server.String(), sshConf)
	if err != nil {
		return nil, fmt.Errorf("could not dial: %v", err)
	}

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Printf("Stopping connection loop")
					break
				}
				log.Printf("error accepting connection: %s", err)
				continue
			}

			protocol = "tcp"

			if st.Remote.Socket != "" {
				protocol = "unix"
			}

			remoteConn, err := sshClientConn.Dial(protocol, st.Remote.String())
			if err != nil {
				log.Printf("error opening connection to %s: %s", st.Remote.Address(), err)
				continue
			}

			go copyConn(localConn, remoteConn)
			go copyConn(remoteConn, localConn)
		}
	}()

	return &localListener, nil
}
