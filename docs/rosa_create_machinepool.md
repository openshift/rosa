## rosa create machinepool

Add machine pool to cluster

### Synopsis

Add a machine pool to the cluster.

```
rosa create machinepool [flags]
```

### Examples

```
  # Interactively add a machine pool to a cluster named "mycluster"
  rosa create machinepool --cluster=mycluster --interactive

  # Add a machine pool mp-1 with 3 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --replicas=3 --instance-type=m5.xlarge

  # Add a machine pool with labels to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --labels =foo=bar,bar=baz"
```

### Options

```
  -c, --cluster string         Name or ID of the cluster to add the machine pool to (required).
  -h, --help                   help for machinepool
      --instance-type string   Instance type that should be used. (default "m5.xlarge")
      --labels string          Labels for machine pool. Format should be a comma-separated list of 'key=value'. This list will overwrite any modifications made to Node labels on an ongoing basis.
      --name string            Name for the machine pool (required).
      --replicas int           Count of machines for this machine pool (required).
      --taints string          Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. This list will overwrite any modifications made to Node taints on an ongoing basis.
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

