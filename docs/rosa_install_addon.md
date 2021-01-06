## rosa install addon

Install add-ons on cluster

### Synopsis

Install Red Hat managed add-ons on a cluster

```
rosa install addon [flags]
```

### Examples

```
  # Add the CodeReady Workspaces add-on installation to the cluster
  rosa install addon --cluster=mycluster codeready-workspaces
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the IdP to (required).
  -h, --help             help for addon
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
  -y, --yes              Automatically answer yes to confirm operation.
```

### SEE ALSO

* [rosa install](rosa_install.md)	 - Installs a resource into a cluster

