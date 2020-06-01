## moactl delete cluster

Delete cluster

### Synopsis

Delete cluster.

```
moactl delete cluster [ID|NAME] [flags]
```

### Examples

```
  # Delete a cluster named "mycluster"
  moactl delete cluster mycluster

  # Delete a cluster using the --cluster flag
  moactl delete cluster --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete.
  -h, --help             help for cluster
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl delete](moactl_delete.md)	 - Delete a specific resource

