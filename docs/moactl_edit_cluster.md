## moactl edit cluster

Edit cluster

### Synopsis

Edit cluster.

```
moactl edit cluster [flags]
```

### Examples

```
  # Edit a cluster named "mycluster" to make it private
  moactl edit cluster mycluster --private

  # Enable the cluster-admins group using the --cluster flag
  moactl edit cluster --cluster=mycluster --enable-cluster-admins

  # Edit all options interactively
  moactl edit cluster -c mycluster --interactive
```

### Options

```
  -c, --cluster string          Name or ID of the cluster to edit.
      --compute-nodes int       Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, while multizone clusters need at least 9 nodes (3 per zone) for resiliency.
      --private                 Restrict master API endpoint to direct, private connectivity.
      --enable-cluster-admins   Enable the cluster-admins role for your cluster.
  -h, --help                    help for cluster
```

### Options inherited from parent commands

```
      --debug         Enable debug mode.
  -i, --interactive   Enable interactive mode.
  -v, --v Level       log level for V logs
```

### SEE ALSO

* [moactl edit](moactl_edit.md)	 - Edit a specific resource

