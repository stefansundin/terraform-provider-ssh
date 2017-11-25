data "ssh_tunnel" "consul" {
  user            = "stefan"
  host            = "bastion.example.com"
  private_key     = "${file(pathexpand("~/.ssh/id_rsa"))}"
  local_address   = "localhost:8500"
  remote_address  = "localhost:8500"
}

provider "consul" {
  version    = "~> 1.0"
  address    = "${data.ssh_tunnel.consul.local_address}"
  scheme     = "http"
}

data "consul_keys" "keys" {
  key {
    name = "revision"
    path = "revision"
  }
}

output "local_address" {
  value = "${data.ssh_tunnel.consul.local_address}"
}
output "revision" {
  value = "${data.consul_keys.keys.var.revision}"
}
