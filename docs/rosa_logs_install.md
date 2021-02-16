## rosa logs install

Show cluster installation logs

### Synopsis

Show cluster installation logs

```
rosa logs install [flags]
```

### Examples

```
  # Show last 100 install log lines for a cluster named "mycluster"
  rosa logs install mycluster --tail=100

  # Show install logs for a cluster using the --cluster flag
  rosa logs install --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to get logs for.
  -h, --help             help for install
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

