## rosa delete admin

Deletes the admin user

### Synopsis

Deletes the cluster-admin user used to login to the cluster

```
rosa delete admin [flags]
```

### Examples

```
  # Delete the admin user
  rosa delete admin --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the IdP to (required).
  -h, --help             help for admin
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
  -y, --yes              Automatically answer yes to confirm operation.
```

### SEE ALSO

* [rosa delete](rosa_delete.md)	 - Delete a specific resource

