# Installation troubleshooting

Troubleshooting tips for installing a Red Hat OpenShift Service on AWS cluster using `rosa`.

## Inspect installation logs

Run the following command to generate installation logs:

```
rosa logs
```

## Verify your AWS account does not have an SCP

Run the following command to verify your AWS account has the correct permissions:

```
rosa verify permissions
```
If you receive any errors, double check to ensure than an [SCP](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_type-auth.html#orgs_manage_policies_scp) is not applied to your AWS account. If you are required to use an SCP, see [Red Hat Requirements for Customer Cloud Subscriptions](https://www.openshift.com/dedicated/ccs#scp) for details on the minimum required SCP.

## Verify your AWS account and quota

Run the following command to verify you have the available quota on your AWS account.

```
rosa verify quota
```

If you need to increase your quota, navigate to your [AWS console](https://aws.amazon.com/console/), and request a quota increase for the service that failed.
