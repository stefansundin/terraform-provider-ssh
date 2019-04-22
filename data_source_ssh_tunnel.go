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

func agentAuth() ssh.AuthMethod {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	log.Printf("[INFO] opening connection to %q", sshAuthSock)
	conn, err := net.Dial("unix", sshAuthSock)
	log.Print("[INFO] connection open ")
	if err != nil {
		panic(err)
	}
	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

func copyConn(writer, reader net.Conn) {
	_, err := io.Copy(writer, reader)
	if err != nil {
		panic(err)
	}
}

func forward(d *schema.ResourceData, host, remoteAddress string, localListener net.Listener, sshConf *ssh.ClientConfig) {
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

		go copyConn(sshConn, localConn)

		go copyConn(localConn, sshConn)
	}
}


func dataSourceSSHTunnelRead(d *schema.ResourceData, meta interface{}) error {
	user := d.Get("user").(string)
	host := d.Get("host").(string)
	privateKey := d.Get("private_key_path").(string)
	localAddress := d.Get("local_address").(string)
	remoteAddress := d.Get("remote_address").(string)
	tunnelEstablished := d.Get("tunnel_established").(bool)
	sshAgent := d.Get("ssh_agent").(bool)

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
			return fmt.Errorf("Either ssh-agent or a private key must be set.")
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
			sshConf.Auth = append(sshConf.Auth, agentAuth())
		}

		localListener, err := net.Listen("tcp", localAddress)
		if err != nil {
			panic(err)
		}

		effectiveAddress := localListener.Addr().String()
		if effectiveAddress != localAddress {
			log.Printf("[DEBUG] localAddress: %v", effectiveAddress)
			err = d.Set("local_address", effectiveAddress)
			if err != nil {
				panic(err)
			}
		}

		lastColon := strings.LastIndex(effectiveAddress, ":")
		port := effectiveAddress[lastColon+1 : len(effectiveAddress)]
		log.Printf("[DEBUG] port: %v", port)
		err = d.Set("port", port)
		if err != nil {
			panic(err)
		}

		go forward(d, host, remoteAddress, localListener, sshConf)

	}
	d.SetId(localAddress)

	return nil
}
