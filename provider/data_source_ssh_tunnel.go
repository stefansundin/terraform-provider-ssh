package provider

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

type SocketValidator struct{}

func (v SocketValidator) Description(_ context.Context) string {
	return "sockets are not supported on windows"
}

func (v SocketValidator) MarkdownDescription(_ context.Context) string {
	return "sockets are not supported on windows"
}

func (v SocketValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() != "" && runtime.GOOS == "windows" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"socket validation error",
			v.Description(ctx),
		)
	}
}

type SSHTunnelDataSourceModel struct {
	Id     types.String   `tfsdk:"id"`
	Remote *EndpointModel `tfsdk:"remote"`
	Local  *EndpointModel `tfsdk:"local"`
}

type SSHTunnelDataSource struct {
	tunnel *ssh.SSHTunnel
}

func NewSSHTunnelDataSource() datasource.DataSource {
	return &SSHTunnelDataSource{}
}

func (d *SSHTunnelDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "ssh_tunnel"
}

func (d *SSHTunnelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Tunnel identifier",
			},
			"remote": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Remote bind address",
				Validators: []validator.Object{
					objectvalidator.AtLeastOneOf(
						path.MatchRelative().AtName("socket"),
						path.MatchRelative().AtName("port"),
					),
				},
				Attributes: map[string]schema.Attribute{
					"socket": schema.StringAttribute{
						Optional:    true,
						Description: "remote socket",
						Validators: []validator.String{
							SocketValidator{},
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("port"),
								path.MatchRelative().AtParent().AtName("host"),
							),
						},
					},
					"host": schema.StringAttribute{
						Optional:    true,
						Description: "remote bind host",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("socket"),
							),
						},
					},
					"port": schema.Int64Attribute{
						Optional:    true,
						Description: "remote bind port",
						Validators: []validator.Int64{
							int64validator.Between(0, 65535),
							int64validator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("socket"),
							),
						},
					},
					"address": schema.StringAttribute{
						Computed:    true,
						Description: "remote bind address",
					},
				},
			},
			"local": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Local bind address",
				Attributes: map[string]schema.Attribute{
					"socket": schema.StringAttribute{
						Optional:    true,
						Description: "local socket",
						Validators: []validator.String{
							SocketValidator{},
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("port"),
								path.MatchRelative().AtParent().AtName("host"),
							),
						},
					},
					"host": schema.StringAttribute{
						Optional:    true,
						Description: "local bind host",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("socket"),
							),
						},
					},
					"port": schema.Int64Attribute{
						Optional:    true,
						Description: "local bind port",
						Validators: []validator.Int64{
							int64validator.Between(0, 65535),
							int64validator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("socket"),
							),
						},
					},
					"address": schema.StringAttribute{
						Computed:    true,
						Description: "local bind address",
					},
				},
			},
		},
	}
}

func (d *SSHTunnelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sshTunnel, ok := req.ProviderData.(*ssh.SSHTunnel)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *SSHProviderModel, got %T", req.ProviderData),
		)
	}

	d.tunnel = sshTunnel
}

func (d *SSHTunnelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SSHTunnelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Local == nil {
		data.Local = &EndpointModel{}
	}
	if data.Local.Host.ValueString() == "" {
		data.Local.Host = types.StringValue("localhost")
	}
	if data.Remote.Host.ValueString() == "" {
		data.Remote.Host = types.StringValue("localhost")
	}

	proto := "tcp"
	if data.Local.Socket.ValueString() != "" {
		proto = "unix"
	}

	tunnel := &ssh.SSHTunnel{
		User:   d.tunnel.User,
		Auth:   d.tunnel.Auth,
		Server: d.tunnel.Server,
		Local:  data.Local.ToEndpoint(),
		Remote: data.Remote.ToEndpoint(),
	}

	tunnelServer := ssh.NewSSHTunnelServer(tunnel)
	tunnelServerInbound, err := net.Listen(proto, tunnel.Local.RandomPortString())
	if err != nil {
		resp.Diagnostics.AddError("proxy process error", err.Error())
		return
	}

	hash, err := hashstructure.Hash(tunnel.Remote, hashstructure.FormatV2, nil)
	if err != nil {
		resp.Diagnostics.AddError("rpc service name error", err.Error())
	}

	serviceName := fmt.Sprintf("SSHTunnelServer.%d", hash)

	if err = rpc.RegisterName(serviceName, tunnelServer); err != nil {
		resp.Diagnostics.AddError("rpc registration error", err.Error())
		return
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
		fmt.Sprintf("TF_SSH_PROVIDER_TUNNEL_NAME=%s", serviceName),
		fmt.Sprintf("TF_SSH_PROVIDER_TUNNEL_PPID=%d", os.Getppid()),
	}
	cmd.Env = append(cmd.Env, env...)
	err = cmd.Start()
	if err != nil {
		resp.Diagnostics.AddError("proxy start error", err.Error())
		return
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
			resp.Diagnostics.AddError("server proxy error", commandError.Error())
			return
		}
		time.Sleep(1 * time.Second)
	}

	tunnelServerInbound.Close()

	log.Printf("[DEBUG] local port: %v", tunnel.Local.Port)
	data.Local.Port = types.Int64Value(int64(tunnel.Local.Port))
	data.Local.Address = types.StringValue(tunnel.Local.Address())
	data.Id = types.StringValue(tunnel.Local.Address())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	return
}
