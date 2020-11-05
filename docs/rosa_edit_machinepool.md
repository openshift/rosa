## rosa edit machinepool

Edit machine pool

### Synopsis

Edit the additional machine pool from a cluster.

```
rosa edit machinepool [flags]
```

### Examples

```
  # Set 4 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --replicas=4 --cluster=mycluster mp1
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the machine pool to (required).
  -h, --help             help for machinepool
      --replicas int     Count of machines for this machine pool (required).
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
  -i, --interactive      Enable interactive mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa edit](rosa_edit.md)	 - Edit a specific resource

