## rosa list users

List cluster users

### Synopsis

List administrative cluster users.

```
rosa list users [flags]
```

### Examples

```
  # List all users on a cluster named "mycluster"
  rosa list users --cluster=mycluster
```

### Options

```
  -c, --cluster string   Name or ID of the cluster to list the users of (required).
  -h, --help             help for users
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [rosa list](rosa_list.md)	 - List all resources of a specific type

