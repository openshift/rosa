## rosa edit ingress

Edit the additional cluster ingress

### Synopsis

Edit the additional non-default application router for a cluster.

```
rosa edit ingress [flags]
```

### Examples

```
  # Make additional ingress with ID 'a1b2' private on a cluster named 'mycluster'
  rosa edit ingress --private --cluster=mycluster a1b2

  # Update the router selectors for the additional ingress with ID 'a1b2'
  rosa edit ingress --label-match=foo=bar --cluster=mycluster a1b2

  # Update the default ingress using the sub-domain identifier
  rosa edit ingress --private=false --cluster=mycluster apps
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
      --debug            Enable debug mode.
  -i, --interactive      Enable interactive mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa edit](rosa_edit.md)	 - Edit a specific resource

