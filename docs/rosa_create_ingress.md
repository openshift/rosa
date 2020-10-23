## rosa create ingress

Add Ingress to cluster

### Synopsis

Add an Ingress endpoint to determine API access to the cluster.

```
rosa create ingress [flags]
```

### Examples

```
  # Add an internal ingress to a cluster named "mycluster"
  rosa create ingress --private --cluster=mycluster

  # Add a public ingress to a cluster
  rosa create ingress --cluster=mycluster

  # Add an ingress with route selector label match
  rosa create ingress -c mycluster --label-match="foo=bar,bar=baz"
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

* [rosa create](rosa_create.md)	 - Create a resource from stdin

