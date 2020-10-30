## rosa revoke user

Revoke role from users

### Synopsis

Revoke role from cluster user

```
rosa revoke user ROLE [flags]
```

### Examples

```
  # Revoke cluster-admin role from a user
  rosa revoke user cluster-admins --user=myusername --cluster=mycluster

  # Revoke dedicated-admin role from a user
  rosa revoke user dedicate-admins --user=myusername --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to delete the users from (required).
  -h, --help             help for user
  -u, --user string      Username to revoke the role from (required).
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
  -y, --yes              Automatically answer yes to confirm operation.
```

### SEE ALSO

* [rosa revoke](rosa_revoke.md)	 - Revoke role from a specific resource

