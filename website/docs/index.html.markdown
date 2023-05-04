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
provider "ssh" {
  user = "root"
  auth = {
    private_key = {
      content = file(pathexpand("~/.ssh/id_rsa"))
    }
  }
  server = {
    host = "localhost"
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
```

## Argument Reference

The following arguments are supported:

* `user` - (Optional) SSH connection username. Uses current OS user by default.
* `auth` - (Optional) Configuration block for SSH server auth. Detailed below.
* `server` - (Required) Configuration block for SSH address. Detailed below.

### auth Configuration Block

The following arguments are supported by the `auth` configuration block:

* `sock` - (Optional) SSH Agent UNIX socket path.
* `password` - (Optional) SSH server auth password. Conflicts with `auth.private_key`.
* `private_key` - (Optional) Configuration block for SSH private key auth. Conflicts with `auth.password`. Detailed below.

### server Configuration Block

The following arguments are supported by the `server` configuration block:

* `host` - (Required) SSH server hostname or IP.
* `port` - (Optional) SSH server port. Default port is equal to 22 by default.

### private_key Configuration Block

The following arguments are supported by the `private_key` configuration block:

* `content` - (Optional) SSH server private key.
* `password` - (Optional) SSH server private key password.
* `certificate` - (Optional) SSH server private key signing certificate.
