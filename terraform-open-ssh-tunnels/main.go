package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <plan>\n", os.Args[0])
		//fmt.Printf("Compiled with sources for Terraform %s\n", terraform.VersionString())
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
				username := d["user"]
				if username == "" {
					currentUser, err := user.Current()
					if err != nil {
						panic(err)
					}
					username = currentUser.Username
				}
				host := d["host"]
				localAddress := d["local_address"]
				remoteAddress := d["remote_address"]
				sshAgent, _ := strconv.ParseBool(d["ssh_agent"])

				sshConf := &ssh.ClientConfig{
					User:            username,
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Auth:            []ssh.AuthMethod{},
				}
				if sshAgent {
					sshAuthSock, ok := os.LookupEnv("SSH_AUTH_SOCK")
					if ok {
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
					fmt.Printf("Error: No authentication method configured. Only SSH agent authentication is supported in this program at the moment.\n")
					return
				}

				fmt.Printf("%s Forwarding %s to %s via %s.\n", m.Path, localAddress, remoteAddress, host)

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
							fmt.Printf("error accepting connection: %s", err)
							continue
						}

						sshConn, err := sshClientConn.Dial("tcp", remoteAddress)
						if err != nil {
							fmt.Printf("error opening connection to %s: %s", remoteAddress, err)
							continue
						}

						// Send traffic from the SSH server -> local program
						go func() {
							_, err = io.Copy(sshConn, localConn)
							if err != nil {
								fmt.Printf("error copying data remote -> local: %s", err)
							}
						}()

						// Send traffic from the local program -> SSH server
						go func() {
							_, err = io.Copy(localConn, sshConn)
							if err != nil {
								fmt.Printf("error copying data local -> remote: %s", err)
							}
						}()
					}
				}()

			}
		}
	}
	wg.Wait()
}
