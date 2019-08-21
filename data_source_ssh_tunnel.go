package main

import (
	"context"
	"log"
	"os/user"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/stefansundin/terraform-provider-ssh/pb"
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
		},
	}
}

func dataSourceSSHTunnelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

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

	// prevent sensitive data from being stored in the state file
	if certificate != "" {
		d.Set("certificate", "REDACTED")
	}
	if privateKey != "" {
		d.Set("private_key", "REDACTED")
	}

	if tunnelEstablished == false {
		d.Set("tunnel_established", true)

		resp, err := client.grpcClient.OpenTunnel(context.Background(), &pb.TunnelConfiguration{
			User:          username,
			Host:          host,
			SshAgent:      sshAgent,
			PrivateKey:    privateKey,
			Certificate:   certificate,
			LocalAddress:  localAddress,
			RemoteAddress: remoteAddress,
		})
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] OpenTunnel: %v", resp.GetSuccess())

		if resp.EffectiveAddress != localAddress {
			log.Printf("[DEBUG] localAddress: %v", resp.EffectiveAddress)
			d.Set("local_address", resp.EffectiveAddress)
		}
		d.Set("port", resp.Port)

	}
	d.SetId(localAddress)

	return nil
}
