package provider

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

var (
	addressStateFunc = func(port int) schema.SchemaStateFunc {
		return func(val interface{}) string {
			listen := val.(string)
			parsedHost, _ := url.Parse(fmt.Sprintf("//%s", listen))
			if parsedHost.Port() == "" {
				if port != -1 {
					listen = fmt.Sprintf("%s:%i", listen, port)
				}
			}
			return listen
		}
	}

	addressValidateFunc = func(requirePort bool) schema.SchemaValidateFunc {
		return func(val interface{}, key string) (warns []string, errs []error) {
			listen := val.(string)
			parsedHost, err := url.Parse(fmt.Sprintf("//%s", listen))
			if err != nil {
				errs = append(errs, fmt.Errorf("Invalid host address %v\n%v\n", listen, err))
			}
			if parsedHost.Port() == "" && requirePort {
				errs = append(errs, fmt.Errorf("Port is required for address %v\n", listen))
			}
			return
		}
	}
)

func dataSourceSSHTunnel() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSSHTunnelRead,
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SSH connection username",
				DefaultFunc: func() (interface{}, error) {
					return user.Current()
				},
			},
			"address": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "SSH host address",
				ValidateFunc: addressValidateFunc(false),
				StateFunc:    addressStateFunc(22),
			},
			"ssh_auth_sock": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Attempt to use the SSH agent (using the SSH_AUTH_SOCK environment variable)",
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("SSH_AUTH_SOCK"), nil
				},
			},
			"private_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The private SSH key",
				Sensitive:   true,
			},
			"certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A signed SSH certificate",
				Sensitive:   true,
			},
			"local_address": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The local bind address (e.g. localhost:8500)",
				ValidateFunc: addressValidateFunc(false),
				StateFunc:    addressStateFunc(0),
			},
			"remote_address": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The remote bind address (e.g. localhost:8500)",
				ValidateFunc: addressValidateFunc(true),
				StateFunc:    addressStateFunc(-1),
			},
			"local_port": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceSSHTunnelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*ssh.SSHTunnel)

	sshTunnelConfig := ssh.SSHTunnelConfig{
		User:          d.Get("user").(string),
		Address:       d.Get("address").(string),
		SshAuthSock:   d.Get("ssh_auth_sock").(string),
		PrivateKey:    d.Get("private_key").(string),
		Certificate:   d.Get("certificate").(string),
		LocalAddress:  d.Get("local_address").(string),
		RemoteAddress: d.Get("remote_address").(string),
	}

	log.Printf("[DEBUG] tunnel configuration: %v", sshTunnelConfig)

	effectiveAddress, err := client.Init(sshTunnelConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] tunnel address: %v", effectiveAddress)

	if effectiveAddress != sshTunnelConfig.LocalAddress {
		sshTunnelConfig.LocalAddress = effectiveAddress
		parsedEffectiveAddress, err := url.Parse(fmt.Sprintf("//%s", sshTunnelConfig.LocalAddress))
		if err != nil {
			return diag.FromErr(err)
		}
		port := parsedEffectiveAddress.Port()
		log.Printf("[DEBUG] local_port: %v", port)
		d.Set("local_port", port)
		d.Set("local_address", sshTunnelConfig.LocalAddress)
	}

	log.Printf("[DEBUG] local_address: %v", sshTunnelConfig.LocalAddress)
	d.SetId(sshTunnelConfig.LocalAddress)

	return diags
}
