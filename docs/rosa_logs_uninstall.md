## rosa logs uninstall

Show cluster uninstallation logs

### Synopsis

Show cluster uninstallation logs

```
rosa logs uninstall [ID|NAME] [flags]
```

### Examples

```
  # Show last 100 uninstall log lines for a cluster named "mycluster"
  rosa logs uninstall mycluster --tail=100

  # Show uninstall logs for a cluster using the --cluster flag
  rosa logs uninstall --cluster=mycluster
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
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa logs](rosa_logs.md)	 - Show installation or uninstallation logs for a cluster

