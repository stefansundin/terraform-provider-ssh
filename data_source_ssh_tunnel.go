package main

import (
	"encoding/pem"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"log"
	"net"
	"os"
	"strings"
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
				Description: "use ssh agent",
				Default:     true,
			},
			"ssh_agent_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The hostname",
				Default:     "",
			},
			"private_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SSH private key",
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

func agentAuth(sshAgentPath string) (ssh.AuthMethod, error) {
	if sshAgentPath == "" {
		sshAgentPath = os.Getenv("SSH_AUTH_SOCK")
	}

	log.Printf("[INFO] opening connection to %q", sshAgentPath)
	conn, err := net.Dial("unix", sshAgentPath)
	log.Print("[INFO] connection open ")
	if err != nil {
		return nil, fmt.Errorf("could not dial with socket, %v", err)
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

func dataSourceSSHTunnelRead(d *schema.ResourceData, meta interface{}) error {
	user := d.Get("user").(string)
	host := d.Get("host").(string)
	privateKey := d.Get("private_key").(string)
	localAddress := d.Get("local_address").(string)
	remoteAddress := d.Get("remote_address").(string)
	tunnelEstablished := d.Get("tunnel_established").(bool)
	sshAgent := d.Get("ssh_agent").(bool)
	sshAgentPath := d.Get("ssh_agent_path").(string)
	// default to port 22 if not specified
	if !strings.Contains(host, ":") {
		host = host + ":22"
		err := d.Set("host", host)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("[DEBUG] user: %v", user)
	log.Printf("[DEBUG] host: %v", host)
	log.Printf("[DEBUG] localAddress: %v", localAddress)
	log.Printf("[DEBUG] remoteAddress: %v", remoteAddress)
	log.Printf("[DEBUG] tunnelEstablished: %v", tunnelEstablished)

	if tunnelEstablished == false {

		sshConf := &ssh.ClientConfig{
			User:            user,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth:            []ssh.AuthMethod{},
		}

		if (os.Getenv("SSH_AUTH_SOCK") == "" && sshAgent) || (!sshAgent && privateKey == "") {
			return fmt.Errorf("either ssh-agent or a private key must be set")
		}

		if privateKey != "" {
			pubKeyAuth, err := readPrivateKey(privateKey)
			if err == nil {
				sshConf.Auth = append(sshConf.Auth, pubKeyAuth)
			} else {
				log.Printf("[INFO] Could not read private key, proceed with ssh-agent: %v", err)
			}
		}

		if os.Getenv("SSH_AUTH_SOCK") != "" && sshAgent {
			agent, err := agentAuth(sshAgentPath)
			if err != nil {
				return err
			}
			sshConf.Auth = append(sshConf.Auth, agent)
		}

		localListener, err := net.Listen("tcp", localAddress)
		if err != nil {
			return fmt.Errorf("could not start local listener, %v", err)
		}

		effectiveAddress := localListener.Addr().String()
		if effectiveAddress != localAddress {
			log.Printf("[DEBUG] localAddress: %v", effectiveAddress)
			err = d.Set("local_address", effectiveAddress)
			if err != nil {
				return fmt.Errorf("could not set local address, %v", err)
			}
		}

		lastColon := strings.LastIndex(effectiveAddress, ":")
		port := effectiveAddress[lastColon+1 : len(effectiveAddress)]
		log.Printf("[DEBUG] port: %v", port)
		err = d.Set("port", port)
		if err != nil {
			return fmt.Errorf("could not set local port, %v", err)
		}

		go func() {
			sshClientConn, err := ssh.Dial("tcp", host, sshConf)
			if err != nil {
				panic(err)
			}
			err = d.Set("tunnel_established", true)
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