## moactl delete user

Delete cluster users

### Synopsis

Delete administrative cluster users.

```
moactl delete user [flags]
```

### Examples

```
  # Delete a user from the cluster-admins group
  moactl delete user --cluster=mycluster --cluster-admins=myusername

  # Delete a user from the dedicated-admins group
  moactl delete user --cluster=mycluster --dedicated-admins=myusername

  # Delete a user following interactive prompts
  moactl delete user --cluster=mycluster
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
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
  -y, --yes       Automatically answer yes to confirm operation.
```

### SEE ALSO

* [moactl delete](moactl_delete.md)	 - Delete a specific resource

