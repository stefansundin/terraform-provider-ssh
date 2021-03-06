terraform {
  required_providers {
    ssh = {
      source = "stephansundin/ssh"
    }
    consul = {
      source  = "consul"
      version = ">= 2.0"
    }
  }
  required_version = ">= 0.13"
}
