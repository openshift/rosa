# IAM Service Account Commands

This document provides comprehensive documentation for the IAM service account management commands in ROSA CLI. These commands allow you to create, manage, and delete IAM roles that can be assumed by Kubernetes service accounts using OpenID Connect (OIDC) identity federation.

## Overview

The IAM service account feature enables Kubernetes pods to assume AWS IAM roles without storing long-lived credentials. This is accomplished using OIDC identity federation, where the cluster's OIDC provider issues tokens that can be exchanged for AWS temporary credentials.

### Prerequisites

- ROSA cluster with STS (Security Token Service) enabled
- Cluster must have an OIDC provider configured
- Appropriate AWS IAM permissions to create and manage roles
- ROSA CLI with AWS credentials configured

### Key Concepts

- **Service Account**: A Kubernetes service account that pods can use to assume an IAM role
- **IAM Role**: An AWS IAM role with a trust policy that allows the OIDC provider to assume it
- **Trust Policy**: Defines which service accounts can assume the role
- **OIDC Provider**: The cluster's OpenID Connect provider used for identity federation

## Commands

### `rosa create iamserviceaccount`

Creates an IAM role that can be assumed by a Kubernetes service account.

#### Usage
```bash
rosa create iamserviceaccount [flags]
```

#### Aliases
- `iam-service-account`

#### Required Flags
- `--cluster, -c`: Name or ID of the cluster
- `--name`: Name of the Kubernetes service account

