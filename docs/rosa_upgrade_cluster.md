## rosa upgrade cluster

Upgrade cluster

### Synopsis

Upgrade cluster to a new available version

```
rosa upgrade cluster [flags]
```

### Examples

```
  # Interactively schedule an upgrade on the cluster named "mycluster"
  rosa upgrade cluster --cluster=mycluster --interactive

  # Schedule a cluster upgrade within the hour
  rosa upgade cluster -c mycluster --version 4.5.20
```

### Options

```
  -c, --cluster string                   Name or ID of the cluster to schedule the upgrade for (required)
      --version string                   Version of OpenShift that the cluster will be upgraded to
      --schedule-date string             Next date the upgrade should run at the specified UTC time. Format should be 'yyyy-mm-dd'
      --schedule-time string             Next UTC time that the upgrade should run on the specified date. Format should be 'HH:mm'
      --node-drain-grace-period string   You may set a grace period for how long Pod Disruption Budget-protected workloads will be respected during upgrades.
                                         After this grace period, any workloads protected by Pod Disruption Budgets that have not been successfully drained from a node will be forcibly evicted.
                                         Valid options are ['15 minutes','30 minutes','45 minutes','1 hour','2 hours','4 hours','8 hours'] (default "1 hour")
  -h, --help                             help for cluster
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
  -i, --interactive      Enable interactive mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa upgrade](rosa_upgrade.md)	 - Upgrade a resource

