## rosa grant user

Grant user access to cluster

### Synopsis

Grant user access to cluster under a specific role

```
rosa grant user ROLE [flags]
```

### Examples

```
  # Add cluster-admin role to a user
  rosa grant user cluster-admin --user=myusername --cluster=mycluster

  # Grant dedicated-admins role to a user
  rosa grant user dedicated-admin --user=myusername --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to add the IdP to (required).
  -h, --help             help for user
  -u, --user string      Username to grant the role to (required).
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
  -i, --interactive      Enable interactive mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa grant](rosa_grant.md)	 - Grant role to a specific resource

