---
layout: "ssh"
page_title: "SSH: ssh_tunnel"
sidebar_current: "docs-ssh-data-source-tunnel"
description: |-
  Create SSH tunnel.
---

# ssh_tunnel

This data source provides a mechanism for creating an SSH tunnel.

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

* `local` - (Required) Configuration block for local SSH bind address. Detailed below.
* `remote` - (Required) Configuration block for remote SSH bind address. Detailed below.

### local Configuration Block

The following arguments are supported by the `local` configuration block:

* `host` - (Optional) local SSH bind hostname or IP. Default is localhost.
* `port` - (Optional) local SSH bind port. Default port is equal to 0 and is selected randomly.
* `socket` - (Optional) local SSH bind socket

### remote Configuration Block

The following arguments are supported by the `remote` configuration block:

* `host` - (Optional) remote SSH bind hostname or IP. Default is localhost.
* `port` - (Optional) remote SSH bind port.
* `socket` - (Optional) remote SSH bind socket
