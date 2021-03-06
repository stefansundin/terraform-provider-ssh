provider "ssh" {
}

data "ssh_tunnel" "consul" {
  user            = "root" // Optional. If not set, your local user's username will be used.
  address         = "49.12.108.106:22"
  private_key     = file(pathexpand("~/.ssh/blacksparkle/prod-admin"))
  local_address   = "localhost:0" // use port 0 to request an ephemeral port (a random port)
  remote_address  = "localhost:8500"
}

provider "consul" {
  address    = data.ssh_tunnel.consul.local_address
  scheme     = "http"
}

data "consul_keys" "keys" {
  key {
    name = "revision"
    path = "secrets/api/password"
  }
}

data "ssh_tunnel_close" "consul" {
  depends_on      = [data.consul_keys.keys]
}

output "local_address" {
  value = data.ssh_tunnel.consul.local_address
}

output "random_port" {
  value = data.ssh_tunnel.consul.local_port
}

output "revision" {
  value = data.consul_keys.keys.var.revision
}