#### Optional Flags
- `--namespace`: Kubernetes namespace for the service account (default: "default")
- `--role-name`: Name of the IAM role (auto-generated if not specified)
- `--attach-policy-arn`: ARN of IAM policy to attach to the role (can be used multiple times)
- `--inline-policy`: Inline policy document (JSON) or path to policy file (use file://path/to/policy.json)
- `--permissions-boundary`: ARN of IAM policy to use as permissions boundary
- `--path`: IAM path for the role (default: "/")
- `--approve`: Approve operation without confirmation prompt
- `--mode`: Creation mode (auto or manual)

#### Examples

**Basic usage with managed policy:**
```bash
rosa create iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
```

**Custom role name with multiple policies:**
```bash
rosa create iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace my-namespace \
  --role-name my-custom-role \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess
```

**With inline policy from file:**
```bash
rosa create iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --inline-policy file://my-policy.json
```

**With permissions boundary:**
```bash
rosa create iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess \
  --permissions-boundary arn:aws:iam::123456789012:policy/boundary \
  --approve
```

**Manual mode (generate AWS CLI commands):**
```bash
rosa create iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess \
  --mode manual
```

#### Output

**Auto mode**: Creates the role and provides the annotation command:
```
Created IAM role 'my-cluster-default-my-app' with ARN 'arn:aws:iam::123456789012:role/my-cluster-default-my-app'
Attached 1 policies to role
Successfully created IAM service account role

To use this role, annotate your service account:
  oc annotate serviceaccount/my-app -n default eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/my-cluster-default-my-app
```

**Manual mode**: Outputs AWS CLI commands to run manually:
```bash
# Save the trust policy to a file
cat > my-cluster-default-my-app-trust-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.example.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.example.com:sub": "system:serviceaccount:default:my-app"
        }
      }
    }
  ]
}
EOF

aws iam create-role --role-name my-cluster-default-my-app --assume-role-policy-document file://my-cluster-default-my-app-trust-policy.json --tags Key=rosa_role_type,Value=service_account Key=rosa_cluster,Value=my-cluster Key=rosa_namespace,Value=default Key=rosa_service_account,Value=my-app

aws iam attach-role-policy --role-name my-cluster-default-my-app --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
```

---

### `rosa delete iamserviceaccount`

Deletes an IAM role that was created for a Kubernetes service account.

#### Usage
```bash
rosa delete iamserviceaccount [flags]
```

#### Aliases
- `iam-service-account`

#### Required Flags
- `--cluster, -c`: Name or ID of the cluster

#### Optional Flags
- `--name`: Name of the Kubernetes service account
- `--namespace`: Kubernetes namespace for the service account (default: "default")
- `--role-name`: Name of the IAM role to delete (auto-detected if not specified)
- `--approve`: Approve operation without confirmation prompt
- `--mode`: Deletion mode (auto or manual)

#### Examples

**Delete by service account details:**
```bash
rosa delete iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default
```

**Delete by explicit role name:**
```bash
rosa delete iamserviceaccount --cluster my-cluster \
  --role-name my-custom-role --approve
```

**Manual mode (generate AWS CLI commands):**
```bash
rosa delete iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --mode manual
```

#### Output

**Auto mode**: Shows role details and confirms deletion:
```
Role details:
  Name: my-cluster-default-my-app
  ARN: arn:aws:iam::123456789012:role/my-cluster-default-my-app
  Service Account: default/my-app
  Attached Policies: 1
    - arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
Successfully deleted IAM service account role 'my-cluster-default-my-app'
```

**Manual mode**: Outputs AWS CLI commands:
```bash
# Detach managed policies
aws iam detach-role-policy --role-name my-cluster-default-my-app --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

# Delete the role
aws iam delete-role --role-name my-cluster-default-my-app
```

---

### `rosa list iamserviceaccounts`

Lists IAM roles that were created for Kubernetes service accounts.

#### Usage
```bash
rosa list iamserviceaccounts [flags]
```

#### Aliases
- `iam-service-accounts`
- `iamserviceaccount`
- `iam-service-account`

#### Optional Flags
- `--cluster, -c`: Name or ID of the cluster to filter by
- `--namespace`: Kubernetes namespace to filter by
- `--output, -o`: Output format (table, json, yaml)

#### Examples

**List all service account roles:**
```bash
rosa list iamserviceaccounts
```

**List for specific cluster:**
```bash
rosa list iamserviceaccounts --cluster my-cluster
```

**List for specific namespace:**
```bash
rosa list iamserviceaccounts --cluster my-cluster --namespace my-namespace
```

**JSON output:**
```bash
rosa list iamserviceaccounts --output json
```

#### Output

**Table format:**
```
NAME                           ARN                                                    CLUSTER      NAMESPACE    SERVICE ACCOUNT    CREATED
my-cluster-default-my-app      arn:aws:iam::123456789012:role/my-cluster-default-... my-cluster   default      my-app            2024-01-15 10:30:00
my-cluster-kube-system-app     arn:aws:iam::123456789012:role/my-cluster-kube-sys... my-cluster   kube-system  app               2024-01-14 09:15:00
```

**JSON format:**
```json
[
  {
    "roleName": "my-cluster-default-my-app",
    "arn": "arn:aws:iam::123456789012:role/my-cluster-default-my-app",
    "cluster": "my-cluster",
    "namespace": "default",
    "serviceAccount": "my-app",
    "createdDate": "2024-01-15T10:30:00Z",
    "path": "/"
  }
]
```

---

### `rosa describe iamserviceaccount`

Shows detailed information about an IAM role created for a Kubernetes service account.

#### Usage
```bash
rosa describe iamserviceaccount [flags]
```

#### Aliases
- `iam-service-account`

#### Required Flags
- `--cluster, -c`: Name or ID of the cluster

#### Optional Flags
- `--name`: Name of the Kubernetes service account
- `--namespace`: Kubernetes namespace for the service account (default: "default")
- `--role-name`: Name of the IAM role to describe (auto-detected if not specified)
- `--output, -o`: Output format (text, json, yaml)

#### Examples

**Describe by service account details:**
```bash
rosa describe iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default
```

**Describe by explicit role name:**
```bash
rosa describe iamserviceaccount --cluster my-cluster \
  --role-name my-custom-role
```

**JSON output:**
```bash
rosa describe iamserviceaccount --cluster my-cluster \
  --name my-app \
  --namespace default \
  --output json
```

#### Output

**Text format:**
```
Name:                    my-cluster-default-my-app
ARN:                     arn:aws:iam::123456789012:role/my-cluster-default-my-app
Cluster:                 my-cluster
Namespace:               default
Service Account:         my-app
Created:                 2024-01-15 10:30:00 UTC
Path:                    /
Max Session Duration:    3600 seconds
OIDC Provider:           oidc.example.com

Attached Policies:
  - AmazonS3ReadOnlyAccess (arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess)

Tags:
  rosa_role_type: service_account
  rosa_cluster: my-cluster
  rosa_namespace: default
  rosa_service_account: my-app

Trust Policy:
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.example.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.example.com:sub": "system:serviceaccount:default:my-app"
        }
      }
    }
  ]
}
```

**JSON format:**
```json
{
  "roleName": "my-cluster-default-my-app",
  "arn": "arn:aws:iam::123456789012:role/my-cluster-default-my-app",
  "cluster": "my-cluster",
  "namespace": "default",
  "serviceAccount": "my-app",
  "createdDate": "2024-01-15T10:30:00Z",
  "path": "/",
  "maxSessionDuration": 3600,
  "attachedPolicies": [
    {
      "policyName": "AmazonS3ReadOnlyAccess",
      "policyArn": "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"
    }
  ],
  "inlinePolicies": [],
  "trustPolicy": "{...}",
  "oidcProvider": "oidc.example.com",
  "tags": {
    "rosa_role_type": "service_account",
    "rosa_cluster": "my-cluster",
    "rosa_namespace": "default",
    "rosa_service_account": "my-app"
  }
}
```

## Role Naming Convention

When not explicitly specified, IAM role names are automatically generated using the pattern:
```
{cluster-name}-{namespace}-{service-account-name}
```

Examples:
- Cluster: `my-cluster`, Namespace: `default`, Service Account: `my-app` → Role: `my-cluster-default-my-app`
- Cluster: `production`, Namespace: `monitoring`, Service Account: `prometheus` → Role: `production-monitoring-prometheus`

## Trust Policy

The trust policy allows the specified service account to assume the IAM role. The policy includes:

- **Principal**: The cluster's OIDC provider ARN
- **Action**: `sts:AssumeRoleWithWebIdentity`
- **Condition**: Restricts the role to the specific service account

Example trust policy:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.example.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.example.com:sub": "system:serviceaccount:default:my-app"
        }
      }
    }
  ]
}
```

## Using Service Account Roles

After creating an IAM role, the integration method depends on your workload type. ROSA supports multiple approaches for consuming IAM service account roles:

### Method 1: Direct Service Account Annotation

For applications that directly support AWS IAM roles for service accounts (IRSA), annotate the Kubernetes service account:

```bash
oc annotate serviceaccount/my-app -n default \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/my-cluster-default-my-app
```

**Use this method for:**
- Custom applications that use AWS SDKs
- Third-party applications with IRSA support
- Simple workloads that directly consume AWS services

### Method 2: Secret-Based Integration

Many operators require the IAM role ARN to be provided via Kubernetes secrets. This is a common pattern for various operators.

#### Example: Operator with S3 Storage

```bash
# Create secret with role ARN and other required parameters
oc -n <namespace> create secret generic "<secret-name>" \
  --from-literal=bucketnames="${BUCKET_NAME}" \
  --from-literal=region="${AWS_REGION}" \
  --from-literal=audience="openshift" \
  --from-literal=role_arn="${ROLE_ARN}" \
  --from-literal=endpoint="https://s3.${AWS_REGION}.amazonaws.com"
