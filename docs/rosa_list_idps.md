## rosa list idps

List cluster IDPs

### Synopsis

List identity providers for a cluster.

```
rosa list idps [flags]
```

### Examples

```
  # List all identity providers on a cluster named "mycluster"
  rosa list idps --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to list the IdP of (required).
  -h, --help             help for idps
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa list](rosa_list.md)	 - List all resources of a specific type

