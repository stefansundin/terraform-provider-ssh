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
  user        = "root"
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

## Argument Reference

The following arguments are supported:

* `user` - (Optional) SSH connection username. Uses current OS user by default.
* `auth` - (Optional) Configuration block for SSH server auth. Detailed below.
* `server` - (Required) Configuration block for SSH address. Detailed below.
* `local` - (Required) Configuration block for local SSH bind address. Detailed below.
* `remote` - (Required) Configuration block for remote SSH bind address. Detailed below.

### auth Configuration Block

The following arguments are supported by the `auth` configuration block:

* `sock` - (Optional) SSH Agent UNIX socket path.
* `password` - (Optional) SSH server auth password. Conflicts with `auth.0.private_key`.
* `private_key` - (Optional) Configuration block for SSH private key auth. Conflicts with `auth.0.password`. Detailed below.

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

### private_key Configuration Block

The following arguments are supported by the `private_key` configuration block:

* `content` - (Optional) SSH server private key.
* `password` - (Optional) SSH server private key password.
* `certificate` - (Optional) SSH server private key signing certificate.