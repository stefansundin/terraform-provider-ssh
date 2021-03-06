package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHTunnel struct {
	User               string
	PrivateKey         string
	PrivateKeyPassword string
	Certificate        string
	SshAuthSock        string
	Local              Endpoint
	Remote             Endpoint
	Server             Endpoint
}

type Endpoint struct {
	Host string
	Port int
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func (st *SSHTunnel) Start(ctx context.Context) (err error) {
	log.Println("[DEBUG] Creating SSH Tunnel")

	sshConf := &ssh.ClientConfig{
		User:            st.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{},
	}

	if st.PrivateKey != "" {
		var signer ssh.Signer
		if st.PrivateKeyPassword != "" {
			log.Println("[DEBUG] using private key without password for authentication")
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(st.PrivateKey), []byte(st.PrivateKeyPassword))
		} else {
			log.Println("[DEBUG] using private key with password for authentication")
			signer, err = ssh.ParsePrivateKey([]byte(st.PrivateKey))
		}
		if err != nil {
			return fmt.Errorf("Failed to parse private key:\n%v", err)
		}
		sshConf.Auth = append(sshConf.Auth, ssh.PublicKeys(signer))
		if st.Certificate != "" {
			log.Println("[DEBUG] using client certificate for authentication")
			pcert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(st.Certificate))
			if err != nil {
				return fmt.Errorf("failed to parse certificate %q: %s", st.Certificate, err)
			}
			certSigner, err := ssh.NewCertSigner(pcert.(*ssh.Certificate), signer)
			if err != nil {
				return fmt.Errorf("failed to create cert signer %q: %s", certSigner, err)
			}
			sshConf.Auth = append(sshConf.Auth, ssh.PublicKeys(certSigner))
		}
	}

	if st.SshAuthSock != "" {
		log.Printf("[DEBUG] opening connection to %q", st.SshAuthSock)
		conn, err := net.Dial("unix", st.SshAuthSock)
		if err != nil {
			return err
		}
		agentClient := agent.NewClient(conn)
		agentAuth := ssh.PublicKeysCallback(agentClient.Signers)
		sshConf.Auth = append(sshConf.Auth, agentAuth)
	}

	if len(sshConf.Auth) == 0 {
		return fmt.Errorf("Error: No authentication method configured.")
	}

	localListener, err := net.Listen("tcp", st.Local.String())
	if err != nil {
		return err
	}

	netParts := strings.Split(localListener.Addr().String(), ":")
	st.Local.Port, _ = strconv.Atoi(netParts[1])
	sshClientConn, err := ssh.Dial("tcp", st.Server.String(), sshConf)
	if err != nil {
		return fmt.Errorf("could not dial: %v", err)
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
				log.Printf("error accepting connection: %s", err)
				continue
			}

			remoteConn, err := sshClientConn.Dial("tcp", st.Remote.String())
			if err != nil {
				log.Printf("error opening connection to %s: %s", st.Remote.String(), err)
				continue
			}

			go copyConn(localConn, remoteConn)
			go copyConn(remoteConn, localConn)

			select {
			case <-ctx.Done():
				localListener.Close()
			}
		}
	}()

	return nil
}
