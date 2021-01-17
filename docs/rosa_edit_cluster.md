## rosa edit cluster

Edit cluster

### Synopsis

Edit cluster.

```
rosa edit cluster [flags]
```

### Examples

```
  # Edit a cluster named "mycluster" to make it private
  rosa edit cluster mycluster --private

  # Edit all options interactively
  rosa edit cluster -c mycluster --interactive
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to edit.
      --private          Restrict master API endpoint to direct, private connectivity.
  -h, --help             help for cluster
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

