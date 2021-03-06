package ssh

import (
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHTunnel struct{}

type SSHTunnelConfig struct {
	User          string
	PrivateKey    string
	Certificate   string
	SshAuthSock   string
	LocalAddress  string
	RemoteAddress string
	Address       string
}

func (st *SSHTunnel) Init(request SSHTunnelConfig) (address string, err error) {
	log.Println("[DEBUG] Creating SSH Tunnel")

	sshConf := &ssh.ClientConfig{
		User:            request.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{},
	}

	if request.PrivateKey != "" {
		if request.Certificate != "" {
			log.Println("[DEBUG] using client certificate for authentication")
			certSigner, err := signCertWithPrivateKey(request.PrivateKey, request.Certificate)
			if err != nil {
				return "", err
			}
			sshConf.Auth = append(sshConf.Auth, certSigner)
		} else {
			log.Printf("[DEBUG] using private key for authentication")
			pubKeyAuth, err := readPrivateKey(request.PrivateKey)
			if err != nil {
				return "", err
			}
			sshConf.Auth = append(sshConf.Auth, pubKeyAuth)
		}
	}

	if request.SshAuthSock != "" {
		log.Printf("[DEBUG] opening connection to %q", request.SshAuthSock)
		conn, err := net.Dial("unix", request.SshAuthSock)
		if err != nil {
			return "", err
		}
		agentClient := agent.NewClient(conn)
		agentAuth := ssh.PublicKeysCallback(agentClient.Signers)
		sshConf.Auth = append(sshConf.Auth, agentAuth)
	}

	if len(sshConf.Auth) == 0 {
		return "", fmt.Errorf("Error: No authentication method configured.")
	}

	localListener, err := net.Listen("tcp", request.LocalAddress)
	if err != nil {
		return "", err
	}
	effectiveAddress := localListener.Addr().String()
	sshClientConn, err := ssh.Dial("tcp", request.Address, sshConf)
	if err != nil {
		return "", fmt.Errorf("could not dial: %v", err)
	}

	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				log.Printf("error accepting connection: %s", err)
				continue
			}
			defer localConn.Close()

			sshConn, err := sshClientConn.Dial("tcp", request.RemoteAddress)
			if err != nil {
				log.Printf("error opening connection to %s: %s", request.RemoteAddress, err)
				continue
			}
			defer sshConn.Close()

			go func() {
				_, err = io.Copy(sshConn, localConn)
				if err != nil {
					log.Printf("error copying data remote -> local: %s", err)
				}
			}()
			go func() {
				_, err = io.Copy(localConn, sshConn)
				if err != nil {
					log.Printf("error copying data local -> remote: %s", err)
				}
			}()
		}
	}()

	return effectiveAddress, nil
}

func signCertWithPrivateKey(pk string, certificate string) (ssh.AuthMethod, error) {
	rawPk, err := ssh.ParseRawPrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key %q: %s", pk, err)
	}

	pcert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certificate))
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate %q: %s", certificate, err)
	}

	usigner, err := ssh.NewSignerFromKey(rawPk)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from raw private key %q: %s", rawPk, err)
	}

	ucertSigner, err := ssh.NewCertSigner(pcert.(*ssh.Certificate), usigner)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert signer %q: %s", usigner, err)
	}

	return ssh.PublicKeys(ucertSigner), nil
}

func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	block, _ := pem.Decode([]byte(pk))
	if block == nil {
		return nil, fmt.Errorf("Failed to read ssh private key: no key found")
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read ssh private key: password protected keys are\n" +
				"not supported. Please decrypt the key prior to use.")
	}

	signer, err := ssh.ParsePrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse ssh private key: %s", err)
	}

	return ssh.PublicKeys(signer), nil
}
