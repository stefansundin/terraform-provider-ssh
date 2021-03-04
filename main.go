package main

import (
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	hashiplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/stefansundin/terraform-provider-ssh/provider"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
	"github.com/stefansundin/terraform-provider-ssh/ssh/shared"
)

func main() {
	var ppid int
	var err error
	SSH_TUNNEL_PPID := os.Getenv("SSH_TUNNEL_PPID")
	if SSH_TUNNEL_PPID != "" {
		log.Println(SSH_TUNNEL_PPID)
		ppid, err = strconv.Atoi(SSH_TUNNEL_PPID)
		if err != nil {
			panic(err)
		}
	}
	if ppid > 0 {
		log.Printf("SSH Server Initialization with pid %d\n", ppid)
		go func() {
			process, err := os.FindProcess(ppid)
			if err != nil {
				panic(err)
			}
			err = nil
			for err == nil {
				time.Sleep(100 * time.Millisecond)
				err = process.Signal(syscall.Signal(0))
			}
			log.Printf("process.Signal on pid %d returned: %v\n", ppid, err)
			os.Exit(0)
		}()
		hashiplugin.Serve(&hashiplugin.ServeConfig{
			HandshakeConfig: shared.Handshake,
			Plugins: map[string]hashiplugin.Plugin{
				"tunnel": &shared.SSHTunnelGRPCPlugin{Impl: &ssh.SSHTunnel{}},
			},
			GRPCServer: hashiplugin.DefaultGRPCServer,
		})
	} else {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: func() *schema.Provider {
				return provider.SSHProvider()
			},
		})
	}
}
