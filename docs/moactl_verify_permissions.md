## moactl verify permissions

Verify AWS permissions are ok for cluster install

### Synopsis

Verify AWS permissions needed to create a cluster are configured as expected

```
moactl verify permissions [flags]
```

### Examples

```
  # Verify AWS permissions are configured correctly
  moactl verify permissions

  # Verify AWS permissions in a different region
  moactl verify permissions --region=us-west-2
```

### Options

```
  -h, --help   help for permissions
```

### Options inherited from parent commands

```
      --debug           Enable debug mode.
  -r, --region string   AWS region in which to run (overrides the AWS_REGION environment variable)
  -v, --v Level         log level for V logs
```

### SEE ALSO

* [moactl verify](moactl_verify.md)	 - Verify resources are configured correctly for cluster install

