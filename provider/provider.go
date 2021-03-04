package provider

import (
	"fmt"
	"os"
	"os/exec"

	hashiplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stefansundin/terraform-provider-ssh/ssh/shared"
)

type SSHProviderClient struct {
	tunnel shared.SSHTunnel
}

func SSHProvider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		DataSourcesMap: map[string]*schema.Resource{
			"ssh_tunnel": dataSourceSSHTunnel(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	ppid := os.Getppid()
	path, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("Couldn't run SSH server proxy: %v\n", err)
	}

	command := exec.Command("sh", "-c", path)
	command.Env = os.Environ()
	command.Env = append(command.Env, fmt.Sprintf("SSH_TUNNEL_PPID=%v", ppid))

	client := hashiplugin.NewClient(&hashiplugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		AutoMTLS:        true,
		Cmd:             command,
		AllowedProtocols: []hashiplugin.Protocol{
			hashiplugin.ProtocolGRPC,
		},
	})

	grpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err.Error())
	}

	raw, err := grpcClient.Dispense("tunnel")
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err.Error())
	}

	tunnel := raw.(shared.SSHTunnel)

	return &SSHProviderClient{
		tunnel: tunnel,
	}, nil
}
