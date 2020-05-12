## moactl delete ingress

Delete the additional cluster ingress

### Synopsis

Delete the additional non-default application router for a cluster.

```
moactl delete ingress [flags]
```

### Examples

```
  # Delete ingress for a cluster named 'mycluster'
  moactl delete ingress --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete the ingress from (required).
  -h, --help             help for ingress
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl delete](moactl_delete.md)	 - Delete a specific resource

