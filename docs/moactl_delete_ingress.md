## moactl delete ingress

Delete cluster ingress

### Synopsis

Delete the additional non-default application router for a cluster.

```
moactl delete ingress [flags]
```

### Examples

```
  # Delete ingress with ID a1b2 from a cluster named 'mycluster'
  moactl delete ingress --cluster=mycluster a1b2

  # Delete secondary ingress using the sub-domain name
  moactl delete ingress --cluster=mycluster apps2
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
  -y, --yes       Automatically answer yes to confirm operation.
```

### SEE ALSO

* [moactl delete](moactl_delete.md)	 - Delete a specific resource

