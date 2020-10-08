terraform {
  required_providers {
    mysql = {
      source = "terraform-providers/mysql"
    }
    null = {
      source = "hashicorp/null"
    }
    ssh = {
      source = "kw.com/ssh/ssh"
      version = "0.0.6"

    }
  }
  required_version = ">= 0.13"
}

provider "ssh" {}

resource "null_resource" "vagrant" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = "screen -dm ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i .vagrant/machines/default/virtualbox/private_key -p 2201 -N -L 3307:127.0.0.1:3306 vagrant@127.0.0.1 sleep 60"
  }
}
