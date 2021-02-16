## rosa describe admin

Show details of the cluster-admin user

### Synopsis

Show details of the cluster-admin user and a command to login to the cluster

```
rosa describe admin [flags]
```

### Examples

```
  # Describe cluster-admin user of a cluster named mycluster
  rosa describe admin -c mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster that cluster-admin belongs to.
  -h, --help             help for admin
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa describe](rosa_describe.md)	 - Show details of a specific resource

