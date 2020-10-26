## rosa create user

Configure user access for cluster

### Synopsis

Configure user access for cluster

```
rosa create user [flags]
```

### Examples

```
  # Add a user to the cluster-admins group
  rosa create user --cluster=mycluster --cluster-admins=myusername

  # Add a user to the dedicated-admins group
  rosa create user --cluster=mycluster --dedicated-admins=myusername

  # Add a user following interactive prompts
  rosa create user --cluster=mycluster
```

### Options

```
  -c, --cluster string            Name or ID of the cluster to add the IdP to (required).
      --cluster-admins string     Grant cluster-admin permission to these users.
      --dedicated-admins string   Grant dedicated-admin permission to these users.
  -h, --help                      help for user
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

