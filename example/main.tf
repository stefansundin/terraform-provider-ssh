provider "ssh" {
}

data "ssh_tunnel" "consul" {
  user = "root"
  auth {
    private_key {
      content = file(pathexpand("~/.ssh/id_rsa"))
    }
  }
  server {
    host = "8.8.8.8"
    port = 22
  }
  remote {
    port = 8500
  }
}

provider "consul" {
  address = data.ssh_tunnel.consul.local.0.address
  scheme  = "http"
}

data "consul_keys" "keys" {
  key {
    name = "revision"
    path = "secrets/api/password"
  }
}

output "local_address" {
  value = data.ssh_tunnel.consul.local.0.host
}

output "random_port" {
  value = data.ssh_tunnel.consul.local.0.port
}

output "revision" {
  value = data.consul_keys.keys.var.revision
}
