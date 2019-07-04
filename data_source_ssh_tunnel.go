package main

import (
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"strings"
	"time"

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
				Optional:    true,
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
			"certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A signed SSH certificate",
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
			"keepalive": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"keepalive_interval": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
		},
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

func dataSourceSSHTunnelRead(d *schema.ResourceData, meta interface{}) error {
	username := d.Get("user").(string)
	if username == "" {
		currentUser, err := user.Current()
		if err != nil {
			panic(err)
		}
		username = currentUser.Username
	}
	host := d.Get("host").(string)
	privateKey := d.Get("private_key").(string)
	certificate := d.Get("certificate").(string)
	localAddress := d.Get("local_address").(string)
	remoteAddress := d.Get("remote_address").(string)
	sshAgent := d.Get("ssh_agent").(bool)
	keepalive := d.Get("keepalive").(bool)
	keepaliveInterval := time.Duration(d.Get("keepalive_interval").(int))
	// default to port 22 if not specified
	if !strings.Contains(host, ":") {
		host = host + ":22"
		d.Set("host", host)
	}

	log.Printf("[DEBUG] user: %v", username)
	log.Printf("[DEBUG] host: %v", host)
	log.Printf("[DEBUG] localAddress: %v", localAddress)
	log.Printf("[DEBUG] remoteAddress: %v", remoteAddress)
	log.Printf("[DEBUG] sshAgent: %v", sshAgent)

	tunnelEstablished := d.Get("tunnel_established").(bool)
	log.Printf("[DEBUG] tunnelEstablished: %v", tunnelEstablished)

	if tunnelEstablished == false {
		d.Set("tunnel_established", true)

		sshConf := &ssh.ClientConfig{
			User:            username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth:            []ssh.AuthMethod{},
		}

		// https://github.com/hashicorp/terraform/blob/43a754829ae7afcb26bccd275fb3ae9d3e0cda88/communicator/ssh/provisioner.go#L240-L258
		if privateKey != "" {
			if certificate != "" {
				log.Println("using client certificate for authentication")

				certSigner, err := signCertWithPrivateKey(privateKey, certificate)
				if err != nil {
					panic(err)
				}
				sshConf.Auth = append(sshConf.Auth, certSigner)

				// prevent the clear text cert from being stored in the state file
				d.Set("certificate", "REDACTED")
			} else {
				log.Println("using private key for authentication")

				pubKeyAuth, err := readPrivateKey(privateKey)
				if err != nil {
					panic(err)
				}
				sshConf.Auth = append(sshConf.Auth, pubKeyAuth)
			}
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
			return fmt.Errorf("error: No authentication method configured")
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

		sshClientConn, err := ssh.Dial("tcp", host, sshConf)
		if err != nil {
			return fmt.Errorf("could not dial: %v", err)
		}

		// send keepalive requests into the established channel
		if keepalive {
			go func() {
				t := time.NewTicker(keepaliveInterval * time.Second)
				defer t.Stop()
				for range t.C {
					_, _, err = sshClientConn.SendRequest("keepalive@golang.org", true, nil)
					if err != nil {
						log.Printf("could not send keepalive: %v", err)
						continue
					}
				}
			}()
		}

		go func() {
			// The accept loop
			for {
				localConn, err := localListener.Accept()
				if err != nil {
					log.Printf("error accepting connection: %s", err)
					continue
				}

				sshConn, err := sshClientConn.Dial("tcp", remoteAddress)
				if err != nil {
					log.Printf("error opening connection to %s: %s", remoteAddress, err)
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
	}
	d.SetId(localAddress)

	return nil
}
