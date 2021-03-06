package provider

import (
	"context"
	"log"
	"os"
	"os/user"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
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
			"server": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "SSH server address",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "SSH server host",
						},
						"port": {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "SSH server port",
							Default:      22,
							ValidateFunc: validation.IsPortNumber,
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SSH server address",
						},
					},
				},
			},
			"local": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Local bind address",
				MaxItems:    1,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "localhost",
							Description: "local bind host",
						},
						"port": {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "local bind port",
							Default:      0,
							ValidateFunc: validation.IsPortNumber,
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "local bind address",
						},
					},
				},
			},
			"remote": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Remote bind address",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "localhost",
							Description: "remote bind host",
						},
						"port": {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "remote bind port",
							ValidateFunc: validation.IsPortNumber,
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "remote bind address",
						},
					},
				},
			},
		},
	}
}

func expandEndpoint(m []interface{}) ssh.Endpoint {
	endpoint := ssh.Endpoint{}
	endpointData := m[0].(map[string]interface{})
	if host, ok := endpointData["host"]; ok {
		endpoint.Host = host.(string)
	}
	if port, ok := endpointData["port"]; ok {
		endpoint.Port = port.(int)
	}
	return endpoint
}

func flattenEndpoint(endpoint ssh.Endpoint) []interface{} {
	m := map[string]interface{}{}
	m["host"] = endpoint.Host
	m["port"] = endpoint.Port
	m["address"] = endpoint.String()
	return []interface{}{m}
}

func dataSourceSSHTunnelRead(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	//client := meta.(*SSHTunnelManager)

	sshTunnel := ssh.SSHTunnel{
		User:        d.Get("user").(string),
		SshAuthSock: d.Get("ssh_auth_sock").(string),
		PrivateKey:  d.Get("private_key").(string),
		Certificate: d.Get("certificate").(string),
	}

	if v, ok := d.GetOk("server"); ok {
		sshTunnel.Server = expandEndpoint(v.([]interface{}))
	}

	if v, ok := d.GetOk("local"); ok {
		sshTunnel.Local = expandEndpoint(v.([]interface{}))
	} else {
		sshTunnel.Local = ssh.Endpoint{Host: "localhost"}
	}

	if v, ok := d.GetOk("remote"); ok {
		sshTunnel.Remote = expandEndpoint(v.([]interface{}))
	}

	err := sshTunnel.Start()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] local port: %v", sshTunnel.Local.Port)
	d.Set("local", flattenEndpoint(sshTunnel.Local))
	d.SetId(sshTunnel.Local.String())

	return diags
}
