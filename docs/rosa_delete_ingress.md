## rosa delete ingress

Delete cluster ingress

### Synopsis

Delete the additional non-default application router for a cluster.

```
rosa delete ingress ID [flags]
```

### Examples

```
  # Delete ingress with ID a1b2 from a cluster named 'mycluster'
  rosa delete ingress --cluster=mycluster a1b2

  # Delete secondary ingress using the sub-domain name
  rosa delete ingress --cluster=mycluster apps2
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete the ingress from (required).
  -h, --help             help for ingress
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

