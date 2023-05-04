terraform {
  required_providers {
    ssh = {
      source = "AndrewChubatiuk/ssh"
    }
    consul = {
      source  = "consul"
      version = ">= 2.0"
    }
  }
  required_version = ">= 0.14"
}
