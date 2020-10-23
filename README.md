# ROSA Command Line Tool

This project contains the `rosa` command line tool that simplifies the use of Red Hat OpenShift Service on AWS.

# Quickstart guide

This guide walks through setting up your first Red Hat OpenShift Service on AWS cluster using `rosa`.

If you have already [installed the required prerequisites](#Installation-prerequisites), here are the commands you need to create a cluster.

```
$ rosa init        ## Configures your AWS account and ensures everything is setup correctly
$ rosa create cluster --cluster-name <my-cluster-name>        ## Starts the cluster creation process (~30-40minutes)
$ rosa logs install -c <my-cluster-name> --watch        ## Watch your logs as your cluster creates
$ rosa create idp --cluster <my-cluster-name>  --interactive        ## Connect your IDP to your cluster
$ rosa create user --cluster <my-cluster-name> --dedicated-admins <admin-username>        ## Promotes a user from your IDP to admin level
$ rosa describe cluster <my-cluster-name>        ## Checks if your install is ready (look for State: Ready), and provides your Console URL to login to the web console.
```

If you get stuck or you are starting out and want more details, the rest of this guide includes the following steps:

* [Installation prerequisites](#Installation-prerequisites)
* [Preparing your AWS account for cluster installation](#preparing-your-aws-account-for-cluster-installation)
* [Creating your cluster](#creating-your-cluster)
* [Accessing your cluster](#accessing-your-cluster)
* [Creating admin users for your cluster](#optional-create-dedicated-and-cluster-admins)
* [Cleaning up](#next-steps)

By the end of this guide you will have an Red Hat OpenShift Service on AWS cluster running in your AWS account.

## Installation prerequisites

Complete the following prerequisites before creating your MOA cluster.

### Select an AWS account to use

Unless you are just testing out MOA, we recommend using a dedicated AWS account to run any production clusters. If you are utilizing AWS Organizations, you can use an AWS account within your organization or [create a new one](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_accounts_create.html#orgs_manage_accounts_create-new).

If you are using AWS organizations and you need to have a Service Control Policy (SCP) applied to the AWS account you plan to use, see the [Red Hat Requirements for Customer Cloud Subscriptions](https://www.openshift.com/dedicated/ccs#scp) for details on the minimum required SCP.

As part of the cluster creation process, `rosa` will create an osdCcsAdmin IAM user. This user will have Programmatic access enabled and have the AdministratorAccess policy attached to it. The AWS credentials provided in the next section will be used to create this user.

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

> NOTE
> If you are not using the default profile for your AWS credentials, run the following command to 
export your profile settings to your shell, replacing `<my-profile>` with the name of your AWS profile: `export AWS_PROFILE=<my-profile>`

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

### Install rosa

Download the [latest release of rosa](https://github.com/openshift/rosa/releases/latest) and add it to your path.

Verify your installation by running the following command:

```
$ rosa
Command line tool for MOA.

Usage:
  rosa [command]

Available Commands:
  completion  Generates bash completion scripts
  create      Create a resource from stdin
  delete      Delete a specific resource
  describe    Show details of a specific resource
  download    Download necessary tools for using your cluster
  edit        Edit a specific resource
  help        Help about any command
  init        Applies templates to support Managed OpenShift on AWS clusters
  list        List all resources of a specific type
  login       Log in to your Red Hat account
  logout      Log out
  logs        Show installation or uninstallation logs for a cluster
  verify      Verify resources are configured correctly for cluster install
  version     Prints the version of the tool
  whoami      Displays user account information

Flags:
      --debug     Enable debug mode.
  -h, --help      help for rosa
  -v, --v Level   log level for V logs

Use "rosa [command] --help" for more information about a command.
```

You can add rosa bash completion to your current terminal session by running the following commands:

```
$ rosa completion > ~/.rosa-completion.sh
$ source ~/.rosa-completion.sh
```

If you want to persist rosa bash completion on new terminal sessions, add the output from `rosa completion` to your `.bashrc` file, or to the appropriate location for your operating system.

Run the following command to verify that your AWS account has the necessary permissions:

```
$ rosa verify permissions

I: Validating SCP policies...
I: AWS SCP policies ok
```

Verify that your AWS account has the necessary quota to deploy an OpenShift cluster. Sometimes quota varies by region, which may prevent you from deploying:

```
$ rosa verify quota --region=us-west-2

I: Validating AWS quota...
E: Insufficient AWS quotas
E: Service ec2 quota code L-0263D0A3 Number of EIPs - VPC EIPs not valid, expected quota of at least 5, but got 2
```

If needed, try another region:
```
$ rosa verify quota --region=us-east-2

I: Validating AWS quota...
I: AWS quota ok
```

If you need to increase your quota, navigate to your [AWS console](https://aws.amazon.com/console/), and request a quota increase for the service that failed.

Once both the permissions and quota checks pass, proceed to preparing your AWS account for cluster installation.

## Preparing your AWS account for cluster installation

In this step you log in to your Red Hat account using `rosa`, and then initialize your AWS account.

### Log in to your Red Hat account with rosa

If you do not already have a Red Hat account, [create one here](https://cloud.redhat.com/). Be sure to accept the required terms and conditions. Then, check your email for a verification link.  

After creating your Red Hat account, follow this link to [get an offline access token](https://cloud.redhat.com/openshift/token/moa
).

Run the following command to log in to your Red Hat account with rosa. Replace &lt;my-offline-access-token&gt; with your token:

```
$ rosa login --token="<my-offline-access-token>"
```

### Verify rosa login and aws-cli defaults

Run the following command to verify your Red Hat and AWS credentials are setup correctly.  Check that your AWS Account ID, Default Region, and ARN match what you expect.  You can safely ignore the rows beginning with OCM for now (OCM stands for OpenShift Cluster Manager).

```
$ rosa whoami

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
$ rosa init

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
W: OpenShift command-line tool is not installed.
Run 'rosa download oc' to download the latest version, then add it to your PATH.
```

> NOTE
> If you have not already installed the OpenShift Command Line Utility, also known as `oc`, run the command in the output to download it now.

## Creating your cluster

To view all of the available options when creating a cluster, run the following command:

```
$ rosa create cluster --help
Create cluster.

Usage:
  rosa create cluster [flags]

Examples:
  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2

Flags:
  -c, --cluster-name string           Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.
      --multi-az                      Deploy to multiple data centers.
  -r, --region string                 AWS region where your worker pool will be located. (overrides the AWS_REGION environment variable)
      --version string                Version of OpenShift that will be used to install the cluster, for example "4.3.10"
      --channel-group string          Channel group is the name of the group where this image belongs, for example "stable" or "fast". (default "stable")
      --compute-machine-type string   Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.
      --compute-nodes int             Number of worker nodes to provision per zone. Single zone clusters need at least 2 nodes, multizone clusters need at least 3 nodes. (default 2)
      --machine-cidr ipNet            Block of IP addresses used by OpenShift while installing the cluster, for example "10.0.0.0/16".
      --service-cidr ipNet            Block of IP addresses for services, for example "172.30.0.0/16".
      --pod-cidr ipNet                Block of IP addresses from which Pod IP addresses are allocated, for example "10.128.0.0/14".
      --host-prefix int               Subnet prefix length to assign to each individual node. For example, if host prefix is set to "23", then each node is assigned a /23 subnet out of the given CIDR.
      --private                       Restrict master API endpoint and application routes to direct, private connectivity.
      --disable-scp-checks            Indicates if cloud permission checks are disabled when attempting installation of the cluster.
      --watch                         Watch cluster installation logs.
      --dry-run                       Simulate creating the cluster.
  -h, --help                          help for cluster

Global Flags:
      --debug         Enable debug mode.
  -i, --interactive   Enable interactive mode.
  -v, --v Level       log level for V logs
```

You can step through each of these options interactively by using the `--interactive` flag.

```
$ rosa create cluster --interactive
```

Otherwise, run the following command to create your cluster with the default cluster settings. The default settings are as follows:

* The AWS region you have configured for the AWS CLI
* The most recent version of OpenShift available to rosa
* A single availability zone
* Public cluster (Public API)
* Master nodes: 3
* Infra nodes: 2
* Compute nodes: 2 (m5.xlarge instance types)

```
$ rosa create cluster --cluster-name=rh-moa-test-cluster

I: Creating cluster with identifier '1de87g7c30g75qechgh7l5b2bha6r04e' and name 'rh-moa-test-cluster'
I: To view list of clusters and their status, run `rosa list clusters`
I: Cluster 'rh-moa-test-cluster' has been created.
I: Once the cluster is 'Ready' you will need to add an Identity Provider and define the list of cluster administrators. See `rosa create idp --help` and `rosa create user --help` for more information.
I: To determine when your cluster is Ready, run `rosa describe cluster rh-moa-test-cluster`.
```

Creating a cluster can take up to 40 minutes, during which the State will transition from `pending` to `installing`, and finally to `ready`.

After creating a cluster, run the following command to list all available clusters:

```
$ rosa list clusters

ID                                NAME                    STATE
1eids212dg6tkkr231t1sl25reskq0q7  rh-moa-test-cluster     pending
```

Run the following command to see more details and check the status of a specific cluster. Replace `<my-cluster-name>` with the name of your cluster.

```
$ rosa describe cluster <my-cluster-name>

Name:        rh-moa-test-cluster
ID:          1de87g7c30g75qechgh7l5b2bha6r04e
External ID: 34322be7-b2a7-45c2-af39-2c684ce624e1
API URL:     https://api.rh-moa-test-cluster.j9n4.s1.devshift.org:6443
Console URL: https://console-openshift-console.apps.rh-moa-test-cluster.j9n4.s1.devshift.org
Nodes:       Master: 3, Infra: 2, Compute: 2
Region:      us-east-2
State:       ready
Created:     May 27, 2020
```

If installation fails or the State does not change to `ready` after 40 minutes, check the [installation troubleshooting](install-troubleshooting.md) documentation for more details.

You can follow the installer logs to track the progress of your cluster:

```
rosa logs install -c rh-moa-test-cluster --watch
```

## Accessing your cluster

To log in to your cluster, you must configure an Identity Provider (IDP).

For this guide we will use GitHub as an example IDP.

For other supported IDPs, run `rosa create idp --help`, and consult the OpenShift documentation on [configuring an IDP](https://docs.openshift.com/container-platform/latest/authentication/understanding-identity-provider.html#supported-identity-providers) for more information.

### Add an IDP

The following command to creates an IDP backed by GitHub. Follow the interactive prompts from the output to access your [Github developer settings](https://github.com/settings/developers) and configure a new OAuth application.

Here are the options we will configure and the values to select:
* Type of identity provider: github
* Restrict to members of: organizations (if you do not have a GitHub Organization, you can [create one now](https://docs.github.com/en/github/setting-up-and-managing-organizations-and-teams/creating-a-new-organization-from-scratch).)
* GitHub organizations: rh-test-org (enter the name of your org)

Follow the URL from the output. This will create a new OAuth application in the GitHub organization you specified. Click *Register applicaton* to access your Client ID and Client Secret.

* Client ID: &lt;my-github-client-id&gt;
* Client Secret: [? for help] &lt;my-github-client-secret&gt;
* Hostname: (optional, you can leave it blank for now)
* Mapping method: claim

```
$ rosa create idp --cluster=rh-moa-test-cluster --interactive
I: Interactive mode enabled.
Any optional fields can be left empty and a default will be selected.
? Type of identity provider: github
? Restrict to members of: organizations
? GitHub organizations: rh-test-org
? To use GitHub as an identity provider, you must first register the application:
  - Open the following URL:
    https://github.com/organizations/rh-moa-test-cluster/settings/applications/new?oauth_application%5Bcallback_url%5D=https%3A%2F%2Foauth-openshift.apps.rh-moa-test-cluster.z7v0.s1.devshift.org%2Foauth2callback%2Fgithub-1&oauth_application%5Bname%5D=rh-moa-test-cluster-stage&oauth_application%5Burl%5D=https%3A%2F%2Fconsole-openshift-console.apps.rh-moa-test-cluster.z7v0.s1.devshift.org
  - Click on 'Register application'
? Client ID: &lt;my-github-client-id&
? Client Secret: [? for help] &lt;my-github-client-secret&
? Hostname:
? Mapping method: claim
I: Configuring IDP for cluster 'rh-moa-test-cluster'
I: Identity Provider 'github-1' has been created. You need to ensure that there is a list of cluster administrators defined. See 'rosa create user --help' for more information. To login into the console, open https://console-openshift-console.apps.rh-test-org.z7v0.s1.devshift.org and click on github-1
```

The IDP can take 1-2 minutes to be configured within your cluster.

Run the following command to verify that your IDP has been configured correctly:

```
$ rosa list idps --cluster rh-moa-test-cluster
NAME        TYPE      AUTH URL
github-1    GitHub    https://oauth-openshift.apps.rh-moa-test-cluster.j9n4.s1.devshift.org/oauth2callback/github-1
```

### Log in to your cluster

At this point you should be able to log in to your cluster. The follow examples continue to use GitHub as an example IDP.

First, run the following command to get the `Console URL` of your cluster:

```
$ rosa describe cluster rh-moa-test-cluster
Name:        rh-moa-test-cluster
ID:          1de87g7c30g75qechgh7l5b2bha6r04e
External ID: 34322be7-b2a7-45c2-af39-2c684ce624e1
API URL:     https://api.rh-moa-test-cluster.j9n4.s1.devshift.org:6443
Console URL: https://console-openshift-console.apps.rh-moa-test-cluster.j9n4.s1.devshift.org
Nodes:       Master: 3, Infra: 2, Compute: 2
Region:      us-east-2
State:       ready
Created:     May 27, 2020
```

Navigate to the `Console URL` and log in using your GitHub credentials (or the credentials for the IDP you added to your cluster).

Once you are logged into your cluster, follow these steps to get your `oc` login command. In the top right of the OpenShift console, click your name and click **Copy Login Command**.  Click **github-1** and finally click **Display Token**. Copy and paste the `oc` login command into your terminal.

```
$ oc login --token=z3sgOGVDk0k4vbqo_wFqBQQTnT-nA-nQLb8XEmWnw4X --server=https://api.rh-moa-test-cluster.j9n4.s1.devshift.org:6443
Logged into "https://api.rh-moa-test-cluster.j9n4.s1.devshift.org:6443" as "rh-moa-test-user" using the token provided.

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

## (Optional) Create dedicated and cluster admins

### Create a dedicated-admin user

Run the following command to promote your Github user to dedicated-admin:

```
$ rosa create user --cluster rh-moa-test-cluster --dedicated-admins=rh-moa-test-user
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
$ rosa edit cluster rh-moa-test-cluster --enable-cluster-admins
```

Next give your user cluster-admin privileges:

```
$ rosa create user --cluster rh-moa-test-cluster --cluster-admins rh-moa-test-user
$ rosa list users --cluster rh-moa-test-cluster
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

Run the following command to delete your cluster, replacing `<my-cluster>` with the name of your cluster:

```
rosa delete cluster -c <my-cluster>
```

Once the cluster is uninstalled, you can clean up your CloudFormation stack (this was created when you ran `rosa init`) by running the following command:

```
rosa init --delete-stack
```

## Build from source

If you'd like to build this project from source use the following steps:

1. Checkout the repostiory into your `$GOPATH`

```
go get -u github.com/openshift/rosa
```

2. `cd` to the checkout out source directory

```
cd $GOPATH/src/github.com/openshift/rosa
```

3. Install the binary (This will install to `$GOPATH/bin`)

```
make install
```

NOTE: If you don't have `$GOPATH/bin` in your `$PATH` you need to add it or move `rosa` to a standard system directory eg. for Linux/OSX:

```
sudo mv $GOPATH/bin/rosa /usr/local/bin
```

## Have you got feedback?

We want to hear it. [Open and issue](https://github.com/openshift/rosa/issues/new) against the repo and someone from the team will be in touch.
