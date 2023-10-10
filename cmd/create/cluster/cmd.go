/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/aws/aws-sdk-go/service/ec2"
	clustervalidations "github.com/openshift-online/ocm-common/pkg/cluster/validations"
	passwordValidator "github.com/openshift-online/ocm-common/pkg/idp/validations"
	diskValidator "github.com/openshift-online/ocm-common/pkg/machinepool/validations"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/cmd/create/operatorroles"
	clusterdescribe "github.com/openshift/rosa/cmd/describe/cluster"
	installLogs "github.com/openshift/rosa/cmd/logs/install"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/helper"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/interactive/consts"
	interactiveOidc "github.com/openshift/rosa/pkg/interactive/oidc"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

// nolint
var kmsArnRE = regexp.MustCompile(
	`^arn:aws[\w-]*:kms:[\w-]+:\d{12}:key\/mrk-[0-9a-f]{32}$|[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

const (
	OidcConfigIdFlag      = "oidc-config-id"
	ClassicOidcConfigFlag = "classic-oidc-config"

	defaultIngressRouteSelectorFlag            = "default-ingress-route-selector"
	defaultIngressExcludedNamespacesFlag       = "default-ingress-excluded-namespaces"
	defaultIngressWildcardPolicyFlag           = "default-ingress-wildcard-policy"
	defaultIngressNamespaceOwnershipPolicyFlag = "default-ingress-namespace-ownership-policy"
	additionalComputeSecurityGroupIdsFlag      = "additional-compute-security-group-ids"

	clusterAutoscalerFlagsPrefix = "autoscaler-"

	MinReplicasSingleAZ = 2
	MinReplicaMultiAZ   = 3
)

var args struct {
	// Watch logs during cluster installation
	watch bool

	// Simulate creating a cluster
	dryRun bool
	// Create a fake cluster with no AWS resources
	fakeCluster bool
	// Set custom properties in cluster spec
	properties []string
	// Use local AWS credentials instead of the 'osdCcsAdmin' user
	useLocalCredentials bool

	// Disable SCP checks in the installer
	disableSCPChecks bool

	// Basic options
	private                   bool
	privateLink               bool
	multiAZ                   bool
	expirationDuration        time.Duration
	expirationTime            string
	clusterName               string
	region                    string
	version                   string
	channelGroup              string
	flavour                   string
	disableWorkloadMonitoring bool
	ec2MetadataHttpTokens     string

	//Encryption
	etcdEncryption           bool
	fips                     bool
	enableCustomerManagedKey bool
	kmsKeyARN                string
	etcdEncryptionKmsARN     string
	// Scaling options
	computeMachineType       string
	computeNodes             int
	autoscalingEnabled       bool
	minReplicas              int
	maxReplicas              int
	defaultMachinePoolLabels string

	// Networking options
	networkType string
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet
	hostPrefix  int

	// The Subnet IDs to use when installing the cluster.
	// SubnetIDs should come in pairs; two per availability zone, one private and one public,
	// unless using PrivateLink, in which case it should only be one private per availability zone
	subnetIDs []string

	// Selecting availability zones for a non-BYOVPC cluster
	availabilityZones []string

	// Force STS mode for interactive and validation
	sts bool

	// Force IAM mode (mint mode) for interactive
	nonSts bool

	// Account IAM Roles
	roleARN             string
	externalID          string
	supportRoleARN      string
	controlPlaneRoleARN string
	workerRoleARN       string

	// Operator IAM Roles
	operatorIAMRoles                 []string
	operatorRolesPrefix              string
	operatorRolesPermissionsBoundary string

	// Oidc Config
	oidcConfigId      string
	classicOidcConfig bool

	// Proxy
	enableProxy               bool
	httpProxy                 string
	httpsProxy                string
	noProxySlice              []string
	additionalTrustBundleFile string

	tags []string

	// Hypershift options:
	hostedClusterEnabled bool
	billingAccount       string

	// Cluster Admin
	createAdminUser      bool
	clusterAdminPassword string
	// Deprecated Cluster Admin
	clusterAdminUser string

	// Audit Log Forwarding
	AuditLogRoleARN string

	// Default Ingress Attributes
	defaultIngressRouteSelectors            string
	defaultIngressExcludedNamespaces        string
	defaultIngressWildcardPolicy            string
	defaultIngressNamespaceOwnershipPolicy  string
	defaultIngressClusterRoutesHostname     string
	defaultIngressClusterRoutesTlsSecretRef string

	// Storage
	machinePoolRootDiskSize string

	// Shared VPC
	privateHostedZoneID string
	sharedVPCRoleARN    string
	baseDomain          string

	// Worker machine pool attributes
	additionalComputeSecurityGroupIds []string
}

var autoscalerArgs *clusterautoscaler.AutoscalerArgs

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Example: `  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	// Basic options
	flags.StringVarP(
		&args.clusterName,
		"name",
		"n",
		"",
		"Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.",
	)
	flags.MarkDeprecated("name", "use --cluster-name instead")

	flags.StringVarP(
		&args.clusterName,
		"cluster-name",
		"c",
		"",
		"Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.",
	)

	flags.BoolVar(
		&args.sts,
		"sts",
		false,
		"Use AWS Security Token Service (STS) instead of IAM credentials to deploy your cluster.",
	)
	flags.BoolVar(
		&args.nonSts,
		"non-sts",
		false,
		"Use legacy method of creating clusters (IAM mode).",
	)
	flags.BoolVar(
		&args.nonSts,
		"mint-mode",
		false,
		"Use legacy method of creating clusters (IAM mode). This is an alias for --non-sts.",
	)
	flags.StringVar(
		&args.roleARN,
		"role-arn",
		"",
		"The Amazon Resource Name of the role that OpenShift Cluster Manager will assume to create the cluster.",
	)
	flags.StringVar(
		&args.externalID,
		"external-id",
		"",
		"An optional unique identifier that might be required when you assume a role in another account.",
	)
	flags.StringVar(
		&args.supportRoleARN,
		"support-role-arn",
		"",
		"The Amazon Resource Name of the role used by Red Hat SREs to enable "+
			"access to the cluster account in order to provide support.",
	)

	flags.StringVar(
		&args.controlPlaneRoleARN,
		"controlplane-iam-role",
		"",
		"The IAM role ARN that will be attached to control plane instances.",
	)

	flags.StringVar(
		&args.controlPlaneRoleARN,
		"master-iam-role",
		"",
		"The IAM role ARN that will be attached to master instances.",
	)
	flags.MarkDeprecated("master-iam-role", "use --controlplane-iam-role instead")

	flags.StringVar(
		&args.workerRoleARN,
		"worker-iam-role",
		"",
		"The IAM role ARN that will be attached to worker instances.",
	)

	flags.StringArrayVar(
		&args.operatorIAMRoles,
		"operator-iam-roles",
		nil,
		"List of OpenShift name and namespace, and role ARNs used to perform credential "+
			"requests by operators needed in the OpenShift installer.",
	)
	flags.MarkDeprecated("operator-iam-roles", "use --operator-roles-prefix instead")
	flags.StringVar(
		&args.operatorRolesPrefix,
		"operator-roles-prefix",
		"",
		"Prefix to use for all IAM roles used by the operators needed in the OpenShift installer. "+
			"Leave empty to use an auto-generated one.",
	)

	flags.StringVar(
		&args.oidcConfigId,
		OidcConfigIdFlag,
		"",
		"Registered OIDC Configuration ID to use for cluster creation",
	)

	flags.BoolVar(
		&args.classicOidcConfig,
		ClassicOidcConfigFlag,
		false,
		"Use classic OIDC configuration without registering an ID.",
	)
	flags.MarkHidden(ClassicOidcConfigFlag)

	flags.StringSliceVar(
		&args.tags,
		"tags",
		nil,
		"Apply user defined tags to all resources created by ROSA in AWS. "+
			"Tags are comma separated, for example: 'key value, foo bar'",
	)

	flags.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	arguments.AddRegionFlag(flags)
	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to install the cluster, for example \"4.3.10\"",
	)
	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the group where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	flags.StringVar(
		&args.flavour,
		"flavour",
		"osd-4",
		"Set of predefined properties of a cluster",
	)
	flags.MarkHidden("flavour")

	flags.BoolVar(
		&args.etcdEncryption,
		"etcd-encryption",
		false,
		"Add etcd encryption. By default etcd data is encrypted at rest. "+
			"This option configures etcd encryption on top of existing storage encryption.",
	)
	flags.BoolVar(
		&args.fips,
		"fips",
		false,
		"Create cluster that uses FIPS Validated / Modules in Process cryptographic libraries.",
	)

	flags.StringVar(
		&args.httpProxy,
		"http-proxy",
		"",
		"A proxy URL to use for creating HTTP connections outside the cluster. The URL scheme must be http.",
	)

	flags.StringVar(
		&args.httpsProxy,
		"https-proxy",
		"",
		"A proxy URL to use for creating HTTPS connections outside the cluster.",
	)

	flags.StringSliceVar(
		&args.noProxySlice,
		"no-proxy",
		nil,
		"A comma-separated list of destination domain names, domains, IP addresses or "+
			"other network CIDRs to exclude proxying.",
	)

	flags.StringVar(
		&args.additionalTrustBundleFile,
		"additional-trust-bundle-file",
		"",
		"A file contains a PEM-encoded X.509 certificate bundle that will be "+
			"added to the nodes' trusted certificate store.")

	flags.BoolVar(&args.enableCustomerManagedKey,
		"enable-customer-managed-key",
		false,
		"Enable to specify your KMS Key to encrypt EBS instance volumes. By default account’s default "+
			"KMS key for that particular region is used.")

	flags.StringVar(&args.kmsKeyARN,
		"kms-key-arn",
		"",
		"The key ARN is the Amazon Resource Name (ARN) of a CMK. It is a unique, "+
			"fully qualified identifier for the CMK. A key ARN includes the AWS account, Region, and the key ID.")

	flags.StringVar(&args.etcdEncryptionKmsARN,
		"etcd-encryption-kms-arn",
		"",
		"The etcd encryption kms key ARN is the key used to encrypt etcd. "+
			"If set it will override etcd-encryption flag to true. It is a unique, "+
			"fully qualified identifier for the CMK. A key ARN includes the AWS account, Region, and the key ID.")

	flags.StringVar(
		&args.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&args.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	flags.MarkHidden("expiration-time")
	flags.MarkHidden("expiration")

	flags.BoolVar(
		&args.privateLink,
		"private-link",
		false,
		"Provides private connectivity between VPCs, AWS services, and your on-premises networks, "+
			"without exposing your traffic to the public internet.",
	)

	flags.StringVar(
		&args.ec2MetadataHttpTokens,
		"ec2-metadata-http-tokens",
		"",
		"Should cluster nodes use both v1 and v2 endpoints or just v2 endpoint "+
			"of EC2 Instance Metadata Service (IMDS)",
	)

	flags.StringSliceVar(
		&args.subnetIDs,
		"subnet-ids",
		nil,
		"The Subnet IDs to use when installing the cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for installer provisioned subnet IDs.",
	)

	flags.StringSliceVar(
		&args.availabilityZones,
		"availability-zones",
		nil,
		"The availability zones to use when installing a non-BYOVPC cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for the installer to pick availability zones")

	// Scaling options
	flags.StringVar(
		&args.computeMachineType,
		"compute-machine-type",
		"",
		"Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.",
	)

	flags.IntVar(
		&args.computeNodes,
		"compute-nodes",
		2,
		"Number of worker nodes to provision. Single zone clusters need at least 2 nodes, "+
			"multizone clusters need at least 3 nodes.",
	)
	flags.MarkDeprecated("compute-nodes", "use --replicas instead")
	flags.IntVar(
		&args.computeNodes,
		"replicas",
		2,
		"Number of worker nodes to provision. Single zone clusters need at least 2 nodes, "+
			"multizone clusters need at least 3 nodes. Hosted clusters require that the number of worker nodes be a "+
			"multiple of the number of private subnets.",
	)

	flags.BoolVar(
		&args.autoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling of compute nodes.",
	)

	autoscalerArgs = clusterautoscaler.AddClusterAutoscalerFlags(Cmd, clusterAutoscalerFlagsPrefix)

	flags.VisitAll(func(f *pflag.Flag) {
		if strings.HasPrefix(f.Name, clusterAutoscalerFlagsPrefix) {
			Cmd.MarkFlagsRequiredTogether("enable-autoscaling", f.Name)
		}
	})

	flags.IntVar(
		&args.minReplicas,
		"min-replicas",
		2,
		"Minimum number of compute nodes.",
	)

	flags.IntVar(
		&args.maxReplicas,
		"max-replicas",
		2,
		"Maximum number of compute nodes.",
	)

	flags.StringVar(
		&args.defaultMachinePoolLabels,
		"default-mp-labels",
		"",
		"Labels for the worker machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.networkType,
		"network-type",
		"",
		"The main controller responsible for rendering the core networking components.",
	)
	flags.MarkHidden("network-type")
	Cmd.RegisterFlagCompletionFunc("network-type", networkTypeCompletion)

	flags.IPNetVar(
		&args.machineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	flags.IPNetVar(
		&args.serviceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	flags.IPNetVar(
		&args.podCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	flags.IntVar(
		&args.hostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set "+
			"to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
	)
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint and application routes to direct, private connectivity.",
	)

	flags.BoolVar(
		&args.disableSCPChecks,
		"disable-scp-checks",
		false,
		"Indicates if cloud permission checks are disabled when attempting installation of the cluster.",
	)
	flags.BoolVar(
		&args.disableWorkloadMonitoring,
		"disable-workload-monitoring",
		false,
		"Enables you to monitor your own projects in isolation from Red Hat Site Reliability Engineer (SRE) "+
			"platform metrics.",
	)

	flags.BoolVarP(
		&args.watch,
		"watch",
		"w",
		false,
		"Watch cluster installation logs.",
	)

	flags.BoolVar(
		&args.dryRun,
		"dry-run",
		false,
		"Simulate creating the cluster.",
	)

	flags.BoolVar(
		&args.fakeCluster,
		"fake-cluster",
		false,
		"Create a fake cluster that uses no AWS resources.",
	)
	flags.MarkHidden("fake-cluster")

	flags.StringArrayVar(
		&args.properties,
		"properties",
		nil,
		"User defined properties for tagging and querying.",
	)
	flags.MarkHidden("properties")

	flags.BoolVar(
		&args.useLocalCredentials,
		"use-local-credentials",
		false,
		"Use local AWS credentials instead of the 'osdCcsAdmin' user. This is not supported.",
	)
	flags.MarkHidden("use-local-credentials")

	flags.StringVar(
		&args.operatorRolesPermissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the operator "+
			"roles in STS clusters.",
	)

	// Options related to HyperShift:
	flags.BoolVar(
		&args.hostedClusterEnabled,
		"hosted-cp",
		false,
		"Technology Preview: Enable the use of Hosted Control Planes",
	)

	flags.StringVar(&args.machinePoolRootDiskSize,
		"worker-disk-size",
		"",
		"Default worker machine pool root disk size with a **unit suffix** like GiB or TiB, "+
			"e.g. 200GiB.")

	flags.StringVar(
		&args.billingAccount,
		"billing-account",
		"",
		"Account used for billing subscriptions purchased via the AWS marketplace",
	)
	flags.MarkHidden("billing-account")

	flags.BoolVar(
		&args.createAdminUser,
		"create-admin-user",
		false,
		`Create cluster admin`,
	)

	flags.StringVar(
		&args.clusterAdminUser,
		"cluster-admin-user",
		"",
		`Deprecated cluster admin flag. Please use --create-admin-user.`,
	)
	flags.StringVar(
		&args.clusterAdminPassword,
		"cluster-admin-password",
		"",
		`The password must
		- Be at least 14 characters (ASCII-standard) without whitespaces
		- Include uppercase letters, lowercase letters, and numbers or symbols (ASCII-standard characters only)`,
	)
	// cluster admin flags deprecated to be removed
	flags.MarkHidden("cluster-admin-user")

	flags.StringVar(
		&args.AuditLogRoleARN,
		"audit-log-arn",
		"",
		"The ARN of the role that is used to forward audit logs to AWS CloudWatch.",
	)

	flags.StringVar(
		&args.defaultIngressRouteSelectors,
		defaultIngressRouteSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	flags.StringVar(
		&args.defaultIngressExcludedNamespaces,
		defaultIngressExcludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&args.defaultIngressWildcardPolicy,
		defaultIngressWildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ingress.ValidWildcardPolicies, ","), ingress.DefaultWildcardPolicy),
	)

	flags.StringVar(
		&args.defaultIngressNamespaceOwnershipPolicy,
		defaultIngressNamespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ingress.ValidNamespaceOwnershipPolicies, ","), ingress.DefaultNamespaceOwnershipPolicy),
	)

	flags.StringVar(
		&args.privateHostedZoneID,
		"private-hosted-zone-id",
		"",
		"ID assigned by AWS to private Route 53 hosted zone associated with intended shared VPC, "+
			"e.g., 'Z05646003S02O1ENCDCSN'.",
	)

	flags.StringVar(
		&args.sharedVPCRoleARN,
		"shared-vpc-role-arn",
		"",
		"AWS IAM role ARN with a policy attached, granting permissions necessary to create and manage Route 53 DNS records "+
			"in private Route 53 hosted zone associated with intended shared VPC.",
	)

	flags.StringVar(
		&args.baseDomain,
		"base-domain",
		"",
		"Base DNS domain name previously reserved and matching the hosted zone name of the private Route 53 hosted zone "+
			"associated with intended shared VPC, e.g., '1vo8.p1.openshiftapps.com'.",
	)

	flags.StringSliceVar(
		&args.additionalComputeSecurityGroupIds,
		additionalComputeSecurityGroupIdsFlag,
		nil,
		"The additional security groups for default worker machine pool.",
	)

	aws.AddModeFlag(Cmd)
	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
	confirm.AddFlag(flags)
}

func networkTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return ocm.NetworkTypes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Validate mode
	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// validate flags for cluster admin
	isHostedCP := args.hostedClusterEnabled
	createAdminUser := args.createAdminUser
	clusterAdminPassword := strings.Trim(args.clusterAdminPassword, " \t")
	if (createAdminUser || clusterAdminPassword != "") && isHostedCP {
		r.Reporter.Errorf("Setting Cluster Admin is only supported in classic ROSA clusters")
		os.Exit(1)
	}
	// error when using deprecated admin flags
	clusterAdminUser := strings.Trim(args.clusterAdminUser, " \t")
	if clusterAdminUser != "" {
		r.Reporter.Errorf("'--cluster-admin-user' flag has been deprecated " +
			"and replaced with '--create-admin-user'")
		os.Exit(1)
	}

	supportedRegions, err := r.OCMClient.GetDatabaseRegionList()
	if err != nil {
		r.Reporter.Errorf("Unable to retrieve supported regions: %v", err)
	}
	awsClient := aws.GetAWSClientForUserRegion(r.Reporter, r.Logger, supportedRegions, args.useLocalCredentials)
	r.AWSClient = awsClient

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		r.Reporter.Errorf("Unable to get IAM credentials: %v", err)
		os.Exit(1)
	}

	shardPinningEnabled := false
	for _, value := range args.properties {
		if strings.HasPrefix(value, properties.ProvisionShardId) {
			shardPinningEnabled = true
			break
		}
	}

	isBYOVPC := cmd.Flags().Changed("subnet-ids")
	isAvailabilityZonesSet := cmd.Flags().Changed("availability-zones")
	// Setting subnet IDs is choosing BYOVPC implicitly,
	// and selecting availability zones is only allowed for non-BYOVPC clusters
	if isBYOVPC && isAvailabilityZonesSet {
		r.Reporter.Errorf("Setting availability zones is not supported for BYO VPC. " +
			"ROSA autodetects availability zones from subnet IDs provided")
	}

	// Select a multi-AZ cluster implicitly by providing three availability zones
	if len(args.availabilityZones) == clustervalidations.MultiAZCount {
		args.multiAZ = true
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Get cluster name
	clusterName := strings.Trim(args.clusterName, " \t")

	if clusterName == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	if interactive.Enabled() {
		clusterName, err = interactive.GetString(interactive.Input{
			Question: "Cluster name",
			Help:     cmd.Flags().Lookup("cluster-name").Usage,
			Default:  clusterName,
			Required: true,
			Validators: []interactive.Validator{
				ocm.ClusterNameValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid cluster name: %s", err)
			os.Exit(1)
		}
	}

	// Trim names to remove any leading/trailing invisible characters
	clusterName = strings.Trim(clusterName, " \t")

	if !ocm.IsValidClusterName(clusterName) {
		r.Reporter.Errorf("Cluster name must consist" +
			" of no more than 15 lowercase alphanumeric characters or '-', " +
			"start with a letter, and end with an alphanumeric character.")
		os.Exit(1)
	}

	if interactive.Enabled() {
		isHostedCP, err = interactive.GetBool(interactive.Input{
			Question: "Deploy cluster with Hosted Control Plane",
			Help:     cmd.Flags().Lookup("hosted-cp").Usage,
			Default:  isHostedCP,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid --hosted-cp value: %s", err)
			os.Exit(1)
		}
	}

	// FIXME: Remove before GA
	if isHostedCP && r.Reporter.IsTerminal() {
		//nolint
		r.Reporter.Infof("NOTE: Hosted control planes are currently in Technology Preview (https://access.redhat.com/support/offerings/techpreview)." +
			" Any Technology Preview clusters will need to be destroyed and recreated prior to general availability.")
	}

	if isHostedCP && args.ec2MetadataHttpTokens != "" {
		r.Reporter.Errorf("ec2-metadata-http-tokens can't be set with hosted-cp")
		os.Exit(1)
	}

	// setting default for ec2MetadataHttpTokens in case cluster is not hosted
	// or ec2MetadataHttpTokens was not set
	if !isHostedCP && args.ec2MetadataHttpTokens == "" {
		args.ec2MetadataHttpTokens = string(v1.Ec2MetadataHttpTokensOptional)
	}

	// Errors when users elects for cluster admin via flags and elects for hosted control plane via interactive prompt"
	if isHostedCP && (createAdminUser || clusterAdminPassword != "") {
		r.Reporter.Errorf("Setting Cluster Admin is only supported in classic ROSA clusters")
		os.Exit(1)
	}

	// isClusterAdmin is a flag indicating if user wishes to create cluster admin
	isClusterAdmin := false
	if !isHostedCP {
		if createAdminUser {
			isClusterAdmin = true
			// user supplies create-admin-user flag without cluster-admin-password will generate random password
			if clusterAdminPassword == "" {
				r.Reporter.Debugf(admin.GeneratingRandomPasswordString)
				clusterAdminPassword, err = admin.GenerateRandomPassword()
				if err != nil {
					r.Reporter.Errorf("Failed to generate a random password")
					os.Exit(1)
				}
			}
		}
		// validates both user inputted custom password and randomly generated password
		if clusterAdminPassword != "" {
			err = passwordValidator.PasswordValidator(clusterAdminPassword)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			isClusterAdmin = true
		}

		//check to remove first condition (interactive mode)
		if interactive.Enabled() && !isClusterAdmin {
			isClusterAdmin, err = interactive.GetBool(interactive.Input{
				Question: "Create cluster admin user",
				Default:  false,
				Required: true,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value: %s", err)
				os.Exit(1)
			}
			if isClusterAdmin {
				isCustomAdminPassword, err := interactive.GetBool(interactive.Input{
					Question: "Create custom password for cluster admin",
					Default:  false,
					Required: true,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid value: %s", err)
					os.Exit(1)
				}
				if !isCustomAdminPassword {
					clusterAdminPassword, err = admin.GenerateRandomPassword()
					if err != nil {
						r.Reporter.Errorf("Failed to generate a random password")
						os.Exit(1)
					}
				} else {
					clusterAdminPassword = idp.GetIdpPasswordFromPrompt(cmd, r,
						"cluster-admin-password", clusterAdminPassword)
					args.clusterAdminPassword = clusterAdminPassword
				}
			}
		}
		outputClusterAdminDetails(r, isClusterAdmin, clusterAdminPassword)
	}

	// Billing Account
	billingAccount := args.billingAccount
	if billingAccount != "" && !isHostedCP {
		r.Reporter.Errorf("Billing accounts are only supported for Hosted Control Plane clusters")
		os.Exit(1)
	}

	if billingAccount != "" && !ocm.IsValidAWSAccount(billingAccount) {
		r.Reporter.Errorf("Expected a valid billing account")
		os.Exit(1)
	}

	if isHostedCP && cmd.Flags().Changed("default-mp-labels") {
		r.Reporter.Errorf("Setting the worker machine pool labels is not supported for hosted clusters")
		os.Exit(1)
	}

	etcdEncryptionKmsARN := args.etcdEncryptionKmsARN

	if etcdEncryptionKmsARN != "" && !isHostedCP {
		r.Reporter.Errorf("etcd encryption kms arn is only allowed for hosted cp")
		os.Exit(1)
	}

	// all hosted clusters are sts
	isSTS := args.sts || args.roleARN != "" || fedramp.Enabled() || isHostedCP
	isIAM := (cmd.Flags().Changed("sts") && !isSTS) || args.nonSts

	if isSTS && isIAM {
		r.Reporter.Errorf("Can't use both STS and mint mode at the same time.")
		os.Exit(1)
	}

	if interactive.Enabled() && (!isSTS && !isIAM) {
		isSTS, err = interactive.GetBool(interactive.Input{
			Question: "Deploy cluster using AWS STS",
			Help:     cmd.Flags().Lookup("sts").Usage,
			Default:  true,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid --sts value: %s", err)
			os.Exit(1)
		}
		isIAM = !isSTS
	}

	isSTS = isSTS || awsCreator.IsSTS

	if r.Reporter.IsTerminal() && !isHostedCP {
		r.Reporter.Warnf("In a future release STS will be the default mode.")
		r.Reporter.Warnf("--sts flag won't be necessary if you wish to use STS.")
		r.Reporter.Warnf("--non-sts/--mint-mode flag will be necessary if you do not wish to use STS.")
	}

	permissionsBoundary := args.operatorRolesPermissionsBoundary
	if permissionsBoundary != "" {
		err = aws.ARNValidator(permissionsBoundary)
		if err != nil {
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	if isIAM {
		if awsCreator.IsSTS {
			r.Reporter.Errorf("Since your AWS credentials are returning an STS ARN you can only " +
				"create STS clusters. Otherwise, switch to IAM credentials.")
			os.Exit(1)
		}
		err := awsClient.CheckAdminUserExists(aws.AdminUserName)
		if err != nil {
			r.Reporter.Errorf("IAM user '%s' does not exist. Run `rosa init` first", aws.AdminUserName)
			os.Exit(1)
		}
		r.Reporter.Debugf("IAM user is valid!")
	}

	// AWS ARN Role
	roleARN := args.roleARN
	supportRoleARN := args.supportRoleARN
	controlPlaneRoleARN := args.controlPlaneRoleARN
	workerRoleARN := args.workerRoleARN

	// OpenShift version:
	version := args.version
	channelGroup := args.channelGroup
	versionList, err := versions.GetVersionList(r, channelGroup, isSTS, isHostedCP, true)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if version == "" {
		version = versionList[0]
	}
	if interactive.Enabled() {
		version, err = interactive.GetOption(interactive.Input{
			Question: "OpenShift version",
			Help:     cmd.Flags().Lookup("version").Usage,
			Options:  versionList,
			Default:  version,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
			os.Exit(1)
		}
	}
	version, err = r.OCMClient.ValidateVersion(version, versionList, channelGroup, isSTS, isHostedCP)
	if err != nil {
		r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
		os.Exit(1)
	}
	if err := r.OCMClient.IsVersionCloseToEol(ocm.CloseToEolDays, version, channelGroup); err != nil {
		r.Reporter.Warnf("%v", err)
		if !confirm.Confirm("continue with version '%s'", ocm.GetRawVersionId(version)) {
			os.Exit(0)
		}
	}

	httpTokens := args.ec2MetadataHttpTokens
	if interactive.Enabled() && !isHostedCP {
		httpTokens, err = interactive.GetString(interactive.Input{
			Question: fmt.Sprintf("Configure the use of IMDSv2 for ec2 instances %s/%s",
				v1.Ec2MetadataHttpTokensOptional, v1.Ec2MetadataHttpTokensRequired),
			Help:    cmd.Flags().Lookup("ec2-metadata-http-tokens").Usage,
			Default: httpTokens,
			Validators: []interactive.Validator{
				ocm.ValidateHttpTokensValue,
			},
		})

	} else {
		err = ocm.ValidateHttpTokensValue(httpTokens)
	}
	if err != nil {
		r.Reporter.Errorf("Expected a valid http tokens value : %v", err)
		os.Exit(1)
	}
	if err := ocm.ValidateHttpTokensVersion(ocm.GetVersionMinor(version), httpTokens); err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}

	// warn if mode is used for non sts cluster
	if !isSTS && mode != "" {
		r.Reporter.Warnf("--mode is only valid for STS clusters")
	}

	// validate mode passed is allowed value

	if isSTS && mode != "" {
		isValidMode := arguments.IsValidMode(aws.Modes, mode)
		if !isValidMode {
			r.Reporter.Errorf("Invalid --mode '%s'. Allowed values are %s", mode, aws.Modes)
			os.Exit(1)
		}
	}

	if args.watch && isSTS && mode == aws.ModeAuto && !confirm.Yes() {
		r.Reporter.Errorf("Cannot watch for STS cluster installation logs in mode 'auto' " +
			"without also supplying '--yes' option." +
			"To watch your cluster installation logs, run 'rosa logs install' instead after the cluster has began creating.")
		os.Exit(1)
	}

	if args.watch && isSTS && mode == aws.ModeManual {
		r.Reporter.Errorf("Cannot watch for STS cluster installation logs in mode 'manual'." +
			"It requires manual commands to be performed as part of the process." +
			"To watch your cluster installation logs, run 'rosa logs install' after the cluster has began creating.")
		os.Exit(1)
	}

	hasRoles := false
	if isSTS && roleARN == "" {
		minor := ocm.GetVersionMinor(version)
		role := aws.AccountRoles[aws.InstallerAccountRole]

		// Find all installer roles in the current account using AWS resource tags
		roleARNs, err := awsClient.FindRoleARNs(aws.InstallerAccountRole, minor)
		if err != nil {
			r.Reporter.Errorf("Failed to find %s role: %s", role.Name, err)
			os.Exit(1)
		}

		if len(roleARNs) > 1 {
			defaultRoleARN := roleARNs[0]
			// Prioritize roles with the default prefix
			for _, rARN := range roleARNs {
				roleName, err := aws.GetResourceIdFromARN(rARN)
				if err != nil {
					continue
				}
				if roleName == fmt.Sprintf("%s-%s-Role", aws.DefaultPrefix, role.Name) {
					defaultRoleARN = rARN
				}
			}
			r.Reporter.Warnf("More than one %s role found", role.Name)
			if !interactive.Enabled() && confirm.Yes() {
				r.Reporter.Infof("Using %s for the %s role", defaultRoleARN, role.Name)
				roleARN = defaultRoleARN
			} else {
				roleARN, err = interactive.GetOption(interactive.Input{
					Question: fmt.Sprintf("%s role ARN", role.Name),
					Help:     cmd.Flags().Lookup(role.Flag).Usage,
					Options:  roleARNs,
					Default:  defaultRoleARN,
					Required: true,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid role ARN: %s", err)
					os.Exit(1)
				}
			}
		} else if len(roleARNs) == 1 {
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Using %s for the %s role", roleARNs[0], role.Name)
			}
			roleARN = roleARNs[0]
		} else {
			createAccountRolesCommand := "rosa create account-roles"
			if isHostedCP {
				createAccountRolesCommand = createAccountRolesCommand + " --hosted-cp"
			}
			r.Reporter.Warnf(fmt.Sprintf("No account roles found. You will need to manually set them in the "+
				"next steps or run '%s' to create them first.", createAccountRolesCommand))
			interactive.Enable()
		}

		if roleARN != "" {
			// check if role has hosted cp policy via AWS tag value
			hostedCPPolicies, err := awsClient.HasHostedCPPolicies(roleARN)
			if err != nil {
				r.Reporter.Errorf("Failed to determine if cluster has hosted CP policies: %v", err)
				os.Exit(1)
			}
			hasRoles = true
			for roleType, role := range aws.AccountRoles {
				if roleType == aws.InstallerAccountRole {
					// Already dealt with
					continue
				}
				if isHostedCP && roleType == aws.ControlPlaneAccountRole {
					// Not needed for Hypershift clusters
					continue
				}
				roleARNs, err := awsClient.FindRoleARNs(roleType, minor)
				if err != nil {
					r.Reporter.Errorf("Failed to find %s role: %s", role.Name, err)
					os.Exit(1)
				}
				selectedARN := ""
				expectedResourceIDForAccRole, rolePrefix, err := getExpectedResourceIDForAccRole(
					hostedCPPolicies, roleARN, roleType)
				if err != nil {
					r.Reporter.Errorf("Failed to get the expected resource ID for role type: %s", roleType)
					os.Exit(1)
				}
				r.Reporter.Debugf("Using '%s' as the role prefix to retrieve the expected resource ID for role type '%s'",
					rolePrefix, roleType)

				for _, rARN := range roleARNs {
					resourceId, err := aws.GetResourceIdFromARN(rARN)
					if err != nil {
						r.Reporter.Errorf("Failed to get resource ID from arn. %s", err)
						os.Exit(1)
					}
					lowerCaseResourceIdToCheck := strings.ToLower(resourceId)
					if lowerCaseResourceIdToCheck == expectedResourceIDForAccRole {
						selectedARN = rARN
						break
					}
				}
				if selectedARN == "" {
					createAccountRolesCommand := "rosa create account-roles"
					if isHostedCP {
						createAccountRolesCommand = createAccountRolesCommand + " --hosted-cp"
					}
					r.Reporter.Warnf(fmt.Sprintf("No %s account roles found. You will need to manually set "+
						"them in the next steps or run '%s' to create "+
						"them first.", role.Name, createAccountRolesCommand))
					interactive.Enable()
					hasRoles = false
					break
				}
				if !output.HasFlag() || r.Reporter.IsTerminal() {
					r.Reporter.Infof("Using %s for the %s role", selectedARN, role.Name)
				}
				switch roleType {
				case aws.InstallerAccountRole:
					roleARN = selectedARN
				case aws.SupportAccountRole:
					supportRoleARN = selectedARN
				case aws.ControlPlaneAccountRole:
					controlPlaneRoleARN = selectedARN
				case aws.WorkerAccountRole:
					workerRoleARN = selectedARN
				}
			}
		}
	}

	if isSTS && !hasRoles && interactive.Enabled() {
		roleARN, err = interactive.GetString(interactive.Input{
			Question: "Role ARN",
			Help:     cmd.Flags().Lookup("role-arn").Usage,
			Default:  roleARN,
			Required: isSTS,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid ARN: %s", err)
			os.Exit(1)
		}
	}

	if roleARN != "" {
		err = aws.ARNValidator(roleARN)
		if err != nil {
			r.Reporter.Errorf("Expected a valid Role ARN: %s", err)
			os.Exit(1)
		}
		isSTS = true
	}

	if !isSTS && mode != "" {
		r.Reporter.Warnf("--mode is only valid for STS clusters")
	}

	externalID := args.externalID
	if isSTS && interactive.Enabled() {
		externalID, err = interactive.GetString(interactive.Input{
			Question: "External ID",
			Help:     cmd.Flags().Lookup("external-id").Usage,
			Validators: []interactive.Validator{
				interactive.RegExp(`^[\w+=,.@:\/-]*$`),
				interactive.MaxLength(1224),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid External ID: %s", err)
			os.Exit(1)
		}
	}

	// Ensure interactive mode if missing required role ARNs on STS clusters
	if isSTS && !hasRoles && !interactive.Enabled() && supportRoleARN == "" {
		interactive.Enable()
	}
	if isSTS && !hasRoles && interactive.Enabled() {
		supportRoleARN, err = interactive.GetString(interactive.Input{
			Question: "Support Role ARN",
			Help:     cmd.Flags().Lookup("support-role-arn").Usage,
			Default:  supportRoleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid ARN: %s", err)
			os.Exit(1)
		}
	}
	if supportRoleARN != "" {
		err = aws.ARNValidator(supportRoleARN)
		if err != nil {
			r.Reporter.Errorf("Expected a valid Support Role ARN: %s", err)
			os.Exit(1)
		}
	} else if roleARN != "" {
		r.Reporter.Errorf("Support Role ARN is required: %s", err)
		os.Exit(1)
	}

	// Instance IAM Roles
	if !isHostedCP {
		// Ensure interactive mode if missing required role ARNs on STS clusters
		if isSTS && !hasRoles && !interactive.Enabled() && controlPlaneRoleARN == "" {
			interactive.Enable()
		}
		if isSTS && !hasRoles && interactive.Enabled() {
			controlPlaneRoleARN, err = interactive.GetString(interactive.Input{
				Question: "Control plane IAM Role ARN",
				Help:     cmd.Flags().Lookup("controlplane-iam-role").Usage,
				Default:  controlPlaneRoleARN,
				Required: true,
				Validators: []interactive.Validator{
					aws.ARNValidator,
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid control plane IAM role ARN: %s", err)
				os.Exit(1)
			}
		}
		if controlPlaneRoleARN != "" {
			err = aws.ARNValidator(controlPlaneRoleARN)
			if err != nil {
				r.Reporter.Errorf("Expected a valid control plane instance IAM role ARN: %s", err)
				os.Exit(1)
			}
		} else if roleARN != "" {
			r.Reporter.Errorf("Control plane instance IAM role ARN is required: %s", err)
			os.Exit(1)
		}
	}

	// Ensure interactive mode if missing required role ARNs on STS clusters
	if isSTS && !hasRoles && !interactive.Enabled() && workerRoleARN == "" {
		interactive.Enable()
	}

	if isSTS && !hasRoles && interactive.Enabled() {
		workerRoleARN, err = interactive.GetString(interactive.Input{
			Question: "Worker IAM Role ARN",
			Help:     cmd.Flags().Lookup("worker-iam-role").Usage,
			Default:  workerRoleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid worker IAM role ARN: %s", err)
			os.Exit(1)
		}
	}
	if workerRoleARN != "" {
		err = aws.ARNValidator(workerRoleARN)
		if err != nil {
			r.Reporter.Errorf("Expected a valid worker instance IAM role ARN: %s", err)
			os.Exit(1)
		}
	} else if roleARN != "" {
		r.Reporter.Errorf("Worker instance IAM role ARN is required: %s", err)
		os.Exit(1)
	}

	// combine role arns to list
	roleARNs := []string{
		roleARN,
		supportRoleARN,
		controlPlaneRoleARN,
		workerRoleARN,
	}

	managedPolicies, err := awsClient.HasManagedPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has managed policies: %v", err)
		os.Exit(1)
	}
	// check if role has hosted cp policy via AWS tag value
	hostedCPPolicies, err := awsClient.HasHostedCPPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has hosted CP policies: %v", err)
		os.Exit(1)
	}

	if managedPolicies {
		rolePrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
		if err != nil {
			r.Reporter.Errorf("Failed to find prefix from account role: %s", err)
			os.Exit(1)
		}

		err = roles.ValidateAccountRolesManagedPolicies(r, rolePrefix, hostedCPPolicies)
		if err != nil {
			r.Reporter.Errorf("Failed while validating account roles: %s", err)
			os.Exit(1)
		}
	} else {
		err = roles.ValidateUnmanagedAccountRoles(roleARNs, awsClient, version)
		if err != nil {
			r.Reporter.Errorf("Failed while validating account roles: %s", err)
			os.Exit(1)
		}
	}

	operatorRolesPrefix := args.operatorRolesPrefix
	expectedOperatorRolePath, _ := aws.GetPathFromARN(roleARN)
	operatorIAMRoles := args.operatorIAMRoles
	computedOperatorIamRoleList := []ocm.OperatorIAMRole{}
	if isSTS {
		if operatorRolesPrefix == "" {
			operatorRolesPrefix = getRolePrefix(clusterName)
		}
		if interactive.Enabled() {
			operatorRolesPrefix, err = interactive.GetString(interactive.Input{
				Question: "Operator roles prefix",
				Help:     cmd.Flags().Lookup("operator-roles-prefix").Usage,
				Required: true,
				Default:  operatorRolesPrefix,
				Validators: []interactive.Validator{
					interactive.RegExp(aws.RoleNameRE.String()),
					interactive.MaxLength(32),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
				os.Exit(1)
			}
		}
		if len(operatorRolesPrefix) == 0 {
			r.Reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
			os.Exit(1)
		}
		if len(operatorRolesPrefix) > 32 {
			r.Reporter.Errorf("Expected a prefix with no more than 32 characters")
			os.Exit(1)
		}
		if !aws.RoleNameRE.MatchString(operatorRolesPrefix) {
			r.Reporter.Errorf("Expected valid operator roles prefix matching %s", aws.RoleNameRE.String())
			os.Exit(1)
		}
	}

	var oidcConfig *v1.OidcConfig
	if isSTS {
		credRequests, err := r.OCMClient.GetCredRequests(isHostedCP)
		if err != nil {
			r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
			os.Exit(1)
		}
		accRolesPrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
		if err != nil {
			r.Reporter.Errorf("Failed to find prefix from account role: %s", err)
			os.Exit(1)
		}
		if expectedOperatorRolePath != "" && !output.HasFlag() && r.Reporter.IsTerminal() {
			r.Reporter.Infof("ARN path '%s' detected. This ARN path will be used for subsequent"+
				" created operator roles and policies, for the account roles with prefix '%s'",
				expectedOperatorRolePath, accRolesPrefix)
		}
		for _, operator := range credRequests {
			//If the cluster version is less than the supported operator version
			if operator.MinVersion() != "" {
				isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(version), operator.MinVersion())
				if err != nil {
					r.Reporter.Errorf("Error validating operator role '%s' version %s", operator.Name(), err)
					os.Exit(1)
				}
				if !isSupported {
					continue
				}
			}
			computedOperatorIamRoleList = append(computedOperatorIamRoleList, ocm.OperatorIAMRole{
				Name:      operator.Name(),
				Namespace: operator.Namespace(),
				RoleARN: aws.ComputeOperatorRoleArn(operatorRolesPrefix, operator,
					awsCreator, expectedOperatorRolePath),
			})
		}
		// If user insists on using the deprecated --operator-iam-roles
		// override the values to support the legacy documentation
		if cmd.Flags().Changed("operator-iam-roles") {
			computedOperatorIamRoleList = []ocm.OperatorIAMRole{}
			for _, role := range operatorIAMRoles {
				if !strings.Contains(role, ",") {
					r.Reporter.Errorf("Expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
					os.Exit(1)
				}
				roleData := strings.Split(role, ",")
				if len(roleData) != 3 {
					r.Reporter.Errorf("Expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
					os.Exit(1)
				}
				computedOperatorIamRoleList = append(computedOperatorIamRoleList, ocm.OperatorIAMRole{
					Name:      roleData[0],
					Namespace: roleData[1],
					RoleARN:   roleData[2],
				})
			}
		}
		oidcConfig = handleOidcConfigOptions(r, cmd, isSTS, isHostedCP)
		err = validateOperatorRolesAvailabilityUnderUserAwsAccount(awsClient, computedOperatorIamRoleList)
		if err != nil {
			if !oidcConfig.Reusable() {
				r.Reporter.Errorf("%v", err)
				os.Exit(1)
			} else {
				err = ocm.ValidateOperatorRolesMatchOidcProvider(r.Reporter, awsClient, computedOperatorIamRoleList,
					oidcConfig.IssuerUrl(), ocm.GetVersionMinor(version), expectedOperatorRolePath, managedPolicies)
				if err != nil {
					r.Reporter.Errorf("%v", err)
					os.Exit(1)
				}
			}
		}
	}

	// Custom tags for AWS resources
	_tags := args.tags
	tagsList := map[string]string{}
	if interactive.Enabled() {
		tagsInput, err := interactive.GetString(interactive.Input{
			Question: "Tags",
			Help:     cmd.Flags().Lookup("tags").Usage,
			Default:  strings.Join(_tags, ","),
			Validators: []interactive.Validator{
				aws.UserTagValidator,
				aws.UserTagDuplicateValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid set of tags: %s", err)
			os.Exit(1)
		}
		if len(tagsInput) > 0 {
			_tags = strings.Split(tagsInput, ",")
		}
	}
	if len(_tags) > 0 {
		if err := aws.UserTagValidator(_tags); err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		delim := aws.GetTagsDelimiter(_tags)
		for _, tag := range _tags {
			t := strings.Split(tag, delim)
			tagsList[t[0]] = strings.TrimSpace(t[1])
		}
	}

	// Multi-AZ:
	multiAZ := args.multiAZ
	if interactive.Enabled() && !isHostedCP {
		multiAZ, err = interactive.GetBool(interactive.Input{
			Question: "Multiple availability zones",
			Help:     cmd.Flags().Lookup("multi-az").Usage,
			Default:  multiAZ,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid multi-AZ value: %s", err)
			os.Exit(1)
		}
	}

	// Hosted clusters will be multiAZ by definition
	if isHostedCP {
		multiAZ = true
		if cmd.Flags().Changed("multi-az") {
			r.Reporter.Warnf("Hosted clusters deprecate the --multi-az flag. " +
				"The hosted control plane will be MultiAZ, machinepools will be created in the different private " +
				"subnets provided under --subnet-ids flag.")
		}
	}

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	// Filter regions by OCP version for displaying in interactive mode
	var versionFilter string
	if interactive.Enabled() {
		versionFilter = version
	} else {
		versionFilter = ""
	}
	regionList, regionAZ, err := r.OCMClient.GetRegionList(multiAZ, roleARN, externalID, versionFilter,
		awsClient, isHostedCP, shardPinningEnabled)
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	if region == "" {
		r.Reporter.Errorf("Expected a valid AWS region")
		os.Exit(1)
	} else if found := helper.Contains(regionList, region); isHostedCP && !shardPinningEnabled && !found {
		r.Reporter.Warnf("Region '%s' not currently available for Hosted Control Plane cluster.", region)
		interactive.Enable()
	}

	if interactive.Enabled() {
		region, err = interactive.GetOption(interactive.Input{
			Question: "AWS region",
			Help:     cmd.Flags().Lookup("region").Usage,
			Options:  regionList,
			Default:  region,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid AWS region: %s", err)
			os.Exit(1)
		}
	}
	if supportsMultiAZ, found := regionAZ[region]; found {
		if !supportsMultiAZ && multiAZ {
			r.Reporter.Errorf("Region '%s' does not support multiple availability zones", region)
			os.Exit(1)
		}
	} else {
		r.Reporter.Errorf("Region '%s' is not supported for this AWS account", region)
		os.Exit(1)
	}

	awsClient, err = aws.NewClient().
		Region(region).
		Logger(r.Logger).
		UseLocalCredentials(args.useLocalCredentials).
		Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create awsClient: %s", err)
		os.Exit(1)
	}
	r.AWSClient = awsClient

	// Cluster privacy:
	useExistingVPC := false
	private := args.private
	isPrivateHostedCP := isHostedCP && private // all private hosted clusters are private-link
	privateLink := args.privateLink || fedramp.Enabled() || isPrivateHostedCP

	privateLinkWarning := "Once the cluster is created, this option cannot be changed."
	if isSTS {
		privateLinkWarning = fmt.Sprintf("STS clusters can only be private if AWS PrivateLink is used. %s ",
			privateLinkWarning)
	}
	if interactive.Enabled() && !fedramp.Enabled() && !isPrivateHostedCP {
		privateLink, err = interactive.GetBool(interactive.Input{
			Question: "PrivateLink cluster",
			Help:     fmt.Sprintf("%s %s", cmd.Flags().Lookup("private-link").Usage, privateLinkWarning),
			Default:  privateLink || (isSTS && args.private),
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid private-link value: %s", err)
			os.Exit(1)
		}
	} else if (privateLink || (isSTS && private)) && !fedramp.Enabled() && !isPrivateHostedCP {
		// do not prompt users for privatelink if it is private hosted cluster
		r.Reporter.Warnf("You are choosing to use AWS PrivateLink for your cluster. %s", privateLinkWarning)
		if !confirm.Confirm("use AWS PrivateLink for cluster '%s'", clusterName) {
			os.Exit(0)
		}
		privateLink = true
	}

	if privateLink {
		private = true
	} else if isSTS && private {
		r.Reporter.Errorf("Private STS clusters are only supported through AWS PrivateLink")
		os.Exit(1)
	} else if !isSTS {
		privateWarning := "You will not be able to access your cluster until " +
			"you edit network settings in your cloud provider."
		if interactive.Enabled() {
			private, err = interactive.GetBool(interactive.Input{
				Question: "Private cluster",
				Help:     fmt.Sprintf("%s %s", cmd.Flags().Lookup("private").Usage, privateWarning),
				Default:  private,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid private value: %s", err)
				os.Exit(1)
			}
		} else if private {
			r.Reporter.Warnf("You are choosing to make your cluster private. %s", privateWarning)
			if !confirm.Confirm("set cluster '%s' as private", clusterName) {
				os.Exit(0)
			}
		}
	}

	if isSTS && private && !privateLink {
		r.Reporter.Errorf("Private STS clusters are only supported through AWS PrivateLink")
		os.Exit(1)
	}

	if privateLink || isHostedCP {
		useExistingVPC = true
	}

	// cluster-wide proxy values set here as we need to know whather to skip the "Install
	// into an existing VPC" question
	enableProxy := false
	httpProxy := args.httpProxy
	httpsProxy := args.httpsProxy
	noProxySlice := args.noProxySlice
	additionalTrustBundleFile := args.additionalTrustBundleFile
	if httpProxy != "" || httpsProxy != "" || len(noProxySlice) > 0 || additionalTrustBundleFile != "" {
		useExistingVPC = true
		enableProxy = true
	}

	dMachinecidr,
		dPodcidr,
		dServicecidr,
		dhostPrefix,
		defaultMachinePoolRootDiskSize,
		defaultComputeMachineType := r.OCMClient.
		GetDefaultClusterFlavors(args.flavour)
	if dMachinecidr == nil || dPodcidr == nil || dServicecidr == nil {
		r.Reporter.Errorf("Error retrieving default cluster flavors")
		os.Exit(1)
	}

	// Machine CIDR:
	machineCIDR := args.machineCIDR
	if ocm.IsEmptyCIDR(machineCIDR) {
		machineCIDR = *dMachinecidr
	}
	if interactive.Enabled() {
		machineCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Machine CIDR",
			Help:     cmd.Flags().Lookup("machine-cidr").Usage,
			Default:  machineCIDR,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}

	// Service CIDR:
	serviceCIDR := args.serviceCIDR
	if ocm.IsEmptyCIDR(serviceCIDR) {
		serviceCIDR = *dServicecidr
	}
	if interactive.Enabled() {
		serviceCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Service CIDR",
			Help:     cmd.Flags().Lookup("service-cidr").Usage,
			Default:  serviceCIDR,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}
	// Pod CIDR:
	podCIDR := args.podCIDR
	if ocm.IsEmptyCIDR(podCIDR) {
		podCIDR = *dPodcidr
	}
	if interactive.Enabled() {
		podCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Pod CIDR",
			Help:     cmd.Flags().Lookup("pod-cidr").Usage,
			Default:  podCIDR,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}

	// Subnet IDs
	subnetIDs := args.subnetIDs
	subnetsProvided := len(subnetIDs) > 0
	r.Reporter.Debugf("Received the following subnetIDs: %v", args.subnetIDs)
	// If the user has set the availability zones (allowed for non-BYOVPC clusters), don't prompt the BYOVPC message
	if !useExistingVPC && !subnetsProvided && !isAvailabilityZonesSet && interactive.Enabled() {
		existingVPCHelp := "To install into an existing VPC you need to ensure that your VPC is configured " +
			"with two subnets for each availability zone that you want the cluster installed into. "
		if privateLink {
			existingVPCHelp += "For PrivateLink, only a private subnet per availability zone is needed."
		}

		useExistingVPC, err = interactive.GetBool(interactive.Input{
			Question: "Install into an existing VPC",
			Help:     existingVPCHelp,
			Default:  useExistingVPC,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value: %s", err)
			os.Exit(1)
		}
	}

	if isHostedCP && !subnetsProvided && !useExistingVPC {
		r.Reporter.Errorf("All hosted clusters need a pre-configured VPC. Make sure to specify the subnet ids")
		os.Exit(1)
	}

	// For hosted cluster we will need the number of the private subnets the users has selected
	privateSubnetsCount := 0

	var availabilityZones []string
	var subnets []*ec2.Subnet
	mapSubnetIDToSubnet := make(map[string]aws.Subnet)
	if useExistingVPC || subnetsProvided {
		initialSubnets, err := getInitialValidSubnets(awsClient, r.Reporter)
		if err != nil {
			r.Reporter.Errorf("Failed to get the list of subnets: %s", err)
			os.Exit(1)
		}
		if subnetsProvided {
			useExistingVPC = true
		}
		_, machineNetwork, err := net.ParseCIDR(machineCIDR.String())
		if err != nil {
			r.Reporter.Errorf("Unable to parse machine CIDR")
			os.Exit(1)
		}
		_, serviceNetwork, err := net.ParseCIDR(serviceCIDR.String())
		if err != nil {
			r.Reporter.Errorf("Unable to parse service CIDR")
			os.Exit(1)
		}
		excludedSubnetsDueToCidr := []string{}
		for _, subnet := range initialSubnets {
			subnetIP, subnetNetwork, err := net.ParseCIDR(*subnet.CidrBlock)
			if err != nil {
				r.Reporter.Errorf("Unable to parse subnet CIDR")
				os.Exit(1)
			}

			if machineNetwork.Contains(subnetIP) &&
				!subnetNetwork.Contains(serviceNetwork.IP) &&
				!serviceNetwork.Contains(subnetIP) {
				subnets = append(subnets, subnet)
			} else {
				excludedSubnetsDueToCidr = append(excludedSubnetsDueToCidr, awssdk.StringValue(subnet.SubnetId))
			}
		}

		if len(subnets) != len(initialSubnets) {
			r.Reporter.Warnf("The following subnets have been excluded"+
				" because they do not fit into chosen CIDR ranges: %s", helper.SliceToSortedString(excludedSubnetsDueToCidr))
		}
		if len(subnets) == 0 {
			r.Reporter.Warnf("No subnets found in current region that are valid for the chosen CIDR ranges")
			if ok := confirm.Prompt(false, "Continue with default? A new RH Managed VPC will be created for your cluster"); !ok {
				os.Exit(1)
			}
			useExistingVPC = false
			subnetsProvided = false
		}
		mapAZCreated := make(map[string]bool)
		options := make([]string, len(subnets))
		defaultOptions := make([]string, len(subnetIDs))

		// Verify subnets provided exist.
		if subnetsProvided {
			for _, subnetArg := range subnetIDs {
				verifiedSubnet := false
				for _, subnet := range subnets {
					if awssdk.StringValue(subnet.SubnetId) == subnetArg {
						verifiedSubnet = true
					}
				}
				if !verifiedSubnet {
					r.Reporter.Errorf("Could not find the following subnet provided in region '%s': %s",
						r.AWSClient.GetRegion(), subnetArg)
					os.Exit(1)
				}
			}
		}

		mapVpcToSubnet := map[string][]*ec2.Subnet{}

		for _, subnet := range subnets {
			mapVpcToSubnet[*subnet.VpcId] = append(mapVpcToSubnet[*subnet.VpcId], subnet)
			subnetID := awssdk.StringValue(subnet.SubnetId)
			availabilityZone := awssdk.StringValue(subnet.AvailabilityZone)
			mapSubnetIDToSubnet[subnetID] = aws.Subnet{
				AvailabilityZone: availabilityZone,
				OwnerID:          awssdk.StringValue(subnet.OwnerId),
			}
			mapAZCreated[availabilityZone] = false
		}
		// Create the options to prompt the user.
		i := 0
		vpcIds := helper.MapKeys(mapVpcToSubnet)
		helper.SortStringRespectLength(vpcIds)
		for _, vpcId := range vpcIds {
			subnetList := mapVpcToSubnet[vpcId]
			for _, subnet := range subnetList {
				options[i] = aws.SetSubnetOption(subnet)
				i++
				if subnetsProvided && helper.Contains(subnetIDs, *subnet.SubnetId) {
					defaultOptions = append(defaultOptions, aws.SetSubnetOption(subnet))
				}
			}
		}
		if isHostedCP && !subnetsProvided {
			interactive.Enable()
		}
		if ((privateLink && !subnetsProvided) || interactive.Enabled()) &&
			len(options) > 0 && (!multiAZ || len(mapAZCreated) >= 3) {
			subnetIDs, err = interactive.GetMultipleOptions(interactive.Input{
				Question: "Subnet IDs",
				Help:     cmd.Flags().Lookup("subnet-ids").Usage,
				Required: false,
				Options:  options,
				Default:  defaultOptions,
				Validators: []interactive.Validator{
					interactive.SubnetsCountValidator(multiAZ, privateLink, isHostedCP),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected valid subnet IDs: %s", err)
				os.Exit(1)
			}
			for i, subnet := range subnetIDs {
				subnetIDs[i] = aws.ParseOption(subnet)
			}
		}

		// Validate subnets in the case the user has provided them using the `args.subnets`
		if useExistingVPC || subnetsProvided {
			if !isHostedCP {
				err = ocm.ValidateSubnetsCount(multiAZ, privateLink, len(subnetIDs))
			} else {
				// Hosted cluster should validate that
				// - Public hosted clusters have at least one public subnet
				// - Private hosted clusters have all subnets private
				privateSubnetsCount, err = ocm.ValidateHostedClusterSubnets(awsClient, privateLink, subnetIDs)
			}
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}

		for _, subnet := range subnetIDs {
			az := mapSubnetIDToSubnet[subnet].AvailabilityZone
			if !mapAZCreated[az] {
				availabilityZones = append(availabilityZones, az)
				mapAZCreated[az] = true
			}
		}
	}
	r.Reporter.Debugf("Found the following availability zones for the subnets provided: %v", availabilityZones)

	// validate flags for shared vpc
	isSharedVPC := false
	privateHostedZoneID := strings.Trim(args.privateHostedZoneID, " \t")
	sharedVPCRoleARN := strings.Trim(args.sharedVPCRoleARN, " \t")
	baseDomain := strings.Trim(args.baseDomain, " \t")
	if privateHostedZoneID != "" ||
		sharedVPCRoleARN != "" {
		isSharedVPC = true
	}

	if len(subnetIDs) == 0 && isSharedVPC {
		r.Reporter.Errorf("Installing a cluster into a shared VPC is only supported for BYO VPC clusters")
		os.Exit(1)
	}

	if isSubnetBelongToSharedVpc(r, awsCreator.AccountID, subnetIDs, mapSubnetIDToSubnet) {
		isSharedVPC = true
		if privateHostedZoneID == "" || sharedVPCRoleARN == "" || baseDomain == "" {
			if !interactive.Enabled() {
				interactive.Enable()
			}

			privateHostedZoneID, err = getPrivateHostedZoneID(cmd, privateHostedZoneID)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}

			sharedVPCRoleARN, err = getSharedVpcRoleArn(cmd, sharedVPCRoleARN)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}

			baseDomain, err = getBaseDomain(r, cmd, baseDomain)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}

	// Select availability zones for a non-BYOVPC cluster
	var selectAvailabilityZones bool
	if !useExistingVPC && !subnetsProvided {
		if isAvailabilityZonesSet {
			availabilityZones = args.availabilityZones
		}

		if !isAvailabilityZonesSet && interactive.Enabled() {
			selectAvailabilityZones, err = interactive.GetBool(interactive.Input{
				Question: "Select availability zones",
				Help:     cmd.Flags().Lookup("availability-zones").Usage,
				Default:  false,
				Required: false,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for select-availability-zones: %s", err)
				os.Exit(1)
			}

			if selectAvailabilityZones {
				optionsAvailabilityZones, err := awsClient.DescribeAvailabilityZones()
				if err != nil {
					r.Reporter.Errorf("Failed to get the list of the availability zone: %s", err)
					os.Exit(1)
				}

				availabilityZones, err = selectAvailabilityZonesInteractively(cmd, optionsAvailabilityZones, multiAZ)
				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}
		}

		if isAvailabilityZonesSet || selectAvailabilityZones {
			err = validateAvailabilityZones(multiAZ, availabilityZones, awsClient)
			if err != nil {
				r.Reporter.Errorf(fmt.Sprintf("%s", err))
				os.Exit(1)
			}
		}
	}

	enableCustomerManagedKey := args.enableCustomerManagedKey
	kmsKeyARN := args.kmsKeyARN

	if kmsKeyARN != "" {
		enableCustomerManagedKey = true
	}
	if interactive.Enabled() && !enableCustomerManagedKey {
		enableCustomerManagedKey, err = interactive.GetBool(interactive.Input{
			Question: "Enable Customer Managed key",
			Help:     cmd.Flags().Lookup("enable-customer-managed-key").Usage,
			Default:  enableCustomerManagedKey,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for enable-customer-managed-key: %s", err)
			os.Exit(1)
		}
	}

	if enableCustomerManagedKey && (kmsKeyARN == "" || interactive.Enabled()) {
		kmsKeyARN, err = interactive.GetString(interactive.Input{
			Question: "KMS Key ARN",
			Help:     cmd.Flags().Lookup("kms-key-arn").Usage,
			Default:  kmsKeyARN,
			Required: enableCustomerManagedKey,
			Validators: []interactive.Validator{
				interactive.RegExp(kmsArnRE.String()),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for kms-key-arn: %s", err)
			os.Exit(1)
		}
	}

	if kmsKeyARN != "" && !kmsArnRE.MatchString(kmsKeyARN) {
		r.Reporter.Errorf("Expected a valid value for kms-key-arn matching %s", kmsArnRE)
		os.Exit(1)
	}

	// Compute node instance type:
	computeMachineType := args.computeMachineType
	computeMachineTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(region, availabilityZones, roleARN,
		awsClient)
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	if computeMachineType == "" {
		computeMachineType = defaultComputeMachineType
	}
	if interactive.Enabled() {
		computeMachineType, err = interactive.GetOption(interactive.Input{
			Question: "Compute nodes instance type",
			Help:     cmd.Flags().Lookup("compute-machine-type").Usage,
			Options:  computeMachineTypeList.GetAvailableIDs(multiAZ),
			Default:  computeMachineType,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid machine type: %s", err)
			os.Exit(1)
		}
	}
	err = computeMachineTypeList.ValidateMachineType(computeMachineType, multiAZ)
	if err != nil {
		r.Reporter.Errorf("Expected a valid machine type: %s", err)
		os.Exit(1)
	}

	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isReplicasSet := cmd.Flags().Changed("compute-nodes") || cmd.Flags().Changed("replicas")

	// Autoscaling
	autoscaling := args.autoscalingEnabled
	if !isReplicasSet && !autoscaling && !isAutoscalingSet && interactive.Enabled() {
		autoscaling, err = interactive.GetBool(interactive.Input{
			Question: "Enable autoscaling",
			Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
			Default:  autoscaling,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for enable-autoscaling: %s", err)
			os.Exit(1)
		}
	}

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")

	minReplicas, maxReplicas := calculateReplicas(
		isMinReplicasSet,
		isMaxReplicasSet,
		args.minReplicas,
		args.maxReplicas,
		privateSubnetsCount,
		isHostedCP,
		multiAZ)

	var clusterAutoscaler *clusterautoscaler.AutoscalerArgs
	if !autoscaling {
		clusterAutoscaler = nil
	} else {
		// if the user set compute-nodes and enabled autoscaling
		if isReplicasSet {
			r.Reporter.Errorf("Compute-nodes can't be set when autoscaling is enabled")
			os.Exit(1)
		}
		if interactive.Enabled() || !isMinReplicasSet {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: true,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount)(minReplicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					maxReplicaValidator(multiAZ, minReplicas, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
		err = maxReplicaValidator(multiAZ, minReplicas, isHostedCP, privateSubnetsCount)(maxReplicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if isHostedCP {
			if clusterautoscaler.IsAutoscalerSetViaCLI(cmd.Flags(), clusterAutoscalerFlagsPrefix) {
				r.Reporter.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
				os.Exit(1)
			}
		} else {
			clusterAutoscaler, err = clusterautoscaler.GetAutoscalerOptions(
				cmd.Flags(), clusterAutoscalerFlagsPrefix, true, autoscalerArgs)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}

	// Compute nodes:
	computeNodes := args.computeNodes
	// Compute node requirements for multi-AZ clusters are higher
	if multiAZ && !autoscaling && !isReplicasSet {
		computeNodes = minReplicas
	}
	if !autoscaling {
		// if the user set min/max replicas and hasn't enabled autoscaling
		if isMinReplicasSet || isMaxReplicasSet {
			r.Reporter.Errorf("Autoscaling must be enabled in order to set min and max replicas")
			os.Exit(1)
		}

		if interactive.Enabled() {
			computeNodes, err = interactive.GetInt(interactive.Input{
				Question: "Compute nodes",
				Help:     cmd.Flags().Lookup("compute-nodes").Usage,
				Default:  computeNodes,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of compute nodes: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount)(computeNodes)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	// Worker machine pool labels
	labels := args.defaultMachinePoolLabels
	if interactive.Enabled() && !isHostedCP {
		labels, err = interactive.GetString(interactive.Input{
			Question: "Worker machine pool labels",
			Help:     cmd.Flags().Lookup("default-mp-labels").Usage,
			Default:  labels,
			Validators: []interactive.Validator{
				mpHelpers.LabelValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	labelMap, err := mpHelpers.ParseLabels(labels)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	isVersionCompatibleComputeSgIds, err := versions.IsGreaterThanOrEqual(
		version, ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay1)
	if err != nil {
		r.Reporter.Errorf("There was a problem checking version compatibility: %v", err)
		os.Exit(1)
	}
	additionalComputeSecurityGroupIds := args.additionalComputeSecurityGroupIds
	hasChangedComputeSGIdsFlag := cmd.Flags().Changed(additionalComputeSecurityGroupIdsFlag)
	if hasChangedComputeSGIdsFlag {
		if !useExistingVPC {
			r.Reporter.Errorf("Setting the `%s` flag is only allowed for BYO VPC clusters",
				additionalComputeSecurityGroupIdsFlag)
			os.Exit(1)
		}
		// HCP is still unsupported
		if isHostedCP {
			r.Reporter.Errorf("Parameter '%s' is not supported for Hosted Control Plane clusters",
				additionalComputeSecurityGroupIdsFlag)
			os.Exit(1)
		}
		if !isVersionCompatibleComputeSgIds {
			r.Reporter.Errorf("Parameter '%s' is not supported prior to version '%s'",
				additionalComputeSecurityGroupIdsFlag, ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay1)
			os.Exit(1)
		}
	} else if interactive.Enabled() && isVersionCompatibleComputeSgIds && useExistingVPC && !isHostedCP {
		vpcId := ""
		for _, subnet := range subnets {
			if awssdk.StringValue(subnet.SubnetId) == subnetIDs[0] {
				vpcId = awssdk.StringValue(subnet.VpcId)
			}
		}
		if vpcId == "" {
			r.Reporter.Warnf("Unexpected situation a VPC ID should have been selected based on chosen subnets")
			os.Exit(1)
		}
		additionalComputeSecurityGroupIds = interactiveSgs.
			GetSecurityGroupIds(r, cmd, vpcId, additionalComputeSecurityGroupIdsFlag)
	}
	for i, sg := range additionalComputeSecurityGroupIds {
		additionalComputeSecurityGroupIds[i] = strings.TrimSpace(sg)
	}

	// Validate all remaining flags:
	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	// Network Type:
	networkType := args.networkType
	if cmd.Flags().Changed("network-type") && interactive.Enabled() {
		networkType, err = interactive.GetOption(interactive.Input{
			Question: "Network Type",
			Help:     cmd.Flags().Lookup("network-type").Usage,
			Options:  ocm.NetworkTypes,
			Default:  networkType,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid network type: %s", err)
			os.Exit(1)
		}
	}

	// Host prefix:
	hostPrefix := args.hostPrefix
	if interactive.Enabled() {
		if hostPrefix == 0 {
			hostPrefix = dhostPrefix
		}
		hostPrefix, err = interactive.GetInt(interactive.Input{
			Question: "Host prefix",
			Help:     cmd.Flags().Lookup("host-prefix").Usage,
			Default:  hostPrefix,
			Validators: []interactive.Validator{
				hostPrefixValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid host prefix value: %s", err)
			os.Exit(1)
		}
	}
	err = hostPrefixValidator(hostPrefix)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	var machinePoolRootDisk *ocm.Volume
	if !isHostedCP &&
		(args.machinePoolRootDiskSize != "" || interactive.Enabled()) {
		var machinePoolRootDiskSizeStr string
		if args.machinePoolRootDiskSize == "" {
			// We don't need to parse the default since it's returned from the OCM API and AWS
			// always defaults to GiB
			machinePoolRootDiskSizeStr = helper.GigybyteStringer(defaultMachinePoolRootDiskSize)
		} else {
			machinePoolRootDiskSizeStr = args.machinePoolRootDiskSize
		}
		if interactive.Enabled() {
			// In order to avoid confusion, we want to display to the user what was passed as an
			// argument
			// Even if it was not valid, we want to display it to the user, then the CLI will show an
			// error and the value can be corrected
			// Also, if nothing is given, we want to display the default value fetched from the OCM API
			machinePoolRootDiskSizeStr, err = interactive.GetString(interactive.Input{
				Question: "Machine pool root disk size (GiB or TiB)",
				Help:     cmd.Flags().Lookup("worker-disk-size").Usage,
				Default:  machinePoolRootDiskSizeStr,
				Validators: []interactive.Validator{
					interactive.MachinePoolRootDiskSizeValidator(version),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid machine pool root disk size value: %v", err)
				os.Exit(1)
			}
		}

		// Parse the value given by either CLI or interactive mode and return it in GigiBytes
		machinePoolRootDiskSize, err := ocm.ParseDiskSizeToGigibyte(machinePoolRootDiskSizeStr)
		if err != nil {
			r.Reporter.Errorf("Expected a valid machine pool root disk size value: %v", err)
			os.Exit(1)
		}

		err = diskValidator.ValidateMachinePoolRootDiskSize(version, machinePoolRootDiskSize)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}

		// If the size given by the user is different than the default, we just let the OCM server
		// handle the default root disk size
		if machinePoolRootDiskSize != defaultMachinePoolRootDiskSize {
			machinePoolRootDisk = &ocm.Volume{
				Size: machinePoolRootDiskSize,
			}
		}
	}

	fips := args.fips || fedramp.Enabled()
	if interactive.Enabled() && !fedramp.Enabled() {
		fips, err = interactive.GetBool(interactive.Input{
			Question: "Enable FIPS support",
			Help:     cmd.Flags().Lookup("fips").Usage,
			Default:  fips,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid FIPS value: %v", err)
			os.Exit(1)
		}
	}

	etcdEncryption := args.etcdEncryption

	// validate and force etcd encryption
	if etcdEncryptionKmsARN != "" {
		if cmd.Flags().Changed("etcd-encryption") && !etcdEncryption {
			r.Reporter.Errorf("etcd encryption cannot be disabled when encryption kms arn is provided")
			os.Exit(1)
		} else {
			etcdEncryption = true
		}
	}

	if interactive.Enabled() && !(fips || etcdEncryptionKmsARN != "") {
		etcdEncryption, err = interactive.GetBool(interactive.Input{
			Question: "Encrypt etcd data",
			Help:     cmd.Flags().Lookup("etcd-encryption").Usage,
			Default:  etcdEncryption,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid etcd-encryption value: %v", err)
			os.Exit(1)
		}
	}
	if fips {
		if cmd.Flags().Changed("etcd-encryption") && !etcdEncryption {
			r.Reporter.Errorf("etcd encryption cannot be disabled on clusters with FIPS mode")
			os.Exit(1)
		} else {
			etcdEncryption = true
		}
	}

	if etcdEncryption && isHostedCP && (etcdEncryptionKmsARN == "" || interactive.Enabled()) {
		etcdEncryptionKmsARN, err = interactive.GetString(interactive.Input{
			Question: "Etcd encryption KMS ARN",
			Help:     cmd.Flags().Lookup("etcd-encryption-kms-arn").Usage,
			Default:  etcdEncryptionKmsARN,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(kmsArnRE.String()),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for etcd-encryption-kms-arn: %s", err)
			os.Exit(1)
		}
	}

	if etcdEncryptionKmsARN != "" && !kmsArnRE.MatchString(etcdEncryptionKmsARN) {
		r.Reporter.Errorf("Expected a valid value for etcd-encryption-kms-arn matching %s", kmsArnRE)
		os.Exit(1)
	}

	disableWorkloadMonitoring := args.disableWorkloadMonitoring
	if interactive.Enabled() {
		disableWorkloadMonitoring, err = interactive.GetBool(interactive.Input{
			Question: "Disable Workload monitoring",
			Help:     cmd.Flags().Lookup("disable-workload-monitoring").Usage,
			Default:  disableWorkloadMonitoring,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid disable-workload-monitoring value: %v", err)
			os.Exit(1)
		}
	}

	// Cluster-wide proxy configuration
	if (subnetsProvided || (useExistingVPC && !enableProxy)) && interactive.Enabled() {
		enableProxy, err = interactive.GetBool(interactive.Input{
			Question: "Use cluster-wide proxy",
			Help: "To install cluster-wide proxy, you need to set one of the following attributes: 'http-proxy', " +
				"'https-proxy', additional-trust-bundle",
			Default: enableProxy,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid proxy-enabled value: %s", err)
			os.Exit(1)
		}
	}

	if enableProxy && interactive.Enabled() {
		httpProxy, err = interactive.GetString(interactive.Input{
			Question: "HTTP proxy",
			Help:     cmd.Flags().Lookup("http-proxy").Usage,
			Default:  httpProxy,
			Validators: []interactive.Validator{
				ocm.ValidateHTTPProxy,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid http proxy: %s", err)
			os.Exit(1)
		}
	}
	err = ocm.ValidateHTTPProxy(httpProxy)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if enableProxy && interactive.Enabled() {
		httpsProxy, err = interactive.GetString(interactive.Input{
			Question: "HTTPS proxy",
			Help:     cmd.Flags().Lookup("https-proxy").Usage,
			Default:  httpsProxy,
			Validators: []interactive.Validator{
				interactive.IsURL,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid https proxy: %s", err)
			os.Exit(1)
		}
	}
	err = interactive.IsURL(httpsProxy)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if enableProxy && interactive.Enabled() {
		noProxyInput, err := interactive.GetString(interactive.Input{
			Question: "No proxy",
			Help:     cmd.Flags().Lookup("no-proxy").Usage,
			Default:  strings.Join(noProxySlice, ","),
			Validators: []interactive.Validator{
				aws.UserNoProxyValidator,
				aws.UserNoProxyDuplicateValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid set of no proxy domains/CIDR's: %s", err)
			os.Exit(1)
		}
		noProxySlice = helper.HandleEmptyStringOnSlice(strings.Split(noProxyInput, ","))
	}

	if len(noProxySlice) > 0 {
		duplicate, found := aws.HasDuplicates(noProxySlice)
		if found {
			r.Reporter.Errorf("Invalid no-proxy list, duplicate key '%s' found", duplicate)
			os.Exit(1)
		}
		for _, domain := range noProxySlice {
			err := aws.UserNoProxyValidator(domain)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}

	if httpProxy == "" && httpsProxy == "" && len(noProxySlice) > 0 {
		r.Reporter.Errorf("Expected at least one of the following: http-proxy, https-proxy")
		os.Exit(1)
	}

	if useExistingVPC && interactive.Enabled() {
		additionalTrustBundleFile, err = interactive.GetCert(interactive.Input{
			Question: "Additional trust bundle file path",
			Help:     cmd.Flags().Lookup("additional-trust-bundle-file").Usage,
			Default:  additionalTrustBundleFile,
			Validators: []interactive.Validator{
				ocm.ValidateAdditionalTrustBundle,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid additional trust bundle file name: %s", err)
			os.Exit(1)
		}
	}
	err = ocm.ValidateAdditionalTrustBundle(additionalTrustBundleFile)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Get certificate contents
	var additionalTrustBundle *string
	if additionalTrustBundleFile != "" {
		cert, err := os.ReadFile(additionalTrustBundleFile)
		if err != nil {
			r.Reporter.Errorf("Failed to read additional trust bundle file: %s", err)
			os.Exit(1)
		}
		additionalTrustBundle = new(string)
		*additionalTrustBundle = string(cert)
	}

	if enableProxy && httpProxy == "" && httpsProxy == "" && additionalTrustBundleFile == "" {
		r.Reporter.Errorf("Expected at least one of the following: http-proxy, https-proxy, additional-trust-bundle")
		os.Exit(1)
	}

	// Audit Log Forwarding
	auditLogRoleARN := args.AuditLogRoleARN

	if auditLogRoleARN != "" && !isHostedCP {
		r.Reporter.Errorf("Audit log forwarding to AWS CloudWatch is only supported for Hosted Control Plane clusters")
		os.Exit(1)
	}

	if interactive.Enabled() && isHostedCP {
		requestAuditLogForwarding, err := interactive.GetBool(interactive.Input{
			Question: "Enable audit log forwarding to AWS CloudWatch",
			Default:  false,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value: %s", err)
			os.Exit(1)
		}
		if requestAuditLogForwarding {

			r.Reporter.Infof("To configure the audit log forwarding role in your AWS account, " +
				"please refer to steps 1 through 6: https://access.redhat.com/solutions/7002219")

			auditLogRoleARN, err = interactive.GetString(interactive.Input{
				Question: "Audit log forwarding role ARN",
				Help:     cmd.Flags().Lookup("audit-log-arn").Usage,
				Default:  auditLogRoleARN,
				Required: true,
				Validators: []interactive.Validator{
					interactive.RegExp(aws.RoleArnRE.String()),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for audit-log-arn: %s", err)
				os.Exit(1)
			}
		} else {
			auditLogRoleARN = ""
		}
	}

	if auditLogRoleARN != "" && !aws.RoleArnRE.MatchString(auditLogRoleARN) {
		r.Reporter.Errorf("Expected a valid value for audit log arn matching %s", aws.RoleArnRE)
		os.Exit(1)
	}

	isVersionCompatibleManagedIngressV2, err := versions.IsGreaterThanOrEqual(
		version, ocm.MinVersionForManagedIngressV2)
	if err != nil {
		r.Reporter.Errorf("There was a problem checking version compatibility: %v", err)
		os.Exit(1)
	}
	routeSelector := ""
	routeSelectors := map[string]string{}
	excludedNamespaces := ""
	sliceExcludedNamespaces := []string{}
	wildcardPolicy := ""
	namespaceOwnershipPolicy := ""
	if isVersionCompatibleManagedIngressV2 {
		if cmd.Flags().Changed(defaultIngressRouteSelectorFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating route selectors is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			routeSelector = args.defaultIngressRouteSelectors
		} else if interactive.Enabled() && !isHostedCP {
			routeSelectorArg, err := interactive.GetString(interactive.Input{
				Question: "Route Selector for ingress",
				Help:     cmd.Flags().Lookup(defaultIngressRouteSelectorFlag).Usage,
				Default:  args.defaultIngressRouteSelectors,
				Validators: []interactive.Validator{
					func(routeSelector interface{}) error {
						_, err := ingress.GetRouteSelector(routeSelector.(string))
						return err
					},
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
				os.Exit(1)
			}
			routeSelector = routeSelectorArg
		}
		routeSelectors, err = ingress.GetRouteSelector(routeSelector)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if cmd.Flags().Changed(defaultIngressExcludedNamespacesFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating excluded namespace is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			excludedNamespaces = args.defaultIngressExcludedNamespaces
		} else if interactive.Enabled() && !isHostedCP {
			excludedNamespacesArg, err := interactive.GetString(interactive.Input{
				Question: "Excluded namespaces for ingress",
				Help:     cmd.Flags().Lookup(defaultIngressExcludedNamespacesFlag).Usage,
				Default:  args.defaultIngressExcludedNamespaces,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
				os.Exit(1)
			}
			excludedNamespaces = excludedNamespacesArg
		}
		sliceExcludedNamespaces = ingress.GetExcludedNamespaces(excludedNamespaces)

		if cmd.Flags().Changed(defaultIngressWildcardPolicyFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating Wildcard Policy is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			wildcardPolicy = args.defaultIngressWildcardPolicy
		} else {
			if interactive.Enabled() && !isHostedCP {
				wildcardPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Wildcard Policy",
					Options:  ingress.ValidWildcardPolicies,
					Help:     cmd.Flags().Lookup(defaultIngressWildcardPolicyFlag).Usage,
					Default:  args.defaultIngressWildcardPolicy,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid Wildcard Policy: %s", err)
					os.Exit(1)
				}
				wildcardPolicy = wildcardPolicyArg
			}
		}

		if cmd.Flags().Changed(defaultIngressNamespaceOwnershipPolicyFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating Namespace Ownership Policy is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			namespaceOwnershipPolicy = args.defaultIngressNamespaceOwnershipPolicy
		} else {
			if interactive.Enabled() && !isHostedCP {
				namespaceOwnershipPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Namespace Ownership Policy",
					Options:  ingress.ValidNamespaceOwnershipPolicies,
					Help:     cmd.Flags().Lookup(defaultIngressNamespaceOwnershipPolicyFlag).Usage,
					Default:  args.defaultIngressNamespaceOwnershipPolicy,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid Namespace Ownership Policy: %s", err)
					os.Exit(1)
				}
				namespaceOwnershipPolicy = namespaceOwnershipPolicyArg
			}
		}
	}

	clusterConfig := ocm.Spec{
		Name:                      clusterName,
		Region:                    region,
		MultiAZ:                   multiAZ,
		Version:                   version,
		ChannelGroup:              channelGroup,
		Flavour:                   args.flavour,
		FIPS:                      fips,
		EtcdEncryption:            etcdEncryption,
		EtcdEncryptionKMSArn:      etcdEncryptionKmsARN,
		EnableProxy:               enableProxy,
		AdditionalTrustBundle:     additionalTrustBundle,
		Expiration:                expiration,
		ComputeMachineType:        computeMachineType,
		ComputeNodes:              computeNodes,
		Autoscaling:               autoscaling,
		MinReplicas:               minReplicas,
		MaxReplicas:               maxReplicas,
		ComputeLabels:             labelMap,
		NetworkType:               networkType,
		MachineCIDR:               machineCIDR,
		ServiceCIDR:               serviceCIDR,
		PodCIDR:                   podCIDR,
		HostPrefix:                hostPrefix,
		Private:                   &private,
		DryRun:                    &args.dryRun,
		DisableSCPChecks:          &args.disableSCPChecks,
		AvailabilityZones:         availabilityZones,
		SubnetIds:                 subnetIDs,
		PrivateLink:               &privateLink,
		IsSTS:                     isSTS,
		RoleARN:                   roleARN,
		ExternalID:                externalID,
		SupportRoleARN:            supportRoleARN,
		OperatorIAMRoles:          computedOperatorIamRoleList,
		ControlPlaneRoleARN:       controlPlaneRoleARN,
		WorkerRoleARN:             workerRoleARN,
		Mode:                      mode,
		Tags:                      tagsList,
		KMSKeyArn:                 kmsKeyARN,
		DisableWorkloadMonitoring: &disableWorkloadMonitoring,
		Hypershift: ocm.Hypershift{
			Enabled: isHostedCP,
		},
		BillingAccount:  billingAccount,
		AuditLogRoleARN: &auditLogRoleARN,
		DefaultIngress: ocm.DefaultIngressSpec{
			RouteSelectors:           routeSelectors,
			ExcludedNamespaces:       sliceExcludedNamespaces,
			WildcardPolicy:           wildcardPolicy,
			NamespaceOwnershipPolicy: namespaceOwnershipPolicy,
		},
		MachinePoolRootDisk:               machinePoolRootDisk,
		AdditionalComputeSecurityGroupIds: additionalComputeSecurityGroupIds,
	}

	if httpTokens != "" {
		clusterConfig.Ec2MetadataHttpTokens = v1.Ec2MetadataHttpTokens(httpTokens)
	}
	if oidcConfig != nil {
		clusterConfig.OidcConfigId = oidcConfig.ID()
	}

	if httpProxy != "" {
		clusterConfig.HTTPProxy = &httpProxy
	}
	if httpsProxy != "" {
		clusterConfig.HTTPSProxy = &httpsProxy
	}
	if len(noProxySlice) > 0 {
		str := strings.Join(noProxySlice, ",")
		clusterConfig.NoProxy = &str
	}
	if additionalTrustBundleFile != "" {
		clusterConfig.AdditionalTrustBundleFile = &additionalTrustBundleFile
		clusterConfig.AdditionalTrustBundle = additionalTrustBundle
	}
	if isClusterAdmin {
		clusterConfig.ClusterAdminUser = admin.ClusterAdminUsername
		clusterConfig.ClusterAdminPassword = clusterAdminPassword
	}
	if isSharedVPC {
		clusterConfig.PrivateHostedZoneID = privateHostedZoneID
		clusterConfig.SharedVPCRoleArn = sharedVPCRoleARN
		clusterConfig.BaseDomain = baseDomain
	}
	if clusterAutoscaler != nil {
		autoscalerConfig, err := clusterautoscaler.CreateAutoscalerConfig(clusterAutoscaler)
		if err != nil {
			r.Reporter.Errorf("Failed creating autoscaler configuration: %s", err)
			os.Exit(1)
		}

		clusterConfig.AutoscalerConfig = autoscalerConfig
	}

	props := args.properties
	if args.fakeCluster {
		props = append(props, properties.FakeCluster)
	}
	if args.useLocalCredentials {
		if isSTS {
			r.Reporter.Errorf("Local credentials are not supported for STS clusters")
			os.Exit(1)
		}
		props = append(props, properties.UseLocalCredentials)
	}
	if len(props) > 0 {
		clusterConfig.CustomProperties = map[string]string{}
	}
	for _, prop := range props {
		if strings.Contains(prop, ":") {
			p := strings.SplitN(prop, ":", 2)
			clusterConfig.CustomProperties[p[0]] = p[1]
		} else {
			clusterConfig.CustomProperties[prop] = "true"
		}
	}

	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating cluster '%s'", clusterName)
		if interactive.Enabled() {
			command := buildCommand(clusterConfig, operatorRolesPrefix, expectedOperatorRolePath,
				isAvailabilityZonesSet || selectAvailabilityZones, labels)
			r.Reporter.Infof("To create this cluster again in the future, you can run:\n   %s", command)
		}
		r.Reporter.Infof("To view a list of clusters and their status, run 'rosa list clusters'")
	}

	cluster, err := r.OCMClient.CreateCluster(clusterConfig)
	if err != nil {
		if args.dryRun {
			r.Reporter.Errorf("Creating cluster '%s' should fail: %s", clusterName, err)
		} else {
			r.Reporter.Errorf("Failed to create cluster: %s", err)
		}
		os.Exit(1)
	}

	if args.dryRun {
		r.Reporter.Infof(
			"Creating cluster '%s' should succeed. Run without the '--dry-run' flag to create the cluster.",
			clusterName)
		os.Exit(0)
	}

	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Cluster '%s' has been created.", clusterName)
		r.Reporter.Infof(
			"Once the cluster is installed you will need to add an Identity Provider " +
				"before you can login into the cluster. See 'rosa create idp --help' " +
				"for more information.")
	}

	clusterdescribe.Cmd.Run(clusterdescribe.Cmd, []string{cluster.ID()})

	if isSTS {
		if mode != "" {
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create operator roles.")
			}
			err := operatorroles.Cmd.RunE(operatorroles.Cmd, []string{clusterName, mode, permissionsBoundary})
			if err != nil {
				r.Reporter.Errorf("There was a problem creating operator roles: %v", err)
				os.Exit(1)
			}
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create OIDC Provider.")
			}
			oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{clusterName, mode, ""})
		} else {
			output := ""
			if cluster.AWS().STS().OidcConfig().Reusable() {
				output = "When using reusable OIDC Config and resources have been created " +
					"prior to cluster specification, this step is not required."
			}
			rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", clusterName)
			if permissionsBoundary != "" {
				rolesCMD = fmt.Sprintf("%s --permissions-boundary %s", rolesCMD, permissionsBoundary)
			}
			oidcCMD := "rosa create oidc-provider"
			oidcCMD = fmt.Sprintf("%s --cluster %s", oidcCMD, clusterName)
			output += "\nRun the following commands to continue the cluster creation:\n\n"
			output = fmt.Sprintf("%s\t%s\n", output, rolesCMD)
			output = fmt.Sprintf("%s\t%s\n", output, oidcCMD)
			r.Reporter.Infof(output)
		}
	}

	if args.watch {
		installLogs.Cmd.Run(installLogs.Cmd, []string{clusterName})
	} else if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof(
			"To determine when your cluster is Ready, run 'rosa describe cluster -c %s'.",
			clusterName,
		)
		r.Reporter.Infof(
			"To watch your cluster installation logs, run 'rosa logs install -c %s --watch'.",
			clusterName,
		)
	}
}

func validateOperatorRolesAvailabilityUnderUserAwsAccount(awsClient aws.Client,
	operatorIAMRoleList []ocm.OperatorIAMRole) error {
	for _, role := range operatorIAMRoleList {
		name, err := aws.GetResourceIdFromARN(role.RoleARN)
		if err != nil {
			return err
		}

		err = awsClient.ValidateRoleNameAvailable(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleOidcConfigOptions(r *rosa.Runtime, cmd *cobra.Command, isSTS bool, isHostedCP bool) *v1.OidcConfig {
	if !isSTS {
		return nil
	}
	oidcConfigId := args.oidcConfigId
	isOidcConfig := false
	if isHostedCP && !args.classicOidcConfig {
		isOidcConfig = true
		if oidcConfigId == "" {
			interactive.Enable()
		}
	}
	if oidcConfigId == "" && interactive.Enabled() {
		if !isHostedCP {
			_isOidcConfig, err := interactive.GetBool(interactive.Input{
				Question: "Deploy cluster using pre registered OIDC Configuration ID",
				Default:  true,
				Required: true,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value: %s", err)
				os.Exit(1)
			}
			isOidcConfig = _isOidcConfig
		}
		if isOidcConfig {
			oidcConfigId = interactiveOidc.GetOidcConfigID(r, cmd)
		}
	}
	if oidcConfigId == "" {
		if !isHostedCP {
			if isOidcConfig {
				r.Reporter.Warnf("No OIDC Configuration found; will continue with the classic flow.")
			}
			return nil
		}
		if args.classicOidcConfig {
			return nil
		}
		r.Reporter.Errorf("Hosted Control Plane requires an OIDC Configuration ID\n" +
			"Please run `rosa create oidc-config -h` and create one.")
		os.Exit(1)
	}
	oidcConfig, err := r.OCMClient.GetOidcConfig(oidcConfigId)
	if err != nil {
		r.Reporter.Errorf("There was a problem retrieving OIDC Config '%s': %v", oidcConfigId, err)
		os.Exit(1)
	}
	return oidcConfig
}

func minReplicaValidator(multiAZ bool, isHostedCP bool, privateSubnetsCount int) interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}

		return clustervalidations.MinReplicasValidator(minReplicas, multiAZ, isHostedCP, privateSubnetsCount)
	}
}

func maxReplicaValidator(multiAZ bool, minReplicas int, isHostedCP bool,
	privateSubnetsCount int) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		return clustervalidations.MaxReplicasValidator(minReplicas, maxReplicas, multiAZ, isHostedCP, privateSubnetsCount)
	}
}

const (
	HostPrefixMin = 23
	HostPrefixMax = 26
)

func hostPrefixValidator(val interface{}) error {
	hostPrefix, err := strconv.Atoi(fmt.Sprintf("%v", val))
	if err != nil {
		return err
	}
	if hostPrefix == 0 {
		return nil
	}
	if hostPrefix < HostPrefixMin || hostPrefix > HostPrefixMax {
		return fmt.Errorf(
			"Invalid Network Host Prefix /%d: Subnet length should be between %d and %d",
			hostPrefix, HostPrefixMin, HostPrefixMax)
	}
	return nil
}

func getAccountRolePrefix(hostedCPPolicies bool, roleARN string, roleType string) (string, error) {

	accountRoles := aws.AccountRoles

	if hostedCPPolicies {
		accountRoles = aws.HCPAccountRoles
	}

	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		return "", err
	}
	rolePrefix := aws.TrimRoleSuffix(roleName, fmt.Sprintf("-%s-Role", accountRoles[roleType].Name))
	return rolePrefix, nil
}

func evaluateDuration(duration time.Duration) time.Time {
	// round up to the nearest second
	return time.Now().Add(duration).Round(time.Second)
}

func validateExpiration() (expiration time.Time, err error) {
	// Validate options
	if len(args.expirationTime) > 0 && args.expirationDuration != 0 {
		err = errors.New("At most one of 'expiration-time' or 'expiration' may be specified")
		return
	}

	// Parse the expiration options
	if len(args.expirationTime) > 0 {
		t, err := parseRFC3339(args.expirationTime)
		if err != nil {
			err = fmt.Errorf("Failed to parse expiration-time: %s", err)
			return expiration, err
		}

		expiration = t
	}
	if args.expirationDuration != 0 {
		expiration = evaluateDuration(args.expirationDuration)
	}

	return
}

func selectAvailabilityZonesInteractively(cmd *cobra.Command, optionsAvailabilityZones []string,
	multiAZ bool) ([]string, error) {
	var availabilityZones []string
	var err error

	if multiAZ {
		availabilityZones, err = interactive.GetMultipleOptions(interactive.Input{
			Question: "Availability zones",
			Help:     cmd.Flags().Lookup("availability-zones").Usage,
			Required: true,
			Options:  optionsAvailabilityZones,
			Validators: []interactive.Validator{
				interactive.AvailabilityZonesCountValidator(multiAZ),
			},
		})
		if err != nil {
			return availabilityZones, fmt.Errorf("Expected valid availability zones: %s", err)
		}
	} else {
		var availabilityZone string
		availabilityZone, err = interactive.GetOption(interactive.Input{
			Question: "Availability zone",
			Help:     cmd.Flags().Lookup("availability-zones").Usage,
			Required: true,
			Options:  optionsAvailabilityZones,
		})
		if err != nil {
			return availabilityZones, fmt.Errorf("Expected valid availability zone: %s", err)
		}
		availabilityZones = append(availabilityZones, availabilityZone)
	}

	return availabilityZones, nil
}

func validateAvailabilityZones(multiAZ bool, availabilityZones []string, awsClient aws.Client) error {
	err := clustervalidations.ValidateAvailabilityZonesCount(multiAZ, len(availabilityZones))
	if err != nil {
		return err
	}

	regionAvailabilityZones, err := awsClient.DescribeAvailabilityZones()
	if err != nil {
		return fmt.Errorf("Failed to get the list of the availability zone: %s", err)
	}
	for _, az := range availabilityZones {
		if !helper.Contains(regionAvailabilityZones, az) {
			return fmt.Errorf("Expected a valid availability zone, "+
				"'%s' doesn't belong to region '%s' availability zones", az, awsClient.GetRegion())
		}
	}

	return nil
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func buildCommand(spec ocm.Spec, operatorRolesPrefix string,
	operatorRolePath string, userSelectedAvailabilityZones bool, labels string) string {
	command := "rosa create cluster"
	command += fmt.Sprintf(" --cluster-name %s", spec.Name)
	if spec.IsSTS {
		command += " --sts"
		if spec.Mode != "" {
			command += fmt.Sprintf(" --mode %s", spec.Mode)
		}
	}
	if spec.ClusterAdminUser != "" {
		// Checks if admin password is from user (both flag and interactive)
		if args.clusterAdminPassword != "" && spec.ClusterAdminPassword != "" {
			command += fmt.Sprintf(" --cluster-admin-password %s", spec.ClusterAdminPassword)
		} else {
			command += " --create-admin-user"
		}
	}
	if spec.RoleARN != "" {
		command += fmt.Sprintf(" --role-arn %s", spec.RoleARN)
		command += fmt.Sprintf(" --support-role-arn %s", spec.SupportRoleARN)
		if !spec.Hypershift.Enabled {
			command += fmt.Sprintf(" --controlplane-iam-role %s", spec.ControlPlaneRoleARN)
		}
		command += fmt.Sprintf(" --worker-iam-role %s", spec.WorkerRoleARN)
	}
	if spec.ExternalID != "" {
		command += fmt.Sprintf(" --external-id %s", spec.ExternalID)
	}
	if operatorRolesPrefix != "" {
		command += fmt.Sprintf(" --operator-roles-prefix %s", operatorRolesPrefix)
	}
	if spec.OidcConfigId != "" {
		command += fmt.Sprintf(" --%s %s", OidcConfigIdFlag, spec.OidcConfigId)
	}
	if args.classicOidcConfig {
		command += fmt.Sprintf(" --%s", ClassicOidcConfigFlag)
	}
	if len(spec.Tags) > 0 {
		command += fmt.Sprintf(" --tags \"%s\"", strings.Join(buildTagsCommand(spec.Tags), ","))
	}
	if spec.MultiAZ && !spec.Hypershift.Enabled {
		command += " --multi-az"
	}
	if spec.Region != "" {
		command += fmt.Sprintf(" --region %s", spec.Region)
	}
	if spec.DisableSCPChecks != nil && *spec.DisableSCPChecks {
		command += " --disable-scp-checks"
	}
	if spec.Version != "" {
		commandVersion := ocm.GetRawVersionId(spec.Version)
		if spec.ChannelGroup != ocm.DefaultChannelGroup {
			command += fmt.Sprintf(" --channel-group %s", spec.ChannelGroup)
		}
		command += fmt.Sprintf(" --version %s", commandVersion)
	}

	// Only account for expiration duration, as a fixed date may be obsolete if command is re-run later
	if args.expirationDuration != 0 {
		command += fmt.Sprintf(" --expiration %s", args.expirationDuration)
	}

	if spec.Autoscaling {
		command += " --enable-autoscaling"
		if spec.MinReplicas > 0 {
			command += fmt.Sprintf(" --min-replicas %d", spec.MinReplicas)
		}
		if spec.MaxReplicas > 0 {
			command += fmt.Sprintf(" --max-replicas %d", spec.MaxReplicas)
		}
	} else {
		if spec.ComputeNodes != 0 {
			command += fmt.Sprintf(" --replicas %d", spec.ComputeNodes)
		}
	}
	if spec.ComputeMachineType != "" {
		command += fmt.Sprintf(" --compute-machine-type %s", spec.ComputeMachineType)
	}

	if len(spec.ComputeLabels) != 0 {
		command += fmt.Sprintf(" --default-mp-labels \"%s\"", labels)
	}

	if spec.NetworkType != "" {
		command += fmt.Sprintf(" --network-type %s", spec.NetworkType)
	}
	if !ocm.IsEmptyCIDR(spec.MachineCIDR) {
		command += fmt.Sprintf(" --machine-cidr %s", spec.MachineCIDR.String())
	}
	if !ocm.IsEmptyCIDR(spec.ServiceCIDR) {
		command += fmt.Sprintf(" --service-cidr %s", spec.ServiceCIDR.String())
	}
	if !ocm.IsEmptyCIDR(spec.PodCIDR) {
		command += fmt.Sprintf(" --pod-cidr %s", spec.PodCIDR.String())
	}
	if spec.HostPrefix != 0 {
		command += fmt.Sprintf(" --host-prefix %d", spec.HostPrefix)
	}
	if spec.PrivateLink != nil && *spec.PrivateLink {
		command += " --private-link"
	} else if spec.Private != nil && *spec.Private {
		command += " --private"
	}
	if len(spec.SubnetIds) > 0 {
		command += fmt.Sprintf(" --subnet-ids %s", strings.Join(spec.SubnetIds, ","))
	}
	if spec.PrivateHostedZoneID != "" {
		command += fmt.Sprintf(" --private-hosted-zone-id %s", spec.PrivateHostedZoneID)
		command += fmt.Sprintf(" --shared-vpc-role-arn %s", spec.SharedVPCRoleArn)
		command += fmt.Sprintf(" --base-domain %s", spec.BaseDomain)
	}
	if spec.FIPS {
		command += " --fips"
	} else if spec.EtcdEncryption {
		command += " --etcd-encryption"
	}

	if spec.EnableProxy {
		if spec.HTTPProxy != nil && *spec.HTTPProxy != "" {
			command += fmt.Sprintf(" --http-proxy %s", *spec.HTTPProxy)
		}
		if spec.HTTPSProxy != nil && *spec.HTTPSProxy != "" {
			command += fmt.Sprintf(" --https-proxy %s", *spec.HTTPSProxy)
		}
		if spec.NoProxy != nil && *spec.NoProxy != "" {
			command += fmt.Sprintf(" --no-proxy \"%s\"", *spec.NoProxy)
		}
	}
	if spec.AdditionalTrustBundleFile != nil && *spec.AdditionalTrustBundleFile != "" {
		command += fmt.Sprintf(" --additional-trust-bundle-file %s", *spec.AdditionalTrustBundleFile)
	}
	if spec.KMSKeyArn != "" {
		command += fmt.Sprintf(" --kms-key-arn %s", spec.KMSKeyArn)
	}
	if spec.DisableWorkloadMonitoring != nil && *spec.DisableWorkloadMonitoring {
		command += " --disable-workload-monitoring"
	}
	if userSelectedAvailabilityZones {
		command += fmt.Sprintf(" --availability-zones %s", strings.Join(spec.AvailabilityZones, ","))
	}
	if spec.Hypershift.Enabled {
		command += " --hosted-cp"
	}
	if spec.EtcdEncryptionKMSArn != "" {
		command += fmt.Sprintf(" --etcd-encryption-kms-arn %s", spec.EtcdEncryptionKMSArn)
	}

	if spec.AuditLogRoleARN != nil && *spec.AuditLogRoleARN != "" {
		command += fmt.Sprintf(" --audit-log-arn %s", *spec.AuditLogRoleARN)
	}
	if spec.MachinePoolRootDisk != nil {
		machinePoolRootDiskSize := spec.MachinePoolRootDisk.Size
		if machinePoolRootDiskSize != 0 {
			command += fmt.Sprintf(" --worker-disk-size %dGiB", machinePoolRootDiskSize)
		}
	}

	if !reflect.DeepEqual(spec.DefaultIngress, ocm.NewDefaultIngressSpec()) {
		if len(spec.DefaultIngress.RouteSelectors) != 0 {
			selectors := []string{}
			for k, v := range spec.DefaultIngress.RouteSelectors {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			command += fmt.Sprintf(" --%s %s", defaultIngressRouteSelectorFlag, strings.Join(selectors, ","))
		}
		if len(spec.DefaultIngress.ExcludedNamespaces) != 0 {
			command += fmt.Sprintf(" --%s %s", defaultIngressExcludedNamespacesFlag,
				strings.Join(spec.DefaultIngress.ExcludedNamespaces, ","))
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.WildcardPolicy) {
			command += fmt.Sprintf(" --%s %s", defaultIngressWildcardPolicyFlag, spec.DefaultIngress.WildcardPolicy)
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.NamespaceOwnershipPolicy) {
			command += fmt.Sprintf(" --%s %s", defaultIngressNamespaceOwnershipPolicyFlag,
				spec.DefaultIngress.NamespaceOwnershipPolicy)
		}
	}

	command += clusterautoscaler.BuildAutoscalerOptions(spec.AutoscalerConfig, clusterAutoscalerFlagsPrefix)

	if len(spec.AdditionalComputeSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			additionalComputeSecurityGroupIdsFlag, strings.Join(spec.AdditionalComputeSecurityGroupIds, ","))
	}

	return command
}

func buildTagsCommand(tags map[string]string) []string {
	// set correct delim, if a key or value contains `:` the delim should be " "
	delim := ":"
	for k, v := range tags {
		if strings.Contains(k, ":") || strings.Contains(v, ":") {
			delim = " "
			break
		}
	}

	// build list of formatted tags to return in command
	var formattedTags []string
	for k, v := range tags {
		formattedTags = append(formattedTags, fmt.Sprintf("%s%s%s", k, delim, v))
	}
	return formattedTags
}

func getRolePrefix(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, helper.RandomLabel(4))
}

func calculateReplicas(
	isMinReplicasSet bool,
	isMaxReplicasSet bool,
	minReplicas int,
	maxReplicas int,
	privateSubnetsCount int,
	isHostedCP bool,
	multiAZ bool) (int, int) {

	newMinReplicas := minReplicas
	newMaxReplicas := maxReplicas

	if !isMinReplicasSet {
		if isHostedCP {
			newMinReplicas = privateSubnetsCount
			if privateSubnetsCount == 1 {
				newMinReplicas = MinReplicasSingleAZ
			}
		} else {
			if multiAZ {
				newMinReplicas = MinReplicaMultiAZ
			}
		}
	}
	if !isMaxReplicasSet && multiAZ {
		newMaxReplicas = newMinReplicas
	}

	return newMinReplicas, newMaxReplicas
}

func getExpectedResourceIDForAccRole(hostedCPPolicies bool, roleARN string, roleType string) (string, string, error) {

	accountRoles := aws.AccountRoles

	rolePrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
	if err != nil {
		return "", "", err
	}

	if hostedCPPolicies {
		accountRoles = aws.HCPAccountRoles
	}

	return strings.ToLower(fmt.Sprintf("%s-%s-Role", rolePrefix, accountRoles[roleType].Name)), rolePrefix, nil
}

func getInitialValidSubnets(awsClient aws.Client, reporter *reporter.Object) ([]*ec2.Subnet, error) {
	initialValidSubnets := []*ec2.Subnet{}
	excludedSubnets := []string{}
	allSubnets, err := awsClient.GetSubnetIDs()
	if err != nil {
		return initialValidSubnets, err
	}
	for _, subnet := range allSubnets {
		hasRHManaged := tags.Ec2ResourceHasTag(subnet.Tags, tags.RedHatManaged, strconv.FormatBool(true))
		if !hasRHManaged {
			initialValidSubnets = append(initialValidSubnets, subnet)
		} else {
			excludedSubnets = append(excludedSubnets, awssdk.StringValue(subnet.SubnetId))
		}
	}
	if len(allSubnets) != len(initialValidSubnets) {
		reporter.Warnf("The following subnets were excluded because they belong"+
			" to a VPC that is managed by Red Hat: %s", helper.SliceToSortedString(excludedSubnets))
	}
	return initialValidSubnets, nil
}

func outputClusterAdminDetails(r *rosa.Runtime, isClusterAdmin bool, createAdminPassword string) {
	if isClusterAdmin {
		r.Reporter.Infof("cluster admin user is %s", admin.ClusterAdminUsername)
		r.Reporter.Infof("cluster admin password is %s", createAdminPassword)
	}
}
