## rosa create admin

Creates an admin user to login to the cluster

### Synopsis

Creates a cluster-admin user with an auto-generated password to login to the cluster

```
rosa create admin [flags]
```

### Examples

```
  # Create an admin user to login to the cluster
  rosa create admin --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the IdP to (required).
  -h, --help             help for admin
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

