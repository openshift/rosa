## rosa delete cluster

Delete cluster

### Synopsis

Delete cluster.

```
rosa delete cluster [ID|NAME] [flags]
```

### Examples

```
  # Delete a cluster named "mycluster"
  rosa delete cluster mycluster

  # Delete a cluster using the --cluster flag
  rosa delete cluster --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete.
  -h, --help             help for cluster
      --watch            Watch cluster uninstallation logs.
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
  -y, --yes              Automatically answer yes to confirm operation.
```

### SEE ALSO

* [rosa delete](rosa_delete.md)	 - Delete a specific resource

