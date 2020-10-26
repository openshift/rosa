## rosa logs

Show installation or uninstallation logs for a cluster

### Synopsis

Show installation or uninstallation logs for a cluster

### Examples

```
  # Show install logs for a cluster named 'mycluster'
  rosa logs install --cluster=mycluster

  # Show uninstall logs for a cluster named 'mycluster'
  rosa logs uninstall --cluster=mycluster
```

### Options

```
  -h, --help   help for logs
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa](rosa.md)	 - Command line tool for ROSA.
* [rosa logs install](rosa_logs_install.md)	 - Show cluster installation logs
* [rosa logs uninstall](rosa_logs_uninstall.md)	 - Show cluster uninstallation logs