```

Then reference the secret in your custom resource:

```yaml
apiVersion: example.com/v1
kind: ExampleResource
metadata:
  name: example
  namespace: <namespace>
spec:
  storage:
    secret:
      name: <secret-name>
      type: s3
      credentialMode: token  # Uses OIDC token exchange
```

#### Example: Operator with AWS Service Integration

```bash
oc -n <namespace> create secret generic "aws-credentials" \
  --from-literal=AWS_ACCESS_KEY_ID="" \
  --from-literal=AWS_SECRET_ACCESS_KEY="" \
  --from-literal=AWS_ROLE_ARN="${ROLE_ARN}"
```

### Method 3: ConfigMap Integration

Some operators use ConfigMaps for IAM role configuration.

#### Example: Monitoring with Remote Write

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: monitoring-config
  namespace: <namespace>
data:
  config.yaml: |
    remoteWrite:
    - url: "https://remote-endpoint.${AWS_REGION}.amazonaws.com/api/v1/write"
      sigv4:
        region: ${AWS_REGION}
        roleArn: ${ROLE_ARN}
```

### Method 4: CustomResource Integration

Some operators accept the IAM role ARN directly in their CustomResource definitions.

#### Example: Custom Operator

```yaml
apiVersion: example.com/v1
kind: MyCustomResource
metadata:
  name: example
  namespace: my-namespace
spec:
  aws:
    roleArn: arn:aws:iam::123456789012:role/my-cluster-default-my-app
    region: us-east-1
```

