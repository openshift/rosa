## rosa list addons

List add-on installations

### Synopsis

List add-ons installed on a cluster.

```
rosa list addons [flags]
```

### Examples

```
  # List all add-on installations on a cluster named "mycluster"
  rosa list addons --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to list the add-ons of (required).
  -h, --help             help for addons
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa list](rosa_list.md)	 - List all resources of a specific type

