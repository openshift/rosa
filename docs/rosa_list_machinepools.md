## rosa list machinepools

List cluster machine pools

### Synopsis

List machine pools configured on a cluster.

```
rosa list machinepools [flags]
```

### Examples

```
  # List all machine pools on a cluster named "mycluster"
  rosa list machinepools --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to list the machine pools of (required).
  -h, --help             help for machinepools
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa list](rosa_list.md)	 - List all resources of a specific type