### Service-Specific Integration Guides

#### Logging and Observability

1. **Create the IAM role** with appropriate storage and logging service permissions
2. **Create a secret** with the role ARN and storage configuration
3. **Configure your logging operator** to use the secret with `credentialMode: token`

#### Monitoring and Metrics

1. **Create the IAM role** with metrics service permissions
2. **Update monitoring configuration** ConfigMap with the role ARN
3. **Configure remote write** with SigV4 authentication

#### Service Mesh

1. **Create the IAM role** with required AWS service permissions
2. **Configure your service mesh control plane** with AWS-specific settings
3. **Set up secrets or ConfigMaps** as required by the service mesh components

#### GitOps and CI/CD

1. **Create the IAM role** with permissions for your deployment targets
2. **Configure your GitOps operator** with AWS credentials via secrets
3. **Use AWS CLI or SDK** in your workflows

### Troubleshooting Integration Issues

#### Common Problems

1. **Role not being assumed:**
   - Verify the trust policy includes the correct OIDC provider and service account
   - Check that the service account exists and is correctly named
   - Ensure the namespace matches the trust policy condition

2. **Permissions denied:**
   - Verify the role has the necessary AWS service permissions
   - Check if a permissions boundary is blocking required actions
   - Review CloudTrail logs for specific permission failures

3. **Token exchange failures:**
   - Ensure the cluster's OIDC provider is properly configured
   - Verify the audience claim in secrets matches the OIDC configuration
   - Check that `credentialMode: token` is set for applicable resources

4. **Secret not found errors:**
   - Verify the secret exists in the correct namespace
   - Check that the secret has all required keys (role_arn, region, etc.)
   - Ensure the operator has permissions to read the secret

#### Debugging Commands

```bash
# Check OIDC provider configuration
rosa describe cluster --cluster <cluster-name> | grep -A5 "OIDC Endpoint URL"

# Verify service account exists
oc get serviceaccount <sa-name> -n <namespace>

# Check secret contents
oc get secret <secret-name> -n <namespace> -o yaml

# Review operator logs
oc logs -n <operator-namespace> deployment/<operator-name>

# Check AWS IAM role details
rosa describe iamserviceaccount --cluster <cluster> --role-name <role-name>
```

### Best Practices for Integration

1. **Follow operator documentation:** Each operator has specific integration requirements
2. **Use appropriate method:** Choose the integration method that matches your workload type
3. **Validate permissions:** Test IAM permissions before deploying to production
4. **Monitor usage:** Use CloudTrail and CloudWatch to monitor role usage
5. **Rotate regularly:** Plan for periodic role and policy rotation
6. **Document integration:** Keep track of which roles are used by which services

