### terraform-provider-ssh

This experimental provider enables SSH port forwarding in Terraform. It is intended as a
bandaid [until it is supported in Terraform itself](https://github.com/hashicorp/terraform/issues/8367).

#### Example

See [example/main.tf](main.tf).

#### Installation

Provider can be automatically installed using Terraform >= 0.13 by providing a `terraform` configuration block:

```
terraform {
    required_providers {
        ssh = {
            source = "AndrewChubatiuk/ssh"
        }
    }
}
```
