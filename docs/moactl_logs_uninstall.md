## moactl logs uninstall

Show cluster uninstallation logs

### Synopsis

Show cluster uninstallation logs

```
moactl logs uninstall [ID|NAME] [flags]
```

### Examples

```
  # Show last 100 uninstall log lines for a cluster named "mycluster"
  moactl logs uninstall mycluster --tail=100

  # Show uninstall logs for a cluster using the --cluster flag
  moactl logs uninstall --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to get logs for.
  -h, --help             help for uninstall
      --tail int         Number of lines to get from the end of the log. (default 2000)
  -w, --watch            After getting the logs, watch for changes.
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl logs](moactl_logs.md)	 - Show installation logs for a cluster

