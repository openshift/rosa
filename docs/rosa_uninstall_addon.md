## rosa uninstall addon

Uninstall add-on from cluster

### Synopsis

Uninstall Red Hat managed add-on from a cluster

```
rosa uninstall addon [flags]
```

### Examples

```
  # Remove the CodeReady Workspaces add-on installation from the cluster
  rosa uninstall addon --cluster=mycluster codeready-workspaces
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

* [rosa uninstall](rosa_uninstall.md)	 - Uninstalls a resource from a cluster

