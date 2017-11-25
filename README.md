### terraform-provider-ssh

This provider enables SSH port forwarding in Terraform. It is intended as a
bandaid [until it is supported in Terraform itself](https://github.com/hashicorp/terraform/issues/8367).

#### Example

See [main.tf](main.tf).

#### Installation

On Linux:

```shell
mkdir -p terraform.d/plugins/linux_amd64
wget -O terraform.d/plugins/linux_amd64/terraform-provider-ssh.xz https://github.com/stefansundin/terraform-provider-ssh/releases/download/v0.0.1/terraform-provider-ssh_0.0.1_linux_amd64.xz
unxz terraform.d/plugins/linux_amd64/terraform-provider-ssh.xz
chmod +x terraform.d/plugins/linux_amd64/terraform-provider-ssh
terraform init
```

On Mac:

```shell
mkdir -p terraform.d/plugins/darwin_amd64
wget -O terraform.d/plugins/darwin_amd64/terraform-provider-ssh.xz https://github.com/stefansundin/terraform-provider-ssh/releases/download/v0.0.1/terraform-provider-ssh_0.0.1_darwin_amd64.xz
unxz terraform.d/plugins/darwin_amd64/terraform-provider-ssh.xz
chmod +x terraform.d/plugins/darwin_amd64/terraform-provider-ssh
terraform init
```

#### TODO

- Support another hop (ProxyJump-like behavior)
- Note that the Windows binary is completely untested!
