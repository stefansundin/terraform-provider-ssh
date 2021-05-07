---
layout: "ssh"
page_title: "Provider: SSH"
sidebar_current: "docs-ssh-index"
description: |-
  The SSH provider is used to create an SSH tunnel to connect to services behind bastion.
---

# SSH Provider

The SSH provider is used to create an SSH tunnel to connect to services behind bastion.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "ssh" {}

data "ssh_tunnel" "consul" {
  user = "root"
  auth {
    private_key {
      content = file(pathexpand("~/.ssh/id_rsa"))
    }
  }
  server {
    host = "localhost"
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

data "ssh_tunnel_close" "consul" {
  depends_on = [data.consul_keys.keys]
}
```
