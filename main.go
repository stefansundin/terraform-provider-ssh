package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stefansundin/terraform-provider-ssh/pb"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"google.golang.org/grpc"
)

type sshTunnelServer struct {
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// copied from https://github.com/hashicorp/terraform/blob/43a754829ae7afcb26bccd275fb3ae9d3e0cda88/communicator/ssh/provisioner.go#L274-L317
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
	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
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

func (s *sshTunnelServer) OpenTunnel(ctx context.Context, conf *pb.TunnelConfiguration) (*pb.Response, error) {
	fmt.Printf("Received OpenTunnel call!\n")

	sshConf := &ssh.ClientConfig{
		User:            conf.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{},
	}

	// https://github.com/hashicorp/terraform/blob/43a754829ae7afcb26bccd275fb3ae9d3e0cda88/communicator/ssh/provisioner.go#L240-L258
	if conf.PrivateKey != "" {
		if conf.Certificate != "" {
			log.Println("using client certificate for authentication")

			certSigner, err := signCertWithPrivateKey(conf.PrivateKey, conf.Certificate)
			if err != nil {
				panic(err)
			}
			sshConf.Auth = append(sshConf.Auth, certSigner)
		} else {
			log.Println("using private key for authentication")

			pubKeyAuth, err := readPrivateKey(conf.PrivateKey)
			if err != nil {
				panic(err)
			}
			sshConf.Auth = append(sshConf.Auth, pubKeyAuth)
		}
	}

	if conf.SshAgent {
		sshAuthSock, ok := os.LookupEnv("SSH_AUTH_SOCK")
		if ok {
			log.Printf("[DEBUG] opening connection to %q", sshAuthSock)
			conn, err := net.Dial("unix", sshAuthSock)
			if err != nil {
				panic(err)
			}
			agentClient := agent.NewClient(conn)
			agentAuth := ssh.PublicKeysCallback(agentClient.Signers)
			sshConf.Auth = append(sshConf.Auth, agentAuth)
		}
	}

	if len(sshConf.Auth) == 0 {
		return nil, fmt.Errorf("Error: No authentication method configured.")
	}

	localListener, err := net.Listen("tcp", conf.LocalAddress)
	check(err)

	effectiveAddress := localListener.Addr().String()

	lastColon := strings.LastIndex(effectiveAddress, ":")
	port, err := strconv.Atoi(effectiveAddress[lastColon+1 : len(effectiveAddress)])
	check(err)
	log.Printf("[DEBUG] port: %v", port)

	sshClientConn, err := ssh.Dial("tcp", conf.Host, sshConf)
	if err != nil {
		return nil, fmt.Errorf("could not dial: %v", err)
	}

	go func() {
		// The accept loop
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				log.Printf("error accepting connection: %s", err)
				continue
			}

			sshConn, err := sshClientConn.Dial("tcp", conf.RemoteAddress)
			if err != nil {
				log.Printf("error opening connection to %s: %s", conf.RemoteAddress, err)
				continue
			}

			// Send traffic from the SSH server -> local program
			go func() {
				_, err = io.Copy(sshConn, localConn)
				if err != nil {
					log.Printf("error copying data remote -> local: %s", err)
				}
			}()

			// Send traffic from the local program -> SSH server
			go func() {
				_, err = io.Copy(localConn, sshConn)
				if err != nil {
					log.Printf("error copying data local -> remote: %s", err)
				}
			}()
		}
	}()

	return &pb.Response{
		Success:          true,
		EffectiveAddress: effectiveAddress,
		Port:             int32(port),
	}, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "server" {
		if len(os.Args) != 4 {
			fmt.Println("Usage: server <server_address> <terraform_pid>")
			os.Exit(1)
		}

		serverAddr := os.Args[2]
		pid, err := strconv.Atoi(os.Args[3])
		check(err)

		fmt.Printf("serverAddr: %s. pid: %d.\n", serverAddr, pid)

		// Wait for the main terraform process to exit, and when it does, we also exit
		go func() {
			process, err := os.FindProcess(pid)
			check(err)

			err = nil
			for err == nil {
				time.Sleep(100 * time.Millisecond)
				err = process.Signal(syscall.Signal(0))
			}
			fmt.Printf("process.Signal on pid %d returned: %v\n", pid, err)
			os.Exit(0)
		}()

		// Start the gRPC server
		listener, err := net.Listen("tcp", serverAddr)
		check(err)
		grpcServer := grpc.NewServer()
		pb.RegisterSshTunnelServer(grpcServer, &sshTunnelServer{})
		grpcServer.Serve(listener)
	} else {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: func() terraform.ResourceProvider {
				return Provider()
			},
		})
	}
}
