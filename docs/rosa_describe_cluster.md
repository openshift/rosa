## rosa describe cluster

Show details of a cluster

### Synopsis

Show details of a cluster

```
rosa describe cluster [flags]
```

### Examples

```
  # Describe a cluster named "mycluster"
  rosa describe cluster mycluster

  # Describe a cluster using the --cluster flag
  rosa describe cluster --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to describe.
  -h, --help             help for cluster
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa describe](rosa_describe.md)	 - Show details of a specific resource

