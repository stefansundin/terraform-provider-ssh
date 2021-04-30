package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net"
)

type SSHProviderManager struct {
	Listeners []*net.Listener
}

func SSHProvider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		DataSourcesMap: map[string]*schema.Resource{
			"ssh_tunnel":       dataSourceSSHTunnel(),
			"ssh_tunnel_close": dataSourceSSHTunnelClose(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	return &SSHProviderManager{}, diags
}
