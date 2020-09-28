## moactl init

Applies templates to support Managed OpenShift on AWS clusters

### Synopsis

Applies templates to support Managed OpenShift on AWS clusters. If you are not
yet logged in to OCM, it will prompt you for credentials.

```
moactl init [flags]
```

### Examples

```
  # Configure your AWS account to allow MOA clusters
  moactl init

  # Configure a new AWS account using pre-existing OCM credentials
  moactl init --token=$OFFLINE_ACCESS_TOKEN
```

### Options

```
  -r, --region string          AWS region in which verify quota and permissions (overrides the AWS_REGION environment variable)
      --delete-stack           Deletes stack template applied to your AWS account during the 'init' command.
                               
      --client-id string       OpenID client identifier. The default value is 'cloud-services'.
      --client-secret string   OpenID client secret.
      --env string             Environment of the API gateway. The value can be the complete URL or an alias. The valid aliases are 'production', 'staging' and 'integration'. (default "https://api.openshift.com")
      --insecure               Enables insecure communication with the server. This disables verification of TLS certificates and host names.
      --scope strings          OpenID scope. If this option is used it will replace completely the default scopes. Can be repeated multiple times to specify multiple scopes. (default [openid])
  -t, --token string           Access or refresh token.
      --token-url string       OpenID token URL. The default value is 'https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token'.
  -h, --help                   help for init
```

### Options inherited from parent commands

```
      --debug            Enable debug mode.
      --profile string   Use a specific AWS profile from your credential file.
  -v, --v Level          log level for V logs
```

### SEE ALSO

* [moactl](moactl.md)	 - 

