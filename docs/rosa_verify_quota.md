## rosa verify quota

Verify AWS quota is ok for cluster install

### Synopsis

Verify AWS quota needed to create a cluster is configured as expected

```
rosa verify quota [flags]
```

### Examples

```
  # Verify AWS quotas are configured correctly
  rosa verify quota

  # Verify AWS quotas in a different region
  rosa verify quota --region=us-west-2
```

### Options

```
  -h, --help             help for quota
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

