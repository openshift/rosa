## moactl logs cluster

Show details of a cluster

### Synopsis

Show details of a cluster

```
moactl logs cluster [ID|NAME] [flags]
```

### Examples

```
  # Show last 100 log lines for a cluster named "mycluster"
  moactl logs cluster mycluster --tail=100

  # Show logs for a cluster using the --cluster flag
  moactl logs cluster --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to get logs for.
  -h, --help             help for cluster
      --tail int         Number of lines to get from the end of the log. (default 2000)
  -w, --watch            After getting the logs, watch for changes.
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl logs](moactl_logs.md)	 - Show logs of a specific resource

