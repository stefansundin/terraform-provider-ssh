package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/stefansundin/terraform-provider-ssh/provider"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

func main() {
	if _, ok := os.LookupEnv(plugin.Handshake.MagicCookieKey); ok {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: func() *schema.Provider {
				return provider.SSHProvider()
			},
		})
	} else if _, ok := os.LookupEnv("TF_SSH_PROVIDER_PARAMS"); ok {
		serializedParams, err := base64.StdEncoding.DecodeString(os.Getenv("TF_SSH_PROVIDER_PARAMS"))
		if err != nil {
			log.Fatalf("[ERROR] Failed to decode base64 params:\n%s", err)
		}
		var sshTunnel ssh.SSHTunnel
		if err = json.Unmarshal([]byte(serializedParams), &sshTunnel); err != nil {
			log.Fatalf("[ERROR] Failed to parse json config:\n%s", err)
		}

		if err = sshTunnel.Run(); err != nil {
			log.Fatalf("[ERROR] Failed to start SSH Tunnel:\n%s", err)
		}
	}
}
