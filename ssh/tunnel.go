package ssh

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"syscall"

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

func (e Endpoint) RandonPortString() string {
	if e.Socket != "" {
		return e.Socket
	}
	return fmt.Sprintf("%s:0", e.Host)
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
		log.Println("[DEBUG] using private key with password for authentication")
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(pk.PrivateKey), []byte(pk.Password))
	} else {
		log.Println("[DEBUG] using private key without password for authentication")
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
	User   string
	Auth   []SSHAuth
}

func (st *SSHTunnel) Run(proto, serverAddress string, ppid int) error {
	log.Println("[DEBUG] Creating SSH Tunnel")
	var ack bool
	gob.Register(SSHPrivateKey{})
	gob.Register(SSHAuthSock{})
	gob.Register(SSHPassword{})
	client, err := rpc.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatal("[ERROR] failed to connect to RPC server:\n", err)
	}

	defer client.Close()
	err = client.Call("SSHTunnelServer.GetSSHTunnel", &ack, &st)
	if err != nil {
		log.Fatal("[ERROR] Failed to execute a RPC call:\n", err)
	}

	sshConf := &ssh.ClientConfig{
		User:            st.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	for _, auth := range st.Auth {
		if auth.Enabled() {
			if methods, err := auth.Authenticate(); err != nil {
				return err
			} else {
				sshConf.Auth = append(sshConf.Auth, methods...)
			}
		}
	}

	localListener, err := net.Listen(proto, st.Local.String())
	if err != nil {
		return err
	}

	defer localListener.Close()

	if st.Local.Socket == "" {
		netParts := strings.Split(localListener.Addr().String(), ":")
		st.Local.Port, _ = strconv.Atoi(netParts[1])
	}

	sshClientConn, err := ssh.Dial("tcp", st.Server.String(), sshConf)
	if err != nil {
		return fmt.Errorf("could not dial: %v", err)
	}
	defer sshClientConn.Close()

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	proto = "tcp"
	if st.Remote.Socket != "" {
		proto = "unix"
	}

	err = client.Call("SSHTunnelServer.PutSSHReady", st.Local.Port, &ack)
	if err != nil {
		log.Fatal("[ERROR] Failed to execute a RPC call:\n", err)
	}

	go func(pid int) {
		for {
			process, err := os.FindProcess(pid)
			if err != nil {
				log.Printf("Failed to find process. Closing server: %s\n", err)
				localListener.Close()
				return
			}
			if err := process.Signal(syscall.Signal(0)); err != nil {
				log.Printf("Process %d is not alive anymore: %v\n", pid, err)
				localListener.Close()
				return
			}
		}
	}(ppid)

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

		remoteConn, err := sshClientConn.Dial(proto, st.Remote.String())
		if err != nil {
			log.Printf("error opening connection to %s: %s", st.Remote.Address(), err)
			continue
		}

		go copyConn(localConn, remoteConn)
		go copyConn(remoteConn, localConn)
	}

	return nil
}
