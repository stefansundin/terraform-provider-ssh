provider "ssh" {}

data "ssh_tunnel" "consul" {
  user        = "root" // Optional. If not set, your local user's username will be used.
  private_key = file(pathexpand("~/.ssh/id_rsa"))
  server {
    host = "localhost"
    port = 22
  }
  remote {
    port = 8500
  }
}

provider "consul" {
  address = data.ssh_tunnel.consul.local[0].address
  scheme  = "http"
}

data "consul_keys" "keys" {
  key {
    name = "revision"
    path = "secrets/api/password"
  }
}

data "ssh_tunnel_close" "consul" {
  depends_on = [data.consul_keys.keys]
}

output "local_address" {
  value = data.ssh_tunnel.consul.local[0].host
}

output "random_port" {
  value = data.ssh_tunnel.consul.local[0].port
}

output "revision" {
  value = data.consul_keys.keys.var.revision
}
