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
```

### Options

```
  -c, --cluster string           Name or ID of the cluster to edit.
      --expiration-time string   Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.
      --expiration duration      Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.
      --compute-nodes int        Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, while multizone clusters need at least 9 nodes (3 per zone) for resiliency.
      --private                  Restrict master API endpoint to direct, private connectivity.
      --enable-cluster-admins    Enable the cluster-admins role for your cluster.
  -h, --help                     help for cluster
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [moactl edit](moactl_edit.md)	 - Edit a specific resource

