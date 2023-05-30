package provider

import (
	"context"
	"os"
	"os/user"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

type GenericEndpointModel struct {
	Host types.String `tfsdk:"host"`
	Port types.Int64  `tfsdk:"port"`
}

func (e *GenericEndpointModel) ToEndpoint() ssh.Endpoint {
	return ssh.Endpoint{
		Host: e.Host.ValueString(),
		Port: int(e.Port.ValueInt64()),
	}
}

type EndpointModel struct {
	Host    types.String `tfsdk:"host"`
	Port    types.Int64  `tfsdk:"port"`
	Socket  types.String `tfsdk:"socket"`
	Address types.String `tfsdk:"address"`
}

func (e *EndpointModel) ToEndpoint() ssh.Endpoint {
	return ssh.Endpoint{
		Host:   e.Host.ValueString(),
		Port:   int(e.Port.ValueInt64()),
		Socket: e.Socket.ValueString(),
	}
}

var _ provider.Provider = &SSHProvider{}

type SSHProviderModel struct {
	User   types.String          `tfsdk:"user"`
	Auth   *SSHProviderAuthModel `tfsdk:"auth"`
	Server *GenericEndpointModel `tfsdk:"server"`
}

type SSHProviderAuthModel struct {
	Sock       types.String                    `tfsdk:"sock"`
	PrivateKey *SSHProviderAuthPrivateKeyModel `tfsdk:"private_key"`
	Password   types.String                    `tfsdk:"password"`
}

type SSHProviderAuthPrivateKeyModel struct {
	Content     types.String `tfsdk:"content"`
	Password    types.String `tfsdk:"password"`
	Certificate types.String `tfsdk:"certificate"`
}

type SSHProvider struct{}

func (p *SSHProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ssh"
}

func (p *SSHProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user": schema.StringAttribute{
				Optional:    true,
				Description: "SSH connection username",
			},
			"auth": schema.SingleNestedAttribute{
				Description: "SSH server auth",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"sock": schema.StringAttribute{
						Optional:    true,
						Description: "Attempt to use the SSH agent (using the SSH_AUTH_SOCK environment variable)",
						Validators: []validator.String{
							SocketValidator{},
						},
					},
					"private_key": schema.SingleNestedAttribute{
						Description: "SSH server private key",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"content": schema.StringAttribute{
								Sensitive:   true,
								Description: "The private SSH key data",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Sensitive:   true,
								Description: "The private SSH key password",
								Optional:    true,
							},
							"certificate": schema.StringAttribute{
								Sensitive:   true,
								Description: "A signed SSH certificate",
								Optional:    true,
							},
						},
						Validators: []validator.Object{
							objectvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("password"),
							),
						},
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Description: "The private SSH key password",
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("private_key"),
							),
						},
					},
				},
			},
			"server": schema.SingleNestedAttribute{
				Description: "SSH Server configuration",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Required:    true,
						Description: "SSH server host",
					},
					"port": schema.Int64Attribute{
						Optional:    true,
						Description: "SSH server port",
						Validators: []validator.Int64{
							int64validator.Between(0, 65535),
						},
					},
				},
			},
		},
	}
}

func (p *SSHProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	var data SSHProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sshTunnel := ssh.SSHTunnel{
		Server: data.Server.ToEndpoint(),
	}
	if data.User.ValueString() != "" {
		sshTunnel.User = data.User.ValueString()
	} else {
		currentUser, err := user.Current()
		if err != nil {
			resp.Diagnostics.AddError(
				"current user retrieval error", err.Error(),
			)
			return
		}
		sshTunnel.User = currentUser.Username
	}
	if data.Auth.Sock.ValueString() != "" {
		authSock = data.Auth.Sock.ValueString()
	}

	sshTunnel.Auth = []ssh.SSHAuth{}
	privateKey := ssh.SSHPrivateKey{}
	if data.Auth.PrivateKey.Content.ValueString() != "" {
		privateKey.PrivateKey = data.Auth.PrivateKey.Content.ValueString()
	}
	if data.Auth.PrivateKey.Password.ValueString() != "" {
		privateKey.Password = data.Auth.PrivateKey.Password.ValueString()
	}
	if data.Auth.PrivateKey.Certificate.ValueString() != "" {
		privateKey.Certificate = data.Auth.PrivateKey.Certificate.ValueString()
	}
	sshTunnel.Auth = append(sshTunnel.Auth, privateKey)
	if authSock != "" {
		sshTunnel.Auth = append(sshTunnel.Auth, ssh.SSHAuthSock{
			Path: authSock,
		})
	}
	if data.Auth.Password.ValueString() != "" {
		sshTunnel.Auth = append(sshTunnel.Auth, ssh.SSHPassword{
			Password: data.Auth.Password.ValueString(),
		})
	}
	resp.DataSourceData = &sshTunnel
}

func (p *SSHProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSSHTunnelDataSource,
	}
}

func (p *SSHProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &SSHProvider{}
	}
}
