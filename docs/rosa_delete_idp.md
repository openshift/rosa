## rosa delete idp

Delete cluster IDPs

### Synopsis

Delete a specific identity provider for a cluster.

```
rosa delete idp [IDP NAME] [flags]
```

### Examples

```
  # Delete an identity provider named github-1
  rosa delete idp github-1 --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete the IdP from (required).
  -h, --help             help for idp
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
  -y, --yes              Automatically answer yes to confirm operation.
```

### SEE ALSO

* [rosa delete](rosa_delete.md)	 - Delete a specific resource

