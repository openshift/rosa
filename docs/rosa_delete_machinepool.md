## rosa delete machinepool

Delete machine pool

### Synopsis

Delete the additional machine pool from a cluster.

```
rosa delete machinepool ID [flags]
```

### Examples

```
  # Delete machine pool with ID mp-1 from a cluster named 'mycluster'
  rosa delete machinepool --cluster=mycluster mp-1
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete the machine pool from (required).
  -h, --help             help for machinepool
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

