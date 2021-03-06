---
layout: "insightops"
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
  user        = "root"
  private_key = file(pathexpand("~/.ssh/id_rsa"))
  server {
    host = "localhost"
    port = 22
  }
  remote {
    port = 8500
  }
}
```