provider "ssh" {}

data "ssh_tunnel" "consul" {
  user            = "stefan" // Optional. If not set, your local user's username will be used.
  host            = "bastion.example.com"
  private_key     = "${file(pathexpand("~/.ssh/id_rsa"))}"
  # certificate     = "${file(pathexpand("~/.ssh/id_rsa-cert.pub"))}" // if you need to use a certificate
  ssh_agent       = false // by default, SSH agent authentication is attempted if the SSH_AUTH_SOCK environment variable is set
  local_address   = "localhost:0" // use port 0 to request an ephemeral port (a random port)
  remote_address  = "localhost:8500"
  # http_proxy    = "http://proxy.corp:80" // http/https proxy to connect throuth
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
output "random_port" {
  value = "${data.ssh_tunnel.consul.port}"
}
output "revision" {
  value = "${data.consul_keys.keys.var.revision}"
}
