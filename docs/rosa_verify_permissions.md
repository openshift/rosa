## rosa verify permissions

Verify AWS permissions are ok for cluster install

### Synopsis

Verify AWS permissions needed to create a cluster are configured as expected

```
rosa verify permissions [flags]
```

### Examples

```
  # Verify AWS permissions are configured correctly
  rosa verify permissions

  # Verify AWS permissions in a different region
  rosa verify permissions --region=us-west-2
```

### Options

```
  -h, --help             help for permissions
      --profile string   Use a specific AWS profile from your credential file.
      --region string    Use a specific AWS region, overriding the AWS_REGION environment variable.
```

### Options inherited from parent commands

```
      --debug     Enable debug mode.
  -v, --v Level   log level for V logs
```

### SEE ALSO

* [rosa verify](rosa_verify.md)	 - Verify resources are configured correctly for cluster install

