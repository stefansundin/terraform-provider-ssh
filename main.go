package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/stefansundin/terraform-provider-ssh/provider"
	"github.com/stefansundin/terraform-provider-ssh/ssh"
)

func logSignals() {
	signals := make(chan os.Signal, 1)
	for {
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
		s := <-signals
		log.Printf("pid=%d received signal %s", os.Getpid(), s)
		signal.Reset()
		currentProcess, err := os.FindProcess(os.Getpid())
		if err != nil {
			log.Fatalf("[ERROR] failed to find current process: %s", err)
		}
		currentProcess.Signal(s)
	}
}

func main() {
	log.Printf("[DEBUG] pid=%d in main", os.Getpid())
	go logSignals()
	if _, ok := os.LookupEnv("TF_PLUGIN_MAGIC_COOKIE"); ok {
		var debug bool
		log.SetFlags(0)
		flag.BoolVar(&debug, "debug", false, "enable provider debug")
		flag.Parse()
		err := providerserver.Serve(context.Background(), provider.New(), providerserver.ServeOpts{
			Address: "registry.terraform.io/AndrewChubatiuk/ssh",
			Debug:   debug,
		})
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		var addr string
		var ppid int
		var proto string
		var name string
		var err error

		log.SetFlags(0)

		flag.IntVar(&ppid, "ppid", 0, "parent process pid")
		flag.StringVar(&addr, "addr", os.Getenv("TF_SSH_PROVIDER_TUNNEL_ADDR"), "set rpc server address")
		flag.StringVar(&name, "name", os.Getenv("TF_SSH_PROVIDER_TUNNEL_NAME"), "set rpc server name")
		flag.StringVar(&proto, "proto", os.Getenv("TF_SSH_PROVIDER_TUNNEL_PROTO"), "set rpc server protocol")
		flag.Parse()
		if ppid == 0 {
			if ppid, err = strconv.Atoi(os.Getenv("TF_SSH_PROVIDER_TUNNEL_PPID")); err != nil {
				log.Fatalf("[ERROR] parent process pid wasn't set")
			}
		}
		if addr == "" {
			log.Fatalf("[ERROR] RPC server address wasn't set")
		}
		var sshTunnel ssh.SSHTunnel
		if err := sshTunnel.Run(proto, name, addr, ppid); err != nil {
			log.Fatalf("[ERROR] failed to start SSH Tunnel:\n%s", err)
		}
	}
}
