# MOA Command Line Tool

This project contains the `moactl` command line tool that simplifies the use of Amazon Red Hat OpenShift, also known as _MOA_.

# Quickstart guide

This guide walks through setting up your first Amazon Red Hat OpenShift cluster using `moactl`.

Before you can start working with `moactl` commands you have to have your AWS privileges set up correctly.
To do this run the following command:

```
$ aws iam attach-user-policy --user-name <username> --policy-arn "arn:aws:iam::aws:policy/AdministratorAccess"
```

This tool has a built in possibility to verify that the privileges have been set up correctly.
To do so, run the following command:

```
$ moactl verify permissions
```

After [installing the required prerequisites](#Installation-prerequisites) you can get started with these commands to create a cluster:

```
$ moactl init        ## Configures your AWS account and ensures everything is setup correctly
$ moactl create cluster --name <my-cluster-name>        ## Starts the cluster creation process (~30-40minutes)
$ moactl logs cluster <my-cluster-name> --watch        ## Watch your logs as your cluster creates
$ moactl create idp --cluster <my-cluster-name>  --interactive        ## Connect your IDP to your cluster
$ moactl create user --cluster <my-cluster-name> --dedicated-admins <admin-username>        ## Promotes a user from your IDP to admin level
$ moactl describe cluster <my-cluster-name>        ## Checks if your install is ready (look for State: Ready), and provides your Console URL to login to the web console.
```

If you get stuck or you are starting out and want more details, the rest of this guide includes the following steps:

* Installation prerequisites
* Initializing your AWS account
* Creating your cluster
* Accessing your cluster
* Installing an addon to your cluster
* Creating admin users to your cluster
* Cleaning up

By the end of this guide you will have an Amazon Red Hat OpenShift cluster running in your AWS account.

## Installation prerequisites

Complete the following prerequisites before creating your MOA cluster.

### Select an AWS account to use

Unless your just testing out MOA, we recommend using a dedicated AWS account to run any production clusters. If you are utilizing AWS Organizations, you can use an AWS account within your organization or [create a new one](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_accounts_create.html#orgs_manage_accounts_create-new).

If you are using AWS organizations and you need to have a Service Control Policy (SCP) applied to the AWS account you plan to use, see the [Red Hat Requirements for Customer Cloud Subscriptions](https://www.openshift.com/dedicated/ccs#scp) for details on the minimum required SCP.

As part of the cluster creation process, `moactl` will perform the following actions:

- Create an osdCcsAdmin IAM user:
  - This user will have Programmatic access enabled.
  - This user will have the AdministratorAccess policy attached to it.
  - This user will leverage the IAM credentials you provide when configuring the AWS cli in the next section.


### Install and configure the AWS cli

Install the [aws-cli](https://aws.amazon.com/cli/).

Configure the `aws-cli` to use the AWS account that you would like to deploy your cluster into. Modify your `~/.aws/credentials` file to specify the correct `aws_access_key_id` and `aws_secret_access_key`.

```
$ cat ~/.aws/credentials
[default]
aws_access_key_id = <my-aws-access-key-id>
aws_secret_access_key = <my-aws-secret-access-key>
```

Modify your `~/.aws/config` file to specify the AWS region you want to use.
```
$ cat ~/.aws/config
[default]
output = table
region = us-east-2
```

To verify your configuration, run the following command to query the AWS api:
```
$ aws ec2 describe-regions
---------------------------------------------------------------------------------
|                                DescribeRegions                                |
+-------------------------------------------------------------------------------+
||                                   Regions                                   ||
|+-----------------------------------+-----------------------+-----------------+|
||             Endpoint              |      OptInStatus      |   RegionName    ||
|+-----------------------------------+-----------------------+-----------------+|
||  ec2.eu-north-1.amazonaws.com     |  opt-in-not-required  |  eu-north-1     ||
||  ec2.ap-south-1.amazonaws.com     |  opt-in-not-required  |  ap-south-1     ||
||  ec2.eu-west-3.amazonaws.com      |  opt-in-not-required  |  eu-west-3      ||
||  ec2.eu-west-2.amazonaws.com      |  opt-in-not-required  |  eu-west-2      ||
||  ec2.eu-west-1.amazonaws.com      |  opt-in-not-required  |  eu-west-1      ||
||  ec2.ap-northeast-2.amazonaws.com |  opt-in-not-required  |  ap-northeast-2 ||
||  ec2.ap-northeast-1.amazonaws.com |  opt-in-not-required  |  ap-northeast-1 ||
||  ec2.sa-east-1.amazonaws.com      |  opt-in-not-required  |  sa-east-1      ||
||  ec2.ca-central-1.amazonaws.com   |  opt-in-not-required  |  ca-central-1   ||
||  ec2.ap-southeast-1.amazonaws.com |  opt-in-not-required  |  ap-southeast-1 ||
||  ec2.ap-southeast-2.amazonaws.com |  opt-in-not-required  |  ap-southeast-2 ||
||  ec2.eu-central-1.amazonaws.com   |  opt-in-not-required  |  eu-central-1   ||
||  ec2.us-east-1.amazonaws.com      |  opt-in-not-required  |  us-east-1      ||
||  ec2.us-east-2.amazonaws.com      |  opt-in-not-required  |  us-east-2      ||
||  ec2.us-west-1.amazonaws.com      |  opt-in-not-required  |  us-west-1      ||
||  ec2.us-west-2.amazonaws.com      |  opt-in-not-required  |  us-west-2      ||
|+-----------------------------------+-----------------------+-----------------+|

```

### Install moactl

Download the [latest release of moactl](https://github.com/openshift/moactl/releases/latest) and add it to your path.

Verify your installation by running the following command:

```
$ moactl
Command line tool for MOA.

Usage:
  moactl [command]

Available Commands:
  completion  Generates bash completion scripts
  create      Create a resource from stdin
  delete      Delete a specific resource
  describe    Show details of a specific resource
  edit        Edit a specific resource
  help        Help about any command
  init        Applies templates to support Managed OpenShift on AWS clusters
  list        List all resources of a specific type
  login       Log in to your Red Hat account
  logout      Log out
  logs        Show logs of a specific resource
  verify      Verify resources are configured correctly for cluster install
  version     Prints the version of the tool

Flags:
      --debug     Enable debug mode.
  -h, --help      help for moactl
  -v, --v Level   log level for V logs

Use "moactl [command] --help" for more information about a command.
```

You can run `moactl completion` to generate a bash completion file. Add this file to the correct location for your operating system. For example, on a Linux machine run the following command to enable moactl bash completion:

```
moactl completion > /etc/bash_completion.d/moactl
source /etc/bash_completion.d/moactl
```

Run the following command to verify that your AWS account has the necessary permissions:

```
$ moactl verify permissions
I: Validating SCP policies...
I: AWS SCP policies ok
```

Verify that your AWS account has the necessary quota to deploy an OpenShift cluster. Sometimes quota varies by region, which may prevent you from deploying:

```
$ export AWS_DEFAULT_REGION=us-west-2 && moactl verify quota
I: Validating AWS quota...
E: Insufficient AWS quotas
E: Service ebs quota code L-FD252861 Provisioned IOPS SSD (io1) volume storage not valid
```

If needed, try another region:
```
$ export AWS_DEFAULT_REGION=us-east-2 && moactl verify quota
I: Validating AWS quota...
I: AWS quota ok
```

If you need to increase your quota, navigate to your [AWS console](https://aws.amazon.com/console/), and request a quota increase for the service that failed.

Once both the permissions and quota checks pass, proceed to initializing your AWS account.

## Initializing your AWS account

In this step you log in to your Red Hat account using `moactl`, and then initialize your AWS account.

### Log in to your Red Hat account with moactl

If you do not already have a Red Hat account, [create one here](https://cloud.redhat.com/). Be sure to accept the required terms and conditions. Then, check your email for a verification link.  

After creating your Red Hat account, follow this link to [get an offline access token](https://cloud.redhat.com/openshift/token/moa
).

Run the following command to log in to your Red Hat account with moactl. Replace &lt;my-offline-access-token&gt; with your token:

```
$ moactl login --token="<my-offline-access-token>"
```

### Verify moactl login and aws-cli defaults

Run the following command to verify your Red Hat and AWS credentials are setup correctly.  Check that your AWS Account ID, Default Region and ARN match what you expect.  You can safely ignore the rows beginning with OCM for now (OCM stands for OpenShift Cluster Manager).

```
$ moactl whoami
AWS Account ID:               000000000000
AWS Default Region:           us-east-2
AWS ARN:                      arn:aws:iam::000000000000:user/hello
OCM API:                      https://api.openshift.com
OCM Account ID:               1DzGIdIhqEWyt8UUXQhSoWaaaaa
OCM Account Name:             Your Name
OCM Account Username:         you@domain.com
OCM Account Email:            you@domain.com
OCM Organization ID:          1HopHfA2hcmhup5gCr2uH5aaaaa
OCM Organization Name:        Red Hat
OCM Organization External ID: 0000000
```

### Initialize your AWS account

This step runs a CloudFormation template that prepares your AWS account for OpenShift deployment and management. This step typically takes 1-2 minutes to complete.

```
$ moactl init
I: Logged in as 'rh-moa-user' on 'https://api.openshift.com'
I: Validating AWS credentials...
I: AWS credentials are valid!
I: Validating SCP policies...
I: AWS SCP policies ok
I: Validating AWS quota...
I: AWS quota ok
I: Ensuring cluster administrator user 'osdCcsAdmin'...
I: Admin user 'osdCcsAdmin' created successfuly!
I: Verifying whether OpenShift command-line tool is available...
E: OpenShift command-line tool is not installed.
Go to https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/ to download the OpenShift client and add it to your PATH.
```

> NOTE
> If you have not already installed the OpenShift Command Line Utility, also known as `oc`, follow the link in the output to install it now.


## Create your cluster

Run the following command to create your cluster with the default cluster settings.

> To view other options when creating a cluster run `moactl create cluster --help`.
> To follow a set of interactive prompts run `moactl create cluster --interactive`.

```
$ moactl create cluster --name=rh-moa-test-cluster1
I: Creating cluster with identifier '1de87g7c30g75qechgh7l5b2bha6r04e' and name 'rh-moa-test-cluster1'
I: To view list of clusters and their status, run `moactl list clusters`
I: Cluster 'rh-moa-test-cluster1' has been created.
I: Once the cluster is 'Ready' you will need to add an Identity Provider and define the list of cluster administrators. See `moactl create idp --help` and `moactl create user --help` for more information.
I: To determine when your cluster is Ready, run `moactl describe cluster rh-moa-test-cluster1`.
```

Creating a cluster can take up to 40 minutes, during which the State will transition from `pending` to `installing`, and finally to `ready`.

Run the following command to check the status of your MOA cluster:

```
$ moactl describe cluster rh-moa-test-cluster1
Name:        rh-moa-test-cluster1
ID:          1de87g7c30g75qechgh7l5b2bha6r04e
External ID: 34322be7-b2a7-45c2-af39-2c684ce624e1
API URL:     https://api.rh-moa-test-cluster1.j9n4.s1.devshift.org:6443
Console URL: https://console-openshift-console.apps.rh-moa-test-cluster1.j9n4.s1.devshift.org
Nodes:       Master: 3, Infra: 3, Compute: 4
Region:      us-east-2
State:       ready
Created:     May 27, 2020
```

If installation fails or the State does not change to `ready` after 40 minutes, check the [installation troubleshooting](install-troubleshooting.md) documentation for more details.

You can follow the OpenShift installer logs to track the progress of your cluster:
```
moactl logs cluster rh-moa-test-cluster1 --watch
```

## Access your cluster

To login to your cluster, you must configure an Identity Provider (IDP).

For this guide we will use Github as an example IDP. 

For other supported IDPs, run `moactl create idp --help`, and consult the OpenShift documentation on [configuring an IDP](https://docs.openshift.com/container-platform/latest/authentication/understanding-identity-provider.html#supported-identity-providers) for more information.

### Add an IDP

Run the following command to create an IDP backed by Github. Follow the prompts from the output to access your [Github developer settings](https://github.com/settings/developers) and configure a new OAuth application.

```
$ moactl create idp --cluster rh-moa-test-cluster1 --type github                                                                                               
I: Loading cluster 'rh-moa-test-cluster1'                                                                                                                                                               
I: Loading identity providers for cluster 'rh-moa-test-cluster1'                                                                                                                                        
To use GitHub as an identity provider, you must first register the application:                                                                                                                 
? List of GitHub organizations or teams that will have access to this cluster: openshift-online                                                                                                 
* Open the following URL: https://github.com/organizations/openshift-online/settings/applications/new?oauth_application%5Bcallback_url%5D=https%3A%2F%2Foauth-openshift.apps.rh-moa-test-cluster1.j9n4.s
1.devshift.org%2Foauth2callback%2Fgithub-1&oauth_application%5Bname%5D=rh-moa-test-cluster1&oauth_application%5Burl%5D=https%3A%2F%2Fconsole-openshift-console.apps.rh-moa-test-cluster1.j9n4.s1.devshift.org   
* Click on 'Register application'                                                                                                                                                               
? Copy the Client ID provided by GitHub: &lt;my-github-client-id&gt;                                                                                                                                   
? Copy the Client Secret provided by GitHub: &lt;my-github-client-secret&gt;                                                                                                           
I: Configuring IDP for cluster 'rh-moa-test-cluster1'                                                                                                                                                   
I: Identity Provider 'github-1' has been created. You need to ensure that there is a list of cluster administrators defined. See `moactl user add --help` for more information. To login into th
e console, open https://console-openshift-console.apps.rh-moa-test-cluster1.j9n4.s1.devshift.org and click on github-1
```

The IDP can take 1-2 minutes to be configured within your cluster.

Run the following command to verify that your IDP has been configured correctly:

```
$ moactl list idps --cluster rh-moa-test-cluster1
NAME        TYPE      AUTH URL
github-1    GitHub    https://oauth-openshift.apps.rh-moa-test-cluster1.j9n4.s1.devshift.org/oauth2callback/github-1
```

### Log in to your cluster

At this point you should be able to login to your cluster using your Github ID.

First, run the following command to get the `Console URL` of your cluster:

```
$ moactl describe cluster rh-moa-test-cluster1
Name:        rh-moa-test-cluster1
ID:          1de87g7c30g75qechgh7l5b2bha6r04e
External ID: 34322be7-b2a7-45c2-af39-2c684ce624e1
API URL:     https://api.rh-moa-test-cluster1.j9n4.s1.devshift.org:6443
Console URL: https://console-openshift-console.apps.rh-moa-test-cluster1.j9n4.s1.devshift.org
Nodes:       Master: 3, Infra: 3, Compute: 4
Region:      us-east-2
State:       ready
Created:     May 27, 2020
```

Navigate to the `Console URL` and log in using your Github credentials.

Once you are logged into your cluster, follow these steps to get your `oc` login command. In the top right of the OpenShift console, click your name and click **Copy Login Command**.  Click **github-1** and finally click **Display Token**. Copy and paste the `oc` login command into your terminal.

```
$ oc login --token=z3sgOGVDk0k4vbqo_wFqBQQTnT-nA-nQLb8XEmWnw4X --server=https://api.rh-moa-test-cluster1.j9n4.s1.devshift.org:6443
Logged into "https://api.rh-moa-test-cluster1.j9n4.s1.devshift.org:6443" as "rh-moa-test-user" using the token provided.

You have access to 67 projects, the list has been suppressed. You can list all projects with 'oc projects'

Using project "default".
```

Run a simple `oc` command to verify everything is setup properly and you are logged in.

```
$ oc version
Client Version: 4.4.0-202005231254-4a4cd75
Server Version: 4.3.18
Kubernetes Version: v1.16.2
```

## (Optional) Addons example

(coming soon)


## (Optional) Create dedicated and cluster admins

### Create a dedicated-admin user

Run the following command to promote your Github user to dedicated-admin:

```
$ moactl create user --cluster rh-moa-test-cluster1 --dedicated-admins=rh-moa-test-user
```

Run the following command to verify your user now has dedicated-admin access. As a dedicated-admin you should receive some errors when running the following command:

```
$ oc get all -n openshift-apiserver
NAME                  READY   STATUS    RESTARTS   AGE
pod/apiserver-6ndg2   1/1     Running   0          17h
pod/apiserver-lrmxs   1/1     Running   0          17h
pod/apiserver-tsqhz   1/1     Running   0          17h
NAME          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/api   ClusterIP   172.30.23.241   <none>        443/TCP   17h
NAME                       DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                     AGE
daemonset.apps/apiserver   3         3         3       3            3           node-role.kubernetes.io/master=   17h
Error from server (Forbidden): horizontalpodautoscalers.autoscaling is forbidden: User "rh-moa-test-user" cannot list resource "horizontalpodautoscalers" in API group "autoscaling" in the namespace 
"openshift-apiserver"
Error from server (Forbidden): jobs.batch is forbidden: User "rh-moa-test-user" cannot list resource "jobs" in API group "batch" in the namespace "openshift-apiserver"
Error from server (Forbidden): cronjobs.batch is forbidden: User "rh-moa-test-user" cannot list resource "cronjobs" in API group "batch" in the namespace "openshift-apiserver"
Error from server (Forbidden): imagestreams.image.openshift.io is forbidden: User "rh-moa-test-user" cannot list resource "imagestreams" in API group "image.openshift.io" in the namespace "openshift
-apiserver"
```

### Create a cluster-admin user

To add a cluster-admin user, first enable cluster-admin capability on the cluster:

```
$ moactl edit cluster rh-moa-test-cluster1 --enable-cluster-admins
```

Next give your user cluster-admin privileges:

```
$ moactl create user --cluster rh-moa-test-cluster1 --cluster-admins rh-moa-test-user
$ moactl list users --cluster rh-moa-test-cluster1
GROUP             NAME
cluster-admins    rh-moa-test-user
dedicated-admins  rh-moa-test-user
```

Run the following command to verify your user now has cluster-admin access. As a cluster-admin you should be able to run the following command without errors.

```
$ oc get all -n openshift-apiserver                       
NAME                  READY   STATUS    RESTARTS   AGE
pod/apiserver-6ndg2   1/1     Running   0          17h
pod/apiserver-lrmxs   1/1     Running   0          17h
pod/apiserver-tsqhz   1/1     Running   0          17h
NAME          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/api   ClusterIP   172.30.23.241   <none>        443/TCP   18h
NAME                       DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                     AGE
daemonset.apps/apiserver   3         3         3       3            3           node-role.kubernetes.io/master=   18h
```

## Next steps

After installing your cluster you can move on to installing an example app, or clean up if you are just giving MOA a test drive.

### Deleting your cluster

Run the following command to delete your cluster, replacing &lt;my-cluster&gt; with the name of your cluster:

```
moactl delete cluster -c <my-cluster>
```

To clean up your CloudFormation stack (this was created when you ran `moactl init`), run the following command:

```
moactl init --delete-stack
```

## Have you got feedback?

We want to hear it. [Open and issue](https://github.com/openshift/moactl/issues/new) against the repo and someone from the team will be in touch.
