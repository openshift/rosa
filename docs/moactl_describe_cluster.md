## moactl describe cluster

Show details of a cluster

### Synopsis

Show details of a cluster

```
moactl describe cluster [ID|NAME] [flags]
```

### Examples

```
  # Describe a cluster named "mycluster"
  moactl describe cluster mycluster

  # Describe a cluster using the --cluster flag
  moactl describe cluster --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to describe.
  -h, --help             help for cluster
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl describe](moactl_describe.md)	 - Show details of a specific resource

