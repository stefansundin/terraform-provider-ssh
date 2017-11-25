### terraform-provider-ssh

This provider enables SSH port forwarding in Terraform. It is intended as a
bandaid [until it is supported in Terraform itself](https://github.com/hashicorp/terraform/issues/8367).

#### Example

See [main.tf](main.tf).

#### TODO

- Support another hop (ProxyJump-like behavior)
- Note that the Windows binary is completely untested!
