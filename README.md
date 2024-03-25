# Red Hat OpenShift Service on AWS (ROSA) Command Line Tool

This project contains the `rosa` command line tool that simplifies the use of Red Hat OpenShift Service on AWS, also known as _ROSA_.

## Quickstart guide

Refer to the official ROSA documentation: https://access.redhat.com/products/red-hat-openshift-service-aws

1. Follow the [AWS Command Line Interface](https://aws.amazon.com/cli/) documentation to install and configure the AWS CLI for your operating system.
2. Download the [latest release of rosa](https://github.com/openshift/rosa/releases/latest) and add it to your path.
3. Initialize your AWS account by running `rosa init` and following the instructions.
4. Create your first ROSA cluster by running `rosa create cluster --interactive`

## Build from source

If you'd like to build this project from source use the following steps:

1. Clone the repository

```
git clone https://github.com/openshift/rosa.git
```

2. `cd` to the checkout out source directory

```
cd rosa
```

3. Install the binary

```
make install
```

NOTE: If you don't have `$GOPATH/bin` in your `$PATH` you need to add it or move `rosa` to a standard system directory eg. for Linux/OSX:

```
sudo mv $GOPATH/bin/rosa /usr/local/bin
```
## Try the ROSA cli from binary

If you don't want to build from sources you can retrieve the `rosa` binary from the latest image.

You can copy it to your local with this command:

```
podman run --pull=always --rm registry.ci.openshift.org/ci/rosa-aws-cli:latest cat /usr/bin/rosa > ~/rosa && chmod +x ~/rosa
```

Also you can test a binary created after a specific merged commit just using the commit hash as image tag:

```
podman run --pull=always --rm registry.ci.openshift.org/ci/rosa-aws-cli:f7925249718111e3e9b61e2df608a6ea9cf5b6ce cat /usr/bin/rosa > ~/rosa && chmod +x ~/rosa
```

NOTE: There is a side-effect of container image registry authentication which results in an [auth error](https://docs.ci.openshift.org/docs/how-tos/use-registries-in-build-farm/#why-i-am-getting-an-authentication-error) when your token is expired even when the image requires no authentication. In that case all you need to do is authenticate again:
```
$ oc registry login
info: Using registry public hostname registry.ci.openshift.org
Saved credentials for registry.ci.openshift.org

$ cat ~/.docker/config.json | jq '.auths["registry.ci.openshift.org"]'
{
  "auth": "token"
}
```
## Secure Credentials Storage
The `OCM_KEYRING` environment variable provides the ability to store the ROSA 
configuration containing your authentication tokens in your OS keyring. This is provided
as an alternative to storing the configuration in plain-text on your system. 
`OCM_KEYRING` will override all other token or configuration related flags.

`OCM_KEYRING` supports the following keyrings:

* [Windows Credential Manager](https://support.microsoft.com/en-us/windows/accessing-credential-manager-1b5c916a-6a16-889f-8581-fc16e8165ac0) - `wincred`
* [macOS Keychain](https://support.apple.com/en-us/guide/keychain-access/welcome/mac) - `keychain`
* Secret Service ([Gnome Keyring](https://wiki.gnome.org/Projects/GnomeKeyring), [KWallet](https://apps.kde.org/kwalletmanager5/), etc.) - `secret-service`
* [Pass](https://www.passwordstore.org/) - `pass`

To ensure `OCM_KEYRING` is provided to all `rosa` commands, it is recommended to set it in your `~/.bashrc` file or equivalent.

| | wincred | keychain | secret-service | pass |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| Windows  | :heavy_check_mark: | :x:  | :x:  | :x:  |
| macOS  | :x:  | :heavy_check_mark:  | :x:  | :heavy_check_mark:  |
| Linux  | :x:  | :x:  | :heavy_check_mark: | :heavy_check_mark: |
## Have you got feedback?

We want to hear it. [Open an issue](https://github.com/openshift/rosa/issues/new) against the repo and someone from the team will be in touch.
