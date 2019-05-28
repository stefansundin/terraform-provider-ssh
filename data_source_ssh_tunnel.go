package main

import (
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func dataSourceSSHTunnel() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSSHTunnelRead,
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The hostname",
			},
			"ssh_agent": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Attempt to use the SSH agent (using the SSH_AUTH_SOCK environment variable)",
				Default:     true,
			},
			"private_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The private SSH key",
			},
			"local_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The local bind address (e.g. localhost:8500)",
			},
			"remote_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The remote bind address (e.g. localhost:8500)",
			},
			"port": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel_established": {
				// Probably not the proper way to store this
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

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

func dataSourceSSHTunnelRead(d *schema.ResourceData, meta interface{}) error {
	user := d.Get("user").(string)
	host := d.Get("host").(string)
	privateKey := d.Get("private_key").(string)
	localAddress := d.Get("local_address").(string)
	remoteAddress := d.Get("remote_address").(string)
	tunnelEstablished := d.Get("tunnel_established").(bool)
	sshAgent := d.Get("ssh_agent").(bool)
	// default to port 22 if not specified
	if !strings.Contains(host, ":") {
		host = host + ":22"
		d.Set("host", host)
	}

	log.Printf("[DEBUG] user: %v", user)
	log.Printf("[DEBUG] host: %v", host)
	log.Printf("[DEBUG] localAddress: %v", localAddress)
	log.Printf("[DEBUG] remoteAddress: %v", remoteAddress)
	log.Printf("[DEBUG] tunnelEstablished: %v", tunnelEstablished)
	log.Printf("[DEBUG] sshAgent: %v", sshAgent)

	if tunnelEstablished == false {
		d.Set("tunnel_established", true)

		sshConf := &ssh.ClientConfig{
			User:            user,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth:            []ssh.AuthMethod{},
		}

		if privateKey != "" {
			pubKeyAuth, err := readPrivateKey(privateKey)
			if err != nil {
				panic(err)
			}
			sshConf.Auth = append(sshConf.Auth, pubKeyAuth)
			// prevent the clear text key from being stored in the state file
			d.Set("private_key", "REDACTED")
		}

		if sshAgent {
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
			return fmt.Errorf("Error: No authentication method configured.")
		}

		localListener, err := net.Listen("tcp", localAddress)
		if err != nil {
			panic(err)
		}

		effectiveAddress := localListener.Addr().String()
		if effectiveAddress != localAddress {
			log.Printf("[DEBUG] localAddress: %v", effectiveAddress)
			d.Set("local_address", effectiveAddress)
		}

		lastColon := strings.LastIndex(effectiveAddress, ":")
		port := effectiveAddress[lastColon+1 : len(effectiveAddress)]
		log.Printf("[DEBUG] port: %v", port)
		d.Set("port", port)

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
	d.SetId(localAddress)

	return nil
}
