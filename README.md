### terraform-provider-ssh

This provider enables SSH port forwarding in Terraform. It is intended as a
bandaid [until it is supported in Terraform itself](https://github.com/hashicorp/terraform/issues/8367).

*This provider only supports terraform versions `<= 0.11`. It is not compatible with [ Terraform versions 0.12 and above](#terraform-012).*

#### Example

See [main.tf](main.tf).

#### Installation

On Linux:

```shell
mkdir -p terraform.d/plugins/linux_amd64
wget https://github.com/stefansundin/terraform-provider-ssh/releases/download/v0.0.4/terraform-provider-ssh_v0.0.4_linux_amd64.zip
unzip terraform-provider-ssh_v0.0.4_linux_amd64.zip -d terraform.d/plugins/linux_amd64
rm terraform-provider-ssh_v0.0.4_linux_amd64.zip
terraform init
```

On Mac:

```shell
mkdir -p terraform.d/plugins/darwin_amd64
wget https://github.com/stefansundin/terraform-provider-ssh/releases/download/v0.0.4/terraform-provider-ssh_v0.0.4_darwin_amd64.zip
unzip terraform-provider-ssh_v0.0.4_darwin_amd64.zip -d terraform.d/plugins/darwin_amd64
rm terraform-provider-ssh_v0.0.4_darwin_amd64.zip
terraform init
```

#### Applying an output file

Note that there is a gotcha when trying to apply a generated plan output file (see [issue #1](https://github.com/stefansundin/terraform-provider-ssh/issues/1)). In this case, the SSH tunnels will not be automatically opened.

As a workaround, before you apply, run the companion program `terraform-open-ssh-tunnels` on the plan file first in order to reopen the SSH tunnels. [Download from the releases.](https://github.com/stefansundin/terraform-provider-ssh/releases/latest)

Because of [this commit](https://github.com/stefansundin/terraform-provider-ssh/commit/37fa9835b75fde095c863fca89e2f28a0169919d), only the SSH agent is currently supported in this program. Let me know if you can think of a good fix for this.

#### terraform 0.12

This provider relies on a workaround to keep SSH connections open until `terraform` is ready to exit. As of terraform 0.12, provider plugins run in their own process per instance and are destroyed as soon as their resource and data source updates are finished. This provider cannot be updated to add support for Terraform 0.12 and later. For more information, read the [explanation from the Terraform core team](https://discuss.hashicorp.com/t/provider-plugins-that-live-for-the-duration-of-a-terraform-run/2262/2). 

#### TODO

- Support another hop (ProxyJump-like behavior)
- Note that the Windows binary is completely untested!
