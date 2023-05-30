provider "ssh" {
  auth = {
    private_key = {
      content = file(pathexpand("~/.ssh/id_rsa"))
    }
  }
  server = {
    host = "10.18.21.31"
    port = 22
  }
}

data "ssh_tunnel" "consul" {
  remote = {
    port = 8500
  }
}

provider "consul" {
  address = data.ssh_tunnel.consul.local.address
  scheme  = "http"
}

data "consul_keys" "keys" {
  key {
    name = "revision"
    path = "secrets/api/password"
  }
}

output "local_address" {
  value = data.ssh_tunnel.consul.local.host
}

output "random_port" {
  value = data.ssh_tunnel.consul.local.port
}

output "revision" {
  value = data.consul_keys.keys.var.revision
}