## Tags

Service account roles are automatically tagged with:
- `rosa_role_type`: `service_account`
- `rosa_cluster`: The cluster name
- `rosa_namespace`: The namespace name
- `rosa_service_account`: The service account name

Additional custom tags can be added during role creation.

## Permissions

### Required AWS Permissions

To use these commands, you need the following AWS IAM permissions:

**For creating roles:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:AttachRolePolicy",
        "iam:PutRolePolicy",
        "iam:TagRole",
        "iam:GetRole",
        "iam:ListOpenIDConnectProviders"
      ],
      "Resource": "*"
    }
  ]
}
```

**For deleting roles:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:DeleteRole",
        "iam:DetachRolePolicy",
        "iam:DeleteRolePolicy",
        "iam:GetRole",
        "iam:ListAttachedRolePolicies",
        "iam:ListRolePolicies"
      ],
      "Resource": "*"
    }
  ]
}
```

**For listing and describing:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:ListRoles",
        "iam:GetRole",
        "iam:ListAttachedRolePolicies",
        "iam:ListRolePolicies",
        "iam:GetRolePolicy"
      ],
      "Resource": "*"
    }
  ]
}
```

## Troubleshooting

### Common Issues

**1. "Cluster is not an STS cluster"**
- Ensure your cluster was created with STS enabled
- Verify using: `rosa describe cluster --cluster <name>`

**2. "OIDC provider not found"**
- Check if the cluster has an OIDC provider configured
- For managed OIDC, it should be created automatically
- For unmanaged OIDC, ensure the provider exists in your AWS account

**3. "Role already exists"**
- Use `--approve` flag to continue with existing role
- Or choose a different role name with `--role-name`

**4. "Invalid policy ARN"**
- Verify the policy ARN exists and is accessible
- Check AWS region and account ID in the ARN

**5. "Permission denied"**
- Ensure your AWS credentials have the required IAM permissions
- Check if there are policy restrictions or permission boundaries

### Best Practices

1. **Use descriptive role names** when specifying custom names
2. **Follow least privilege principle** when attaching policies
3. **Use permissions boundaries** in regulated environments
4. **Tag roles appropriately** for cost tracking and governance
5. **Regularly audit** service account roles using the list command
6. **Test role functionality** after creation
7. **Use managed policies** when possible instead of inline policies

### Security Considerations

- Service account roles can only be assumed by the specific service account they were created for
- Trust policies are automatically scoped to the exact service account
- Consider using permissions boundaries for additional security controls
- Regularly review and rotate policies attached to roles
- Monitor role usage through AWS CloudTrail

## Examples by Use Case

### Web Application with S3 Access
```bash
# Create role with S3 read access
rosa create iamserviceaccount --cluster web-cluster \
  --name webapp \
  --namespace production \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

# Annotate service account
oc annotate serviceaccount/webapp -n production \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/web-cluster-production-webapp
```

### Monitoring Application with CloudWatch
```bash
# Create role with CloudWatch permissions
rosa create iamserviceaccount --cluster monitoring-cluster \
  --name prometheus \
  --namespace monitoring \
  --attach-policy-arn arn:aws:iam::aws:policy/CloudWatchReadOnlyAccess \
  --attach-policy-arn arn:aws:iam::aws:policy/EC2InstanceProfileForImageBuilder
```

### Custom Application with Inline Policy
```bash
# Create policy file
cat > custom-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::my-app-bucket/*"
    }
  ]
}
EOF

# Create role with inline policy
rosa create iamserviceaccount --cluster app-cluster \
  --name custom-app \
  --namespace apps \
  --inline-policy file://custom-policy.json
```

This comprehensive documentation covers all aspects of the IAM service account commands, providing users with the information they need to effectively manage service account roles in their ROSA clusters.