package provider

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

func dataSourceSSHTunnel() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSSHTunnelRead,
		Schema: map[string]*schema.Schema{
			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SSH connection username",
				DefaultFunc: func() (interface{}, error) {
					return user.Current()
				},
			},
			"auth": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "SSH server auth",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sock": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Attempt to use the SSH agent (using the SSH_AUTH_SOCK environment variable)",
							DefaultFunc: func() (interface{}, error) {
								return os.Getenv("SSH_AUTH_SOCK"), nil
							},
						},
						"private_key": {
							Type:          schema.TypeList,
							Optional:      true,
							Description:   "SSH server auth",
							MaxItems:      1,
							ConflictsWith: []string{"auth.0.password"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The private SSH key",
										Sensitive:   true,
									},
									"password": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The private SSH key password",
										Sensitive:   true,
									},
									"certificate": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "A signed SSH certificate",
										Sensitive:   true,
									},
								},
							},
						},
						"password": {
							Type:          schema.TypeString,
							Optional:      true,
							Description:   "The private SSH key password",
							Sensitive:     true,
							ConflictsWith: []string{"auth.0.private_key"},
						},
					},
				},
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
						"socket": {
							Type:          schema.TypeString,
							Optional:      true,
							Description:   "local socket",
							ConflictsWith: []string{"local.0.host", "local.0.port"},
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								if val.(string) != "" && runtime.GOOS == "windows" {
									errs = append(errs, fmt.Errorf("%q is not allowed to be set on windows", key))
								}
								return
							},
						},
						"host": {
							Type:          schema.TypeString,
							Optional:      true,
							Default:       "localhost",
							Description:   "local bind host",
							ConflictsWith: []string{"local.0.socket"},
						},
						"port": {
							Type:          schema.TypeInt,
							Optional:      true,
							Description:   "local bind port",
							Default:       0,
							ValidateFunc:  validation.IsPortNumber,
							ConflictsWith: []string{"local.0.socket"},
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
						"socket": {
							Type:          schema.TypeString,
							Optional:      true,
							Description:   "remote socket",
							ConflictsWith: []string{"remote.0.host", "remote.0.port"},
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								if val.(string) != "" && runtime.GOOS == "windows" {
									errs = append(errs, fmt.Errorf("%q is not allowed to be set on windows", key))
								}
								return
							},
						},
						"host": {
							Type:          schema.TypeString,
							Optional:      true,
							Default:       "localhost",
							Description:   "remote bind host",
							ConflictsWith: []string{"remote.0.socket"},
						},
						"port": {
							Type:          schema.TypeInt,
							Optional:      true,
							Description:   "remote bind port",
							ValidateFunc:  validation.IsPortNumber,
							ConflictsWith: []string{"remote.0.socket"},
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
	if socket, ok := endpointData["socket"]; ok {
		endpoint.Socket = socket.(string)
	}
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
	m["socket"] = endpoint.Socket
	m["address"] = endpoint.Address()
	return []interface{}{m}
}

func dataSourceSSHTunnelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	sshTunnel := ssh.SSHTunnel{
		User: d.Get("user").(string),
	}

	sshTunnel.Auth = []ssh.SSHAuth{}

	if v, ok := d.GetOk("auth"); ok {
		methodsData := v.([]interface{})
		if len(methodsData) > 0 {
			methods := methodsData[0].(map[string]interface{})
			if v, ok := methods["sock"]; ok && runtime.GOOS != "windows" {
				sshTunnel.Auth = append(sshTunnel.Auth, ssh.SSHAuthSock{
					Path: v.(string),
				})
			}
			if v, ok := methods["private_key"]; ok && len(v.([]interface{})) > 0 {
				pkData := v.([]interface{})
				if len(pkData) > 0 {
					privateKey := ssh.SSHPrivateKey{}
					pk := pkData[0].(map[string]interface{})
					if content, ok := pk["content"]; ok {
						privateKey.PrivateKey = content.(string)
					}
					if password, ok := pk["password"]; ok {
						privateKey.Password = password.(string)
					}
					if certificate, ok := pk["certificate"]; ok {
						privateKey.Certificate = certificate.(string)
					}
					sshTunnel.Auth = append(sshTunnel.Auth, privateKey)
				}
			} else if v, ok := methods["password"]; ok {
				sshTunnel.Auth = append(sshTunnel.Auth, ssh.SSHPassword{
					Password: v.(string),
				})
			}
		}
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

	proto := "tcp"
	if sshTunnel.Local.Socket != "" {
		proto = "unix"
	}

	tunnelServer := ssh.NewSSHTunnelServer(&sshTunnel)
	tunnelServerInbound, err := net.Listen(proto, sshTunnel.Local.RandonPortString())
	if err != nil {
		return diag.FromErr(err)
	}

	if err = rpc.Register(tunnelServer); err != nil {
		return diag.FromErr(err)
	}

	go rpc.Accept(tunnelServerInbound)

	log.Printf("[DEBUG] starting RPC Server %s://%s monitoring ppid %d", proto, tunnelServerInbound.Addr().String(), os.Getppid())
	var cmdargs []string
	if runtime.GOOS == "windows" {
		cmdargs = []string{"cmd", "/C"}
	} else {
		cmdargs = []string{"/bin/sh", "-c"}
	}
	cmdargs = append(cmdargs, os.Args[0])
	cmd := exec.Command(cmdargs[0], cmdargs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := []string{
		fmt.Sprintf("TF_SSH_PROVIDER_TUNNEL_PROTO=%s", proto),
		fmt.Sprintf("TF_SSH_PROVIDER_TUNNEL_ADDR=%s", tunnelServerInbound.Addr().String()),
		fmt.Sprintf("TF_SSH_PROVIDER_TUNNEL_PPID=%d", os.Getppid()),
	}
	cmd.Env = append(cmd.Env, env...)
	err = cmd.Start()
	if err != nil {
		return diag.FromErr(err)
	}

	var commandError error
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	go func() {
		if err := cmd.Wait(); err != nil {
			commandError = err
		}
	}()

	go func() {
		<-timer.C
		commandError = fmt.Errorf("timed out during a tunnel setup")
	}()

	for !tunnelServer.Ready {
		log.Printf("[DEBUG] waiting for local port availability")
		if commandError != nil {
			return diag.FromErr(commandError)
		}
		time.Sleep(1 * time.Second)
	}

	tunnelServerInbound.Close()

	log.Printf("[DEBUG] local port: %v", sshTunnel.Local.Port)
	d.Set("local", flattenEndpoint(sshTunnel.Local))
	d.SetId(sshTunnel.Local.Address())

	return diags
}
