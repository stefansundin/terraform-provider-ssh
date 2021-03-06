---
layout: "insightops"
page_title: "SSH: ssh_tunnel"
sidebar_current: "docs-ssh-data-source-tunnel"
description: |-
  Create SSH tunnel.
---

# ssh_tunnel

This data source provides a mechanism for creating an SSH tunnel.

## Example Usage

```hcl
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
```

## Argument Reference

The following arguments are supported:

* `user` - (Optional) SSH connection username. Uses current OS user by default.
* `private_key` - (Optional) SSH connection private key.
* `private_key_password` - (Optional) SSH connection private key password.
* `certificate` - (Optional) SSH connection private key certificate.
* `server` - (Required) Configuration block for SSH address. Detailed below.
* `local` - (Required) Configuration block for local SSH bind address. Detailed below.
* `remote` - (Required) Configuration block for remote SSH bind address. Detailed below.

### server Configuration Block

The following arguments are supported by the `server` configuration block:

* `host` - (Required) SSH server hostname or IP.
* `port` - (Optional) SSH server port. Default port is equal to 22 by default.

### local Configuration Block

The following arguments are supported by the `local` configuration block:

* `host` - (Optional) local SSH bind hostname or IP. Default is localhost.
* `port` - (Optional) local SSH bind port. Default port is equal to 0 and is selected randomly.

### remote Configuration Block

The following arguments are supported by the `remote` configuration block:

* `host` - (Optional) remote SSH bind hostname or IP. Default is localhost.
* `port` - (Required) remote SSH bind port.