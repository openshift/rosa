## rosa delete user

Delete cluster users

### Synopsis

Delete administrative cluster users.

```
rosa delete user [flags]
```

### Examples

```
  # Delete a user from the cluster-admins group
  rosa delete user --cluster=mycluster --cluster-admins=myusername

  # Delete a user from the dedicated-admins group
  rosa delete user --cluster=mycluster --dedicated-admins=myusername

  # Delete a user following interactive prompts
  rosa delete user --cluster=mycluster
```

### Options

```
  -c, --cluster string            Name or ID of the cluster to delete the users from (required).
      --cluster-admins string     Grant cluster-admin permission to these users.
      --dedicated-admins string   Delete dedicated-admin users.
  -h, --help                      help for user
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

