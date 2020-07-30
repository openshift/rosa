## moactl logs

Show cluster installation logs

### Synopsis

Show cluster installation logs

```
moactl logs [ID|NAME] [flags]
```

### Examples

```
  # Show last 100 log lines for a cluster named "mycluster"
  moactl logs mycluster --tail=100

  # Show logs for a cluster using the --cluster flag
  moactl logs --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to get logs for.
  -h, --help             help for logs
      --tail int         Number of lines to get from the end of the log. (default 2000)
  -w, --watch            After getting the logs, watch for changes.
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl](moactl.md)	 - 

