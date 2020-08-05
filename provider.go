package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/stefansundin/terraform-provider-ssh/pb"
	"google.golang.org/grpc"
)

type Client struct {
	port       int
	grpcClient pb.SshTunnelClient
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"server_started": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ssh_tunnel": dataSourceSSHTunnel(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	port := d.Get("port").(int)
	if port == 0 {
		port, _ = getFreePort()
	}
	serverStarted := d.Get("server_started").(bool)
	serverAddr := fmt.Sprintf("localhost:%d", port)

	if serverStarted == false {
		d.Set("server_started", true)

		path, err := os.Executable()
		if err != nil {
			panic(err)
		}

		// is this the master terraform process? always?
		ppid := os.Getppid()

		cmd := exec.Command(path, "server", serverAddr, strconv.Itoa(ppid))
		err = cmd.Start()
		check(err)

		// wait for the server to start up
		time.Sleep(100 * time.Millisecond)
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure()) // TODO: don't use WithInsecure
	check(err)
	// defer conn.Close() // TODO: close this cleanly when the provider exits
	client := pb.NewSshTunnelClient(conn)

	return &Client{
		port:       port,
		grpcClient: client,
	}, nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
