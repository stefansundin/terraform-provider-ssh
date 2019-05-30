package main

import (
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/crypto/ssh"
)

// copied from https://github.com/hashicorp/terraform/blob/7149894e418d06274bc5827c872edd58d887aad9/communicator/ssh/provisioner.go#L213-L232
func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode([]byte(pk))
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	signer, err := ssh.ParsePrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key file %q: %s", pk, err)
	}

	return ssh.PublicKeys(signer), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <plan>\n", os.Args[0])
		fmt.Printf("Compiled with sources for Terraform %s\n", terraform.VersionString())
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error loading file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	plan, err := terraform.ReadPlan(f)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}
	// fmt.Printf("Plan: %v\n", plan)

	var wg sync.WaitGroup
	for _, m := range plan.State.Modules {
		for _, r := range m.Resources {
			if r.Type == "ssh_tunnel" {
				d := r.Primary.Attributes
				username := d["username"]
				if username == "" {
					currentUser, err := user.Current()
					if err != nil {
						panic(err)
					}
					username = currentUser.Username

				}
				host := d["host"]
				privateKey := d["private_key"]
				localAddress := d["local_address"]
				remoteAddress := d["remote_address"]

				fmt.Printf("%s Forwarding %s to %s via %s.\n", m.Path, localAddress, remoteAddress, host)

				pubKeyAuth, err := readPrivateKey(privateKey)
				if err != nil {
					panic(err)
				}
				sshConf := &ssh.ClientConfig{
					User:            username,
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Auth:            []ssh.AuthMethod{pubKeyAuth},
				}

				localListener, err := net.Listen("tcp", localAddress)
				if err != nil {
					panic(err)
				}

				wg.Add(1)
				go func() {
					sshClientConn, err := ssh.Dial("tcp", host, sshConf)
					if err != nil {
						panic(err)
					}

					// The accept loop
					for {
						localConn, err := localListener.Accept()
						if err != nil {
							panic(err)
						}

						sshConn, err := sshClientConn.Dial("tcp", remoteAddress)
						if err != nil {
							panic(err)
						}

						// Send traffic from the SSH server -> local program
						go func() {
							_, err = io.Copy(sshConn, localConn)
							if err != nil {
								panic(err)
							}
						}()

						// Send traffic from the local program -> SSH server
						go func() {
							_, err = io.Copy(localConn, sshConn)
							if err != nil {
								panic(err)
							}
						}()
					}
				}()

			}
		}
	}
	wg.Wait()
}
