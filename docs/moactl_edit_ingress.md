## moactl edit ingress

Edit the additional cluster ingress

### Synopsis

Edit the additional non-default application router for a cluster.

```
moactl edit ingress [flags]
```

### Examples

```
  # Make additional ingress private on a cluster named 'mycluster'
  moactl edit ingress --private --cluster=mycluster

  # Update the router selectors for the additional ingress
  moactl edit ingress --label-match=foo=bar --cluster=mycluster
```

### Options

```
  -c, --cluster string       Name or ID of the cluster to add the ingress to (required).
  -h, --help                 help for ingress
      --label-match string   Label match for ingress. Format should be a comma-separated list of 'key=value'. If no label is specified, all routes will be exposed on both routers.
      --private              Restrict application route to direct, private connectivity.
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl edit](moactl_edit.md)	 - Edit a specific resource

