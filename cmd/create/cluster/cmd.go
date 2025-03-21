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
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	clustervalidations "github.com/openshift-online/ocm-common/pkg/cluster/validations"
	idputils "github.com/openshift-online/ocm-common/pkg/idp/utils"
	passwordValidator "github.com/openshift-online/ocm-common/pkg/idp/validations"
	diskValidator "github.com/openshift-online/ocm-common/pkg/machinepool/validations"
	kmsArnRegexpValidator "github.com/openshift-online/ocm-common/pkg/resource/validations"
	accountsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"slices"

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
	"github.com/openshift/rosa/pkg/clusterregistryconfig"
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
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	OidcConfigIdFlag                 = "oidc-config-id"
	ClassicOidcConfigFlag            = "classic-oidc-config"
	ExternalAuthProvidersEnabledFlag = "external-auth-providers-enabled"
	workerDiskSizeFlag               = "worker-disk-size"
	// #nosec G101
	Ec2MetadataHttpTokensFlag = "ec2-metadata-http-tokens"

	clusterAutoscalerFlagsPrefix = "autoscaler-"

	MinReplicasSingleAZ = 2
	MinReplicaMultiAZ   = 3

	listInputMessage          = "Format should be a comma-separated list."
	listBillingAccountMessage = "To see the list of billing account options, you can use interactive mode by passing '-i'."

	// nolint:lll
	createVpcForHcpDoc = "https://docs.openshift.com/rosa/rosa_hcp/rosa-hcp-sts-creating-a-cluster-quickly.html#rosa-hcp-creating-vpc"

	duplicateIamRoleArnErrorMsg = "ROSA IAM roles must have unique ARNs " +
		"and should not be shared with other IAM roles within the same cluster. " +
		"Duplicated ARN: %s"

	route53RoleArnFlag                       = "route53-role-arn"
	vpcEndpointRoleArnFlag                   = "vpc-endpoint-role-arn"
	hcpInternalCommunicationHostedZoneIdFlag = "hcp-internal-communication-hosted-zone-id"
	ingressPrivateHostedZoneIdFlag           = "ingress-private-hosted-zone-id"
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
	domainPrefix              string
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
	oidcConfigId                 string
	classicOidcConfig            bool
	externalAuthProvidersEnabled bool

	// Proxy
	enableProxy               bool
	httpProxy                 string
	httpsProxy                string
	noProxySlice              []string
	additionalTrustBundleFile string

	tags []string

	// Hypershift options:
	hostedClusterEnabled        bool
	billingAccount              string
	noCni                       bool
	additionalAllowedPrincipals []string

	// Cluster Admin
	createAdminUser      bool
	clusterAdminPassword string
	clusterAdminUser     string

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

	// HCP Shared VPC
	vpcEndpointRoleArn string
	//
	//route53RoleArn string
	// Route53 Role Arn is the same thing as `sharedVpcRoleArn` for now- deprecation warning will be in place
	// This is the same behavior as create/operatorroles
	//
	hcpInternalCommunicationHostedZoneId string
	//
	//ingressPrivateHostedZoneId string
	// Ingress Private Hosted Zone ID is the same thing as `privateHostedZoneID` for now- deprecation warning
	// will be in place
	//

	// Worker machine pool attributes
	additionalComputeSecurityGroupIds []string

	// Infra machine pool attributes
	additionalInfraSecurityGroupIds []string

	// Control Plane machine pool attributes
	additionalControlPlaneSecurityGroupIds []string

	// Cluster Registry
	allowedRegistries          []string
	blockedRegistries          []string
	insecureRegistries         []string
	allowedRegistriesForImport string
	platformAllowlist          string
	additionalTrustedCa        string
}

var clusterRegistryConfigArgs *clusterregistryconfig.ClusterRegistryConfigArgs
var autoscalerArgs *clusterautoscaler.AutoscalerArgs
var autoscalerValidationArgs *clusterautoscaler.AutoscalerValidationArgs
var userSpecifiedAutoscalerValues []*pflag.Flag

var Cmd = makeCmd()

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Create cluster",
		Long:  "Create cluster.",
		Example: `  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2`,
		Run:  run,
		Args: cobra.NoArgs,
	}
}

func init() {
	initFlags(Cmd)
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	// Basic options
	flags.StringVarP(
		&args.clusterName,
		"name",
		"n",
		"",
		"Name of the cluster.",
	)
	flags.MarkDeprecated("name", "use --cluster-name instead")

	flags.StringVarP(
		&args.clusterName,
		"cluster-name",
		"c",
		"",
		"The unique name of the cluster. The name can be used as the identifier of the cluster."+
			" The maximum length is 54 characters."+
			"Once set, the cluster name cannot be changed",
	)

	flags.StringVar(
		&args.domainPrefix,
		"domain-prefix",
		"",
		"An optional unique domain prefix of the cluster. This will be used when generating a "+
			"sub-domain for your cluster on openshiftapps.com. It must be unique per organization "+
			"and consist of lowercase alphanumeric characters or '-', start with an alphabetic "+
			"character, and end with an alphanumeric character. The maximum length is 15 characters. "+
			"Once set, the cluster domain prefix cannot be changed",
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
		"controlplane-iam-role-arn",
		"",
		"The IAM role ARN that will be attached to control plane instances.",
	)

	flags.StringVar(
		&args.controlPlaneRoleARN,
		"master-iam-role",
		"",
		"The IAM role ARN that will be attached to master instances.",
	)
	flags.MarkDeprecated("master-iam-role", "use --controlplane-iam-role-arn instead")

	flags.StringVar(
		&args.workerRoleARN,
		"worker-iam-role-arn",
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

	flags.BoolVar(
		&args.externalAuthProvidersEnabled,
		ExternalAuthProvidersEnabledFlag,
		false,
		"Enable external authentication configuration for a Hosted Control Plane cluster.",
	)

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
		"Create cluster that uses FIPS Validated / Modules in Process cryptographic libraries. "+
			"This is currently only available without the use of the --hosted-cp flag.",
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

	flags.StringSliceVar(
		&args.additionalAllowedPrincipals,
		"additional-allowed-principals",
		nil,
		"A comma-separated list of additional allowed principal ARNs "+
			"to be added to the Hosted Control Plane's VPC Endpoint Service to enable additional "+
			"VPC Endpoint connection requests to be automatically accepted.",
	)

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
		Ec2MetadataHttpTokensFlag,
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
	clusterRegistryConfigArgs = clusterregistryconfig.AddClusterRegistryConfigFlags(cmd)
	autoscalerArgs = clusterautoscaler.AddClusterAutoscalerFlags(cmd, clusterAutoscalerFlagsPrefix)
	// iterates through all autoscaling flags and stores them in slice to track user input
	flags.VisitAll(func(f *pflag.Flag) {
		if strings.HasPrefix(f.Name, clusterAutoscalerFlagsPrefix) {
			userSpecifiedAutoscalerValues = append(userSpecifiedAutoscalerValues, f)
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

	flags.SetNormalizeFunc(arguments.NormalizeFlags)
	flags.StringVar(
		&args.defaultMachinePoolLabels,
		arguments.NewDefaultMPLabelsFlag,
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
	cmd.RegisterFlagCompletionFunc("network-type", networkTypeCompletion)

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
		"Enable the use of Hosted Control Planes",
	)

	flags.StringVar(&args.machinePoolRootDiskSize,
		workerDiskSizeFlag,
		"",
		"Default worker machine pool root disk size with a **unit suffix** like GiB or TiB, "+
			"e.g. 200GiB.")

	flags.StringVar(
		&args.billingAccount,
		"billing-account",
		"",
		"Account ID used for billing subscriptions purchased through the AWS console for ROSA",
	)

	flags.BoolVar(
		&args.createAdminUser,
		"create-admin-user",
		false,
		`Create cluster admin named "cluster-admin"`,
	)

	flags.BoolVar(
		&args.noCni,
		"no-cni",
		false,
		"Disable CNI creation to let users bring their own CNI.",
	)

	flags.StringVar(
		&args.clusterAdminUser,
		"cluster-admin-user",
		"",
		`Username of Cluster Admin. Username must not contain /, :, or %%`,
	)
	flags.StringVar(
		&args.clusterAdminPassword,
		"cluster-admin-password",
		"",
		`The password must
		- Be at least 14 characters (ASCII-standard) without whitespaces
		- Include uppercase letters, lowercase letters, and numbers or symbols (ASCII-standard characters only)`,
	)
	flags.MarkHidden("cluster-admin-user")

	flags.StringVar(
		&args.AuditLogRoleARN,
		"audit-log-arn",
		"",
		"The ARN of the role that is used to forward audit logs to AWS CloudWatch.",
	)

	flags.StringVar(
		&args.defaultIngressRouteSelectors,
		ingress.DefaultIngressRouteSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	flags.StringVar(
		&args.defaultIngressExcludedNamespaces,
		ingress.DefaultIngressExcludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&args.defaultIngressWildcardPolicy,
		ingress.DefaultIngressWildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ingress.ValidWildcardPolicies, ","), ingress.DefaultWildcardPolicy),
	)

	flags.StringVar(
		&args.defaultIngressNamespaceOwnershipPolicy,
		ingress.DefaultIngressNamespaceOwnershipPolicyFlag,
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
		&args.vpcEndpointRoleArn,
		vpcEndpointRoleArnFlag,
		"",
		"AWS IAM Role ARN with policy attached, associated with the shared VPC."+
			" Grants permissions necessary to communicate with and handle a Hosted Control Plane cross-account VPC.")

	flags.StringVar(
		&args.sharedVPCRoleARN,
		route53RoleArnFlag,
		"",
		"AWS IAM Role Arn with policy attached, associated with shared VPC."+
			" Grants permission necessary to handle route53 operations associated with a cross-account VPC. "+
			"This flag deprecates '--shared-vpc-role-arn'.",
	)

	// Mark old sharedvpc role arn flag as deprecated for future transitioning of the flag name (both are usable for now)
	flags.MarkDeprecated("shared-vpc-role-arn", fmt.Sprintf("'--shared-vpc-role-arn' will be replaced with "+
		"'--%s' in future versions of ROSA.", route53RoleArnFlag))

	flags.StringVar(
		&args.hcpInternalCommunicationHostedZoneId,
		hcpInternalCommunicationHostedZoneIdFlag,
		"",
		"The internal communication Route 53 hosted zone ID to be used for Hosted Control Plane cross-account "+
			"VPC, e.g., 'Z05646003S02O1ENCDCSN'.",
	)

	flags.StringVar(
		&args.privateHostedZoneID,
		ingressPrivateHostedZoneIdFlag,
		"",
		"ID assigned by AWS to private Route 53 hosted zone associated with intended shared VPC, "+
			"e.g., 'Z05646003S02O1ENCDCSN'.",
	)

	// Mark old private hosted zone id flag as deprecated for future transitioning of the flag (both are usable for now)
	flags.MarkDeprecated("private-hosted-zone-id", fmt.Sprintf("'--private-hosted-zone-id' will be "+
		"replaced with '--%s' in future versions of ROSA.", ingressPrivateHostedZoneIdFlag))

	flags.StringVar(
		&args.baseDomain,
		"base-domain",
		"",
		"Base DNS domain name previously reserved and matching the hosted zone name of the private Route 53 hosted zone "+
			"associated with intended shared VPC, e.g., '1vo8.p1.openshiftapps.com'.",
	)

	flags.StringSliceVar(
		&args.additionalComputeSecurityGroupIds,
		securitygroups.ComputeSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the default worker machine pool. "+
			listInputMessage,
	)

	flags.StringSliceVar(
		&args.additionalInfraSecurityGroupIds,
		securitygroups.InfraSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the infra worker nodes. "+
			listInputMessage,
	)

	flags.StringSliceVar(
		&args.additionalControlPlaneSecurityGroupIds,
		securitygroups.ControlPlaneSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the control plane nodes. "+
			listInputMessage,
	)

	interactive.AddModeFlag(cmd)
	interactive.AddFlag(flags)
	output.AddFlag(cmd)
	confirm.AddFlag(flags)
}

func networkTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return ocm.NetworkTypes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Validate mode
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	for _, val := range userSpecifiedAutoscalerValues {
		if val.Changed && !args.autoscalingEnabled {
			r.Reporter.Errorf("Using autoscaling flag '%s', requires flag '--enable-autoscaling'. "+
				"Please try again with flag", val.Name)
			os.Exit(1)
		}
	}

	// validate flags for cluster admin
	isHostedCP := args.hostedClusterEnabled
	if isHostedCP {
		if fedramp.Enabled() {
			r.Reporter.Errorf("Fedramp does not currently support Hosted Control Plane clusters. Please use classic")
			os.Exit(1)
		}
		if cmd.Flag(securitygroups.InfraSecurityGroupFlag).Changed {
			r.Reporter.Errorf("Cannot use '%s' flag with Hosted Control Plane clusters, only '%s' is "+
				"supported", securitygroups.InfraSecurityGroupFlag, securitygroups.ComputeSecurityGroupFlag)
			os.Exit(1)
		}
		if cmd.Flag(securitygroups.ControlPlaneSecurityGroupFlag).Changed {
			r.Reporter.Errorf("Cannot use '%s' flag with Hosted Control Plane clusters, only '%s' is "+
				"supported", securitygroups.ControlPlaneSecurityGroupFlag, securitygroups.ComputeSecurityGroupFlag)
			os.Exit(1)
		}
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
		r.Reporter.Errorf("Cluster name must consist"+
			" of no more than %d lowercase alphanumeric characters or '-', "+
			"start with a letter, and end with an alphanumeric character.", ocm.MaxClusterNameLength)
		os.Exit(1)
	}

	// Get cluster domain prefix
	domainPrefix := strings.Trim(args.domainPrefix, " \t")

	if interactive.Enabled() {
		domainPrefix, err = interactive.GetString(interactive.Input{
			Question: "Domain prefix",
			Help:     cmd.Flags().Lookup("domain-prefix").Usage,
			Default:  domainPrefix,
			Required: false,
			Validators: []interactive.Validator{
				ocm.ClusterDomainPrefixValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid domain prefix: %s", err)
			os.Exit(1)
		}
	}

	// Trim domain prefix to remove any leading/trailing invisible characters
	domainPrefix = strings.Trim(domainPrefix, " \t")

	if domainPrefix != "" && !ocm.IsValidClusterDomainPrefix(domainPrefix) {
		r.Reporter.Errorf("Domain prefix must consist"+
			" of no more than %d lowercase alphanumeric characters or '-', "+
			"start with a letter, and end with an alphanumeric character.", ocm.MaxClusterDomainPrefixLength)
		os.Exit(1)
	}

	if clusterHasLongNameWithoutDomainPrefix(clusterName, domainPrefix) {
		prompt := fmt.Sprintf("Your cluster has a name longer than %d characters, it will contain"+
			" an autogenerated domain prefix as a sub-domain for your cluster on "+
			"openshiftapps.com when provisioned. Do you want to proceed?", ocm.MaxClusterDomainPrefixLength)
		if !confirm.ConfirmRaw(prompt) {
			r.Reporter.Warnf("You opted out from creating a cluster with an autogenerated " +
				"sub-domain for your cluster on openshiftapps.com. To customise the sub-domain" +
				", use the '--domain-prefix' flag")
			os.Exit(0)
		}
	}

	if interactive.Enabled() && !fedramp.Enabled() {
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

	if isHostedCP && r.Reporter.IsTerminal() {
		techPreviewMsg, err := r.OCMClient.GetTechnologyPreviewMessage(ocm.HcpProduct, time.Now())
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if techPreviewMsg != "" {
			r.Reporter.Infof(techPreviewMsg)
		}
	}

	createAdminUser := args.createAdminUser
	clusterAdminUser := admin.ClusterAdminUsername
	clusterAdminPassword := strings.Trim(args.clusterAdminPassword, " \t")
	// isClusterAdmin is a flag indicating if user wishes to create cluster admin
	isClusterAdmin := false

	if createAdminUser || clusterAdminPassword != "" {
		isClusterAdmin = true
		// user supplies create-admin-user flag without cluster-admin-password will generate random password
		if clusterAdminPassword == "" {
			r.Reporter.Debugf(admin.GeneratingRandomPasswordString)
			clusterAdminPassword, err = idputils.GenerateRandomPassword()
			if err != nil {
				r.Reporter.Errorf("Failed to generate a random password")
				os.Exit(1)
			}
		}
		// validates both user inputted custom password and randomly generated password
		err = passwordValidator.PasswordValidator(clusterAdminPassword)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if clusterAdminUser != "" {
			err = idp.UsernameValidator(clusterAdminUser)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		} else {
			clusterAdminUser = admin.ClusterAdminUsername
		}
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
			//clusterAdminUser = idp.GetIdpUserNameFromPrompt(cmd, r, "cluster-admin-user", clusterAdminUser, true)
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
				clusterAdminPassword, err = idputils.GenerateRandomPassword()
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
	outputClusterAdminDetails(r, isClusterAdmin, clusterAdminUser, clusterAdminPassword)

	if isHostedCP && cmd.Flags().Changed(arguments.NewDefaultMPLabelsFlag) {
		r.Reporter.Errorf("Setting the worker machine pool labels is not supported for hosted clusters")
		os.Exit(1)
	}

	// Billing Account
	billingAccount := args.billingAccount
	if isHostedCP {
		isHcpBillingTechPreview, err := r.OCMClient.IsTechnologyPreview(ocm.HcpBillingAccount, time.Now())
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if !isHcpBillingTechPreview {

			if billingAccount != "" && !ocm.IsValidAWSAccount(billingAccount) {
				r.Reporter.Errorf("Provided billing account number %s is not valid. "+
					"Rerun the command with a valid billing account number. %s",
					billingAccount, listBillingAccountMessage)
				os.Exit(1)
			}

			cloudAccounts, err := r.OCMClient.GetBillingAccounts()
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}

			billingAccounts := ocm.GenerateBillingAccountsList(cloudAccounts)

			if billingAccount == "" && !interactive.Enabled() {
				// if a billing account is not provided we will try to use the infrastructure account as default
				billingAccount, err = provideBillingAccount(billingAccounts, awsCreator.AccountID, r)
				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}

			if interactive.Enabled() {
				if len(billingAccounts) > 0 {
					billingAccount, err = interactive.GetOption(interactive.Input{
						Question: "Billing Account",
						Help:     cmd.Flags().Lookup("billing-account").Usage,
						Default:  billingAccount,
						Required: true,
						Options:  billingAccounts,
					})

					if err != nil {
						r.Reporter.Errorf("Expected a valid billing account: '%s'", err)
						os.Exit(1)
					}

					billingAccount = aws.ParseOption(billingAccount)
				}

				err := ocm.ValidateBillingAccount(billingAccount)
				if err != nil {
					r.Reporter.Errorf("%v", err)
					os.Exit(1)
				}

				// Get contract info
				contracts, isContractEnabled := ocm.GetBillingAccountContracts(cloudAccounts, billingAccount)

				if billingAccount != awsCreator.AccountID {
					r.Reporter.Infof(
						"The AWS billing account you selected is different from your AWS infrastructure account. " +
							"The AWS billing account will be charged for subscription usage. " +
							"The AWS infrastructure account contains the ROSA infrastructure.",
					)
				} else {
					r.Reporter.Infof("Using '%s' as billing account.",
						billingAccount)
				}

				if isContractEnabled && len(contracts) > 0 {
					//currently, an AWS account will have only one ROSA HCP active contract at a time
					contractDisplay := GenerateContractDisplay(contracts[0])
					r.Reporter.Infof(contractDisplay)
				}
			}
		}
	}

	if !isHostedCP && billingAccount != "" {
		r.Reporter.Errorf("Billing accounts are only supported for Hosted Control Plane clusters")
		os.Exit(1)
	}

	externalAuthProvidersEnabled := args.externalAuthProvidersEnabled
	if externalAuthProvidersEnabled {
		if !isHostedCP {
			r.Reporter.Errorf(
				"External authentication configuration is only supported for a Hosted Control Plane cluster.",
			)
			os.Exit(1)
		}
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
	defaultVersion, versionList, err := versions.GetVersionList(r, channelGroup, isSTS, isHostedCP, isHostedCP, true)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if version == "" {
		version = defaultVersion
	}
	if interactive.Enabled() {
		version, err = interactive.GetOption(interactive.Input{
			Question: "OpenShift version",
			Help:     cmd.Flags().Lookup("version").Usage,
			Options:  versionList,
			Default:  version,
			Required: true,
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
	if httpTokens == "" {
		httpTokens = string(v1.Ec2MetadataHttpTokensOptional)
	}
	if interactive.Enabled() {
		httpTokens, err = interactive.GetOption(interactive.Input{
			Question: "Configure the use of IMDSv2 for ec2 instances",
			Options:  []string{string(v1.Ec2MetadataHttpTokensOptional), string(v1.Ec2MetadataHttpTokensRequired)},
			Help:     cmd.Flags().Lookup(Ec2MetadataHttpTokensFlag).Usage,
			Required: true,
			Default:  httpTokens,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid http tokens value : %v", err)
			os.Exit(1)
		}
	}
	if err = ocm.ValidateHttpTokensValue(httpTokens); err != nil {
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
		isValidMode := arguments.IsValidMode(interactive.Modes, mode)
		if !isValidMode {
			r.Reporter.Errorf("Invalid --mode '%s'. Allowed values are %s", mode, interactive.Modes)
			os.Exit(1)
		}
	}

	if args.watch && isSTS && mode == interactive.ModeAuto && !confirm.Yes() {
		r.Reporter.Errorf("Cannot watch for STS cluster installation logs in mode 'auto' " +
			"without also supplying '--yes' option." +
			"To watch your cluster installation logs, run 'rosa logs install' instead after the cluster has began creating.")
		os.Exit(1)
	}

	if args.watch && isSTS && mode == interactive.ModeManual {
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
		var roleARNs []string
		if isHostedCP {
			roleARNs, err = awsClient.FindRoleARNsHostedCp(aws.InstallerAccountRole, minor)
		} else {
			roleARNs, err = awsClient.FindRoleARNsClassic(aws.InstallerAccountRole, minor)
		}
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
				defaultRoleMatch := fmt.Sprintf("%s-%s-Role", aws.DefaultPrefix, role.Name)
				if isHostedCP {
					defaultRoleMatch = fmt.Sprintf(
						"%s-%s-%s-Role",
						aws.DefaultPrefix,
						aws.HCPSuffixPattern,
						role.Name,
					)
				}
				if roleName == defaultRoleMatch {
					defaultRoleARN = rARN
					break
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
				createAccountRolesCommand = createAccountRolesCommand + " " + hostedCPFlag
			}
			r.Reporter.Warnf(fmt.Sprintf("No compatible account roles with version '%s' found. "+
				"You will need to manually set them in the next steps or run '%s' to create them first.",
				minor, createAccountRolesCommand))
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
				if isHostedCP {
					roleARNs, err = awsClient.FindRoleARNsHostedCp(roleType, minor)
				} else {
					roleARNs, err = awsClient.FindRoleARNsClassic(roleType, minor)
				}
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
				r.Reporter.Debugf(
					"Using '%s' as the role prefix to retrieve the expected resource ID for role type '%s'",
					rolePrefix,
					roleType,
				)

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
						createAccountRolesCommand = createAccountRolesCommand + " " + hostedCPFlag
					}
					r.Reporter.Warnf(fmt.Sprintf("No compatible '%s' account roles with version '%s' found. "+
						"You will need to manually set them in the next steps or run '%s' to create them first.",
						role.Name, minor, createAccountRolesCommand))
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
	operatorRoles := []string{}
	expectedOperatorRolePath, _ := aws.GetPathFromARN(roleARN)
	operatorIAMRoles := args.operatorIAMRoles
	computedOperatorIamRoleList := []ocm.OperatorIAMRole{}
	if isSTS {
		if operatorRolesPrefix == "" {
			operatorRolesPrefix = roles.GeOperatorRolePrefixFromClusterName(clusterName)
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

		credRequests, err := r.OCMClient.GetAllCredRequests()
		if err != nil {
			r.Reporter.Errorf("Error getting operator credential request from OCM %v", err)
			os.Exit(1)
		}
		operatorRoles, err = r.AWSClient.GetOperatorRolesFromAccountByPrefix(operatorRolesPrefix, credRequests)
		if err != nil {
			r.Reporter.Errorf("There was a problem retrieving the Operator Roles from AWS: %v", err)
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
					oidcConfig.IssuerUrl(), ocm.GetVersionMinor(version), expectedOperatorRolePath, managedPolicies, true)
				if err != nil {
					r.Reporter.Errorf("%v", err)
					os.Exit(1)
				}
			}
		}
		err = validateUniqueIamRoleArnsForStsCluster(roleARNs, computedOperatorIamRoleList)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
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
	subnetIDs := helper.FilterEmptyStrings(args.subnetIDs)
	subnetsProvided := len(subnetIDs) > 0
	r.Reporter.Debugf("Received the following subnetIDs: %v", subnetIDs)
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
	var subnets []ec2types.Subnet
	mapSubnetIDToSubnet := make(map[string]aws.Subnet)
	if useExistingVPC || subnetsProvided {
		initialSubnets, err := getInitialValidSubnets(awsClient, subnetIDs, r.Reporter)
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
		var filterError error
		subnets, filterError = filterCidrRangeSubnets(initialSubnets, machineNetwork, serviceNetwork, r)
		if filterError != nil {
			r.Reporter.Errorf("%s", filterError)
			os.Exit(1)
		}

		var excludedSubnetIdsDueToBeingPublic []string
		if privateLink {
			subnets, excludedSubnetIdsDueToBeingPublic = filterPrivateSubnets(subnets, r)
		}

		if len(subnets) == 0 {
			r.Reporter.Warnf("No subnets found in current region that are valid for the chosen CIDR ranges")
			if isHostedCP {
				r.Reporter.Errorf(
					"All Hosted Control Plane clusters need a pre-configured VPC. Please check: %s",
					createVpcForHcpDoc,
				)
				os.Exit(1)
			}
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
				// Check if subnet is in the excluded list of public subnets
				if slices.Contains(excludedSubnetIdsDueToBeingPublic, subnetArg) {
					r.Reporter.Errorf("The command cannot be executed because %s is public and the cluster is set as private",
						subnetArg)
					os.Exit(1)
				}

				// Check if the provided subnet exists in the filtered list
				if !slices.ContainsFunc(subnets, func(subnet ec2types.Subnet) bool {
					return awssdk.ToString(subnet.SubnetId) == subnetArg
				}) {
					r.Reporter.Errorf("Could not find the following subnet provided in region '%s': %s",
						r.AWSClient.GetRegion(), subnetArg)
					os.Exit(1)
				}
			}
		}

		mapVpcToSubnet := map[string][]ec2types.Subnet{}

		for _, subnet := range subnets {
			mapVpcToSubnet[*subnet.VpcId] = append(mapVpcToSubnet[*subnet.VpcId], subnet)
			subnetID := awssdk.ToString(subnet.SubnetId)
			availabilityZone := awssdk.ToString(subnet.AvailabilityZone)
			mapSubnetIDToSubnet[subnetID] = aws.Subnet{
				AvailabilityZone: availabilityZone,
				OwnerID:          awssdk.ToString(subnet.OwnerId),
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
			len(options) > 0 && (!multiAZ || len(mapAZCreated) >= 3 || isHostedCP) {
			subnetIDs, err = interactive.GetMultipleOptions(interactive.Input{
				Question: "Subnet IDs",
				Help:     cmd.Flags().Lookup("subnet-ids").Usage,
				Required: false,
				Options:  options,
				Default:  defaultOptions,
				Validators: []interactive.Validator{
					interactive.SubnetsValidator(awsClient, multiAZ, privateLink, isHostedCP),
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

		// Validate subnets in the case the user has provided them using the `args.subnetIDs`
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

	// validate flags for hcp shared vpc
	isHcpSharedVpc := false
	route53RoleArn := sharedVPCRoleARN // TODO: Change when fully deprecating old flag name
	vpcEndpointRoleArn := strings.Trim(args.vpcEndpointRoleArn, " \t")
	hcpInternalCommunicationHostedZoneId := strings.Trim(args.hcpInternalCommunicationHostedZoneId, " \t")
	ingressPrivateHostedZoneId := privateHostedZoneID // TODO: Change when fully deprecating old flag name

	anyHcpSharedVpcFlagsUsed := route53RoleArn != "" || vpcEndpointRoleArn != "" ||
		hcpInternalCommunicationHostedZoneId != "" || ingressPrivateHostedZoneId != ""

	if len(subnetIDs) == 0 && (isSharedVPC || isHcpSharedVpc) {
		r.Reporter.Errorf("Installing a cluster into a shared VPC is only supported for BYO VPC clusters")
		os.Exit(1)
	}

	if isSubnetBelongToSharedVpc(r, awsCreator.AccountID, subnetIDs, mapSubnetIDToSubnet) {

		if clusterHasLongNameWithoutDomainPrefix(clusterName, domainPrefix) {
			r.Reporter.Errorf("Installing a cluster into shared VPC is only supported for cluster "+
				"which has a name no longer than %d characters or with a cluster domain prefix",
				ocm.MaxClusterDomainPrefixLength)
			os.Exit(1)
		}

		isSharedVPC = true

		useInteractive := false

		if isHostedCP {
			isHcpSharedVpc = true
			useInteractive = route53RoleArn == "" || vpcEndpointRoleArn == "" || ingressPrivateHostedZoneId == "" ||
				hcpInternalCommunicationHostedZoneId == ""
		}

		useInteractive = useInteractive || privateHostedZoneID == "" || sharedVPCRoleARN == "" || baseDomain == ""

		if useInteractive {
			if !interactive.Enabled() {
				interactive.Enable()
			}

			privateHostedZoneID, err = getPrivateHostedZoneID(cmd, privateHostedZoneID)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}

			// TODO: We can remove this and replace the above once we deprecate the old flags
			ingressPrivateHostedZoneId = privateHostedZoneID

			if isHostedCP {
				hcpInternalCommunicationHostedZoneId, err = getHcpInternalCommunicationHostedZoneId(cmd,
					hcpInternalCommunicationHostedZoneId)

				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}

			sharedVPCRoleARN, err = getSharedVpcRoleArn(cmd, sharedVPCRoleARN)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}

			// TODO: We can remove this and replace the above once we deprecate the old flags
			route53RoleArn = sharedVPCRoleARN

			if isHostedCP {
				vpcEndpointRoleArn, err = getVpcEndpointRoleArn(cmd, vpcEndpointRoleArn)
				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}

			baseDomain, err = getBaseDomain(r, cmd, baseDomain, isHostedCP)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
	}

	if anyHcpSharedVpcFlagsUsed {
		if isHostedCP {
			isHcpSharedVpc = true
			err = validateHcpSharedVpcArgs(route53RoleArn, vpcEndpointRoleArn, ingressPrivateHostedZoneId,
				hcpInternalCommunicationHostedZoneId)
			if err != nil {
				r.Reporter.Errorf("Error when validating flags: %v", err)
				os.Exit(1)
			}
		} else {
			if vpcEndpointRoleArn != "" {
				r.Reporter.Errorf(hcpSharedVpcFlagOnlyErrorMsg,
					vpcEndpointRoleArnFlag)
				os.Exit(1)
			} else if hcpInternalCommunicationHostedZoneId != "" {
				r.Reporter.Errorf(hcpSharedVpcFlagOnlyErrorMsg,
					hcpInternalCommunicationHostedZoneIdFlag)
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
				interactive.RegExp(kmsArnRegexpValidator.KmsArnRE.String()),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for kms-key-arn: %s", err)
			os.Exit(1)
		}
	}

	err = kmsArnRegexpValidator.ValidateKMSKeyARN(&kmsKeyARN)
	if err != nil {
		r.Reporter.Errorf("Expected a valid value for kms-key-arn: %s", err)
		os.Exit(1)
	}

	// Compute node instance type:
	computeMachineType := args.computeMachineType
	computeMachineTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(region, availabilityZones, roleARN,
		awsClient, externalID)
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

	replicaSizeValidation := &machinepool.ReplicaSizeValidation{
		MinReplicas:         minReplicas,
		ClusterVersion:      version,
		PrivateSubnetsCount: privateSubnetsCount,
		IsHostedCp:          isHostedCP,
		MultiAz:             multiAZ,
	}
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
					replicaSizeValidation.MinReplicaValidatorOnClusterCreate(),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		err = replicaSizeValidation.MinReplicaValidatorOnClusterCreate()(minReplicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		replicaSizeValidation.MinReplicas = minReplicas

		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					replicaSizeValidation.MaxReplicaValidatorOnClusterCreate(),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
		err = replicaSizeValidation.MaxReplicaValidatorOnClusterCreate()(maxReplicas)
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

			autoscalerValidationArgs = &clusterautoscaler.AutoscalerValidationArgs{
				ClusterVersion: version,
				MultiAz:        multiAZ,
				IsHostedCp:     false,
			}

			clusterAutoscaler, err = clusterautoscaler.GetAutoscalerOptions(
				cmd.Flags(), clusterAutoscalerFlagsPrefix, true, autoscalerArgs, autoscalerValidationArgs)
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
					replicaSizeValidation.MinReplicaValidatorOnClusterCreate(),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of compute nodes: %s", err)
				os.Exit(1)
			}
		}
		err = replicaSizeValidation.MinReplicaValidatorOnClusterCreate()(computeNodes)
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
			Help:     cmd.Flags().Lookup(arguments.NewDefaultMPLabelsFlag).Usage,
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
	getSecurityGroups(r, cmd, isVersionCompatibleComputeSgIds,
		securitygroups.ComputeKind, useExistingVPC, subnets,
		subnetIDs, &additionalComputeSecurityGroupIds)

	additionalInfraSecurityGroupIds := args.additionalInfraSecurityGroupIds
	additionalControlPlaneSecurityGroupIds := args.additionalControlPlaneSecurityGroupIds
	if !isHostedCP {
		getSecurityGroups(r, cmd, isVersionCompatibleComputeSgIds,
			securitygroups.InfraKind, useExistingVPC, subnets,
			subnetIDs, &additionalInfraSecurityGroupIds)

		getSecurityGroups(r, cmd, isVersionCompatibleComputeSgIds,
			securitygroups.ControlPlaneKind, useExistingVPC, subnets,
			subnetIDs, &additionalControlPlaneSecurityGroupIds)
	}

	// Validate all remaining flags:
	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	// Network Type:
	if err := validateNetworkType(args.networkType); err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if cmd.Flags().Changed("network-type") && interactive.Enabled() {
		args.networkType, err = interactive.GetOption(interactive.Input{
			Question: "Network Type",
			Help:     cmd.Flags().Lookup("network-type").Usage,
			Options:  ocm.NetworkTypes,
			Default:  args.networkType,
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

	machinePoolRootDisk, err := getMachinePoolRootDisk(r, cmd, version,
		isHostedCP, defaultMachinePoolRootDiskSize)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}

	// No CNI
	if cmd.Flags().Changed("no-cni") && !isHostedCP {
		r.Reporter.Errorf("Disabling CNI is supported only for Hosted Control Planes")
		os.Exit(1)
	}
	if cmd.Flags().Changed("no-cni") && cmd.Flags().Changed("network-type") {
		r.Reporter.Errorf("--no-cni and --network-type are mutually exclusive parameters")
		os.Exit(1)
	}
	noCni := args.noCni
	if cmd.Flags().Changed("no-cni") && interactive.Enabled() {
		noCni, err = interactive.GetBool(interactive.Input{
			Question: "Disable CNI",
			Help:     cmd.Flags().Lookup("no-cni").Usage,
			Default:  noCni,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for no CNI: %s", err)
			os.Exit(1)
		}
	}

	if cmd.Flags().Changed("fips") && isHostedCP {
		r.Reporter.Errorf("FIPS support not available for Hosted Control Plane clusters")
		os.Exit(1)
	}
	fips := args.fips || fedramp.Enabled()
	if interactive.Enabled() && !fedramp.Enabled() && !isHostedCP {
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
				interactive.RegExp(kmsArnRegexpValidator.KmsArnRE.String()),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for etcd-encryption-kms-arn: %s", err)
			os.Exit(1)
		}
	}

	err = kmsArnRegexpValidator.ValidateKMSKeyARN(&etcdEncryptionKmsARN)
	if err != nil {
		r.Reporter.Errorf(
			"Expected a valid value for etcd-encryption-kms-arn matching %s",
			kmsArnRegexpValidator.KmsArnRE,
		)
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

	// Additional Allowed Principals
	if cmd.Flags().Changed("additional-allowed-principals") && !isHostedCP {
		r.Reporter.Errorf("Additional Allowed Principals is supported only for Hosted Control Planes")
		os.Exit(1)
	}
	additionalAllowedPrincipals := args.additionalAllowedPrincipals
	if isHostedCP && interactive.Enabled() {
		aapInputs, err := interactive.GetString(interactive.Input{
			Question: "Additional Allowed Principal ARNs",
			Help:     cmd.Flags().Lookup("additional-allowed-principals").Usage,
			Default:  strings.Join(additionalAllowedPrincipals, ","),
			Required: isHcpSharedVpc,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for Additional Allowed Principal ARNs: %s", err)
			os.Exit(1)
		}
		additionalAllowedPrincipals = helper.HandleEmptyStringOnSlice(strings.Split(aapInputs, ","))
	}
	if len(additionalAllowedPrincipals) > 0 {
		if err := roles.ValidateAdditionalAllowedPrincipals(additionalAllowedPrincipals); err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
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
	if ingress.IsDefaultIngressSetViaCLI(cmd.Flags()) {
		if isHostedCP {
			r.Reporter.Errorf("Updating default ingress settings is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		if !isVersionCompatibleManagedIngressV2 {
			formattedVersion, err := versions.FormatMajorMinorPatch(ocm.MinVersionForManagedIngressV2)
			if err != nil {
				r.Reporter.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
				os.Exit(1)
			}
			r.Reporter.Errorf(
				"Updating default ingress settings is not supported for versions prior to '%s'",
				formattedVersion,
			)
			os.Exit(1)
		}
	}
	routeSelector := ""
	routeSelectors := map[string]string{}
	excludedNamespaces := ""
	sliceExcludedNamespaces := []string{}
	wildcardPolicy := ""
	namespaceOwnershipPolicy := ""
	if isVersionCompatibleManagedIngressV2 {
		shouldAskCustomIngress := false
		if interactive.Enabled() && !confirm.Yes() && !isHostedCP {
			shouldAskCustomIngress = confirm.Prompt(false, "Customize the default Ingress Controller?")
		}
		if cmd.Flags().Changed(ingress.DefaultIngressRouteSelectorFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating route selectors is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			routeSelector = args.defaultIngressRouteSelectors
		} else if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
			routeSelectorArg, err := interactive.GetString(interactive.Input{
				Question: "Router Ingress Sharding: Route Selector (e.g. 'route=external')",
				Help:     cmd.Flags().Lookup(ingress.DefaultIngressRouteSelectorFlag).Usage,
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

		if cmd.Flags().Changed(ingress.DefaultIngressExcludedNamespacesFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating excluded namespace is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			excludedNamespaces = args.defaultIngressExcludedNamespaces
		} else if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
			excludedNamespacesArg, err := interactive.GetString(interactive.Input{
				Question: "Router Ingress Sharding: Namespace exclusion",
				Help:     cmd.Flags().Lookup(ingress.DefaultIngressExcludedNamespacesFlag).Usage,
				Default:  args.defaultIngressExcludedNamespaces,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
				os.Exit(1)
			}
			excludedNamespaces = excludedNamespacesArg
		}
		sliceExcludedNamespaces = ingress.GetExcludedNamespaces(excludedNamespaces)

		if cmd.Flags().Changed(ingress.DefaultIngressWildcardPolicyFlag) {
			if isHostedCP {
				r.Reporter.Errorf("Updating Wildcard Policy is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			wildcardPolicy = args.defaultIngressWildcardPolicy
		} else {
			if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
				defaultIngressWildcardSelection := string(v1.WildcardPolicyWildcardsDisallowed)
				if args.defaultIngressWildcardPolicy != "" {
					defaultIngressWildcardSelection = args.defaultIngressWildcardPolicy
				}
				wildcardPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Route Admission: Wildcard Policy",
					Options:  ingress.ValidWildcardPolicies,
					Help:     cmd.Flags().Lookup(ingress.DefaultIngressWildcardPolicyFlag).Usage,
					Default:  defaultIngressWildcardSelection,
					Required: true,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid Wildcard Policy: %s", err)
					os.Exit(1)
				}
				wildcardPolicy = wildcardPolicyArg
			}
		}

		if cmd.Flags().Changed(ingress.DefaultIngressNamespaceOwnershipPolicyFlag) {
			if isHostedCP {
				r.Reporter.Errorf(
					"Updating Namespace Ownership Policy is not supported for Hosted Control Plane clusters",
				)
				os.Exit(1)
			}
			namespaceOwnershipPolicy = args.defaultIngressNamespaceOwnershipPolicy
		} else {
			if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
				defaultIngressNamespaceOwnershipSelection := string(v1.NamespaceOwnershipPolicyStrict)
				if args.defaultIngressNamespaceOwnershipPolicy != "" {
					defaultIngressNamespaceOwnershipSelection = args.defaultIngressNamespaceOwnershipPolicy
				}
				namespaceOwnershipPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Route Admission: Namespace Ownership Policy",
					Options:  ingress.ValidNamespaceOwnershipPolicies,
					Help:     cmd.Flags().Lookup(ingress.DefaultIngressNamespaceOwnershipPolicyFlag).Usage,
					Default:  defaultIngressNamespaceOwnershipSelection,
					Required: true,
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
		Name:                         clusterName,
		DomainPrefix:                 domainPrefix,
		Region:                       region,
		MultiAZ:                      multiAZ,
		Version:                      version,
		ChannelGroup:                 channelGroup,
		Flavour:                      args.flavour,
		FIPS:                         fips,
		EtcdEncryption:               etcdEncryption,
		EtcdEncryptionKMSArn:         etcdEncryptionKmsARN,
		EnableProxy:                  enableProxy,
		AdditionalTrustBundle:        additionalTrustBundle,
		Expiration:                   expiration,
		ComputeMachineType:           computeMachineType,
		ComputeNodes:                 computeNodes,
		Autoscaling:                  autoscaling,
		MinReplicas:                  minReplicas,
		MaxReplicas:                  maxReplicas,
		ComputeLabels:                labelMap,
		NetworkType:                  args.networkType,
		MachineCIDR:                  machineCIDR,
		ServiceCIDR:                  serviceCIDR,
		PodCIDR:                      podCIDR,
		HostPrefix:                   hostPrefix,
		Private:                      &private,
		DryRun:                       &args.dryRun,
		DisableSCPChecks:             &args.disableSCPChecks,
		AvailabilityZones:            availabilityZones,
		SubnetIds:                    subnetIDs,
		PrivateLink:                  &privateLink,
		AWSCreator:                   awsCreator,
		IsSTS:                        isSTS,
		RoleARN:                      roleARN,
		ExternalID:                   externalID,
		ExternalAuthProvidersEnabled: externalAuthProvidersEnabled,
		SupportRoleARN:               supportRoleARN,
		OperatorIAMRoles:             computedOperatorIamRoleList,
		ControlPlaneRoleARN:          controlPlaneRoleARN,
		WorkerRoleARN:                workerRoleARN,
		Mode:                         mode,
		Tags:                         tagsList,
		KMSKeyArn:                    kmsKeyARN,
		DisableWorkloadMonitoring:    &disableWorkloadMonitoring,
		Hypershift: ocm.Hypershift{
			Enabled: isHostedCP,
		},
		BillingAccount:  billingAccount,
		NoCni:           noCni,
		AuditLogRoleARN: &auditLogRoleARN,
		DefaultIngress: ocm.DefaultIngressSpec{
			RouteSelectors:           routeSelectors,
			ExcludedNamespaces:       sliceExcludedNamespaces,
			WildcardPolicy:           wildcardPolicy,
			NamespaceOwnershipPolicy: namespaceOwnershipPolicy,
		},
		MachinePoolRootDisk:                    machinePoolRootDisk,
		AdditionalComputeSecurityGroupIds:      additionalComputeSecurityGroupIds,
		AdditionalInfraSecurityGroupIds:        additionalInfraSecurityGroupIds,
		AdditionalControlPlaneSecurityGroupIds: additionalControlPlaneSecurityGroupIds,
		AdditionalAllowedPrincipals:            additionalAllowedPrincipals,
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
		clusterConfig.ClusterAdminUser = clusterAdminUser
		clusterConfig.ClusterAdminPassword = clusterAdminPassword
	}
	if isSharedVPC {
		clusterConfig.PrivateHostedZoneID = privateHostedZoneID
		clusterConfig.SharedVPCRoleArn = sharedVPCRoleARN
		clusterConfig.BaseDomain = baseDomain
	}
	if isHcpSharedVpc {
		clusterConfig.PrivateHostedZoneID = privateHostedZoneID
		clusterConfig.SharedVPCRoleArn = sharedVPCRoleARN
		clusterConfig.InternalCommunicationHostedZoneId = hcpInternalCommunicationHostedZoneId
		clusterConfig.VpcEndpointRoleArn = vpcEndpointRoleArn
	}
	if clusterAutoscaler != nil {
		autoscalerConfig, err := clusterautoscaler.CreateAutoscalerConfig(clusterAutoscaler)
		if err != nil {
			r.Reporter.Errorf("Failed creating autoscaler configuration: %s", err)
			os.Exit(1)
		}

		clusterConfig.AutoscalerConfig = autoscalerConfig
	}

	clusterRegistryConfigArgs, err = clusterregistryconfig.GetClusterRegistryConfigOptions(
		cmd.Flags(), clusterRegistryConfigArgs, isHostedCP, nil)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if clusterRegistryConfigArgs != nil {
		allowedRegistries, blockedRegistries, insecureRegistries,
			additionalTrustedCa, allowedRegistriesForImport,
			platformAllowlist := clusterregistryconfig.GetClusterRegistryConfigArgs(
			clusterRegistryConfigArgs)
		clusterConfig.AllowedRegistries = allowedRegistries
		clusterConfig.BlockedRegistries = blockedRegistries
		clusterConfig.InsecureRegistries = insecureRegistries
		clusterConfig.PlatformAllowlist = platformAllowlist

		if additionalTrustedCa != "" {
			ca, err := clusterregistryconfig.BuildAdditionalTrustedCAFromInputFile(additionalTrustedCa)
			if err != nil {
				r.Reporter.Errorf("Failed to build the additional trusted ca from file %s, got error: %s", additionalTrustedCa, err)
				os.Exit(1)
			}
			clusterConfig.AdditionalTrustedCa = ca
			clusterConfig.AdditionalTrustedCaFile = additionalTrustedCa
		}
		clusterConfig.AllowedRegistriesForImport = allowedRegistriesForImport
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

	clusterConfig, err = clusterConfigFor(r.Reporter, clusterConfig, awsCreator, awsClient)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating cluster '%s'", clusterName)
		if interactive.Enabled() {
			command := buildCommand(clusterConfig, operatorRolesPrefix, expectedOperatorRolePath,
				isAvailabilityZonesSet || selectAvailabilityZones, labels, args.properties)
			r.Reporter.Infof("To create this cluster again in the future, you can run:\n   %s", command)
		}
		r.Reporter.Infof("To view a list of clusters and their status, run 'rosa list clusters'")
	}

	if !clusterConfig.IsSTS {
		if err := r.OCMClient.EnsureNoPendingClusters(awsCreator); err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}
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

	arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
	clusterdescribe.Cmd.Run(clusterdescribe.Cmd, []string{cluster.ID()})

	if isSTS {
		if mode != "" {
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create operator roles.")
			}
			// Set flags for HCP shared VPC if needed
			if vpcEndpointRoleArn != "" && route53RoleArn != "" && isHostedCP {
				operatorroles.Cmd.Flags().Set(route53RoleArnFlag, route53RoleArn)
				operatorroles.Cmd.Flags().Set(vpcEndpointRoleArnFlag, vpcEndpointRoleArn)
				operatorroles.Cmd.Flags().Set(operatorroles.HostedCpFlag, strconv.FormatBool(isHostedCP))
				operatorroles.Cmd.Flags().Set(operatorroles.OidcConfigIdFlag, oidcConfig.ID())
			}
			operatorroles.Cmd.Run(operatorroles.Cmd, []string{clusterName, mode, permissionsBoundary})
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create OIDC Provider.")
			}
			if oidcConfig != nil {
				oidcprovider.Cmd.Flags().Set(oidcprovider.OidcConfigIdFlag, oidcConfig.ID())
			}
			oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{clusterName, mode, ""})
		} else {
			output := ""
			if len(operatorRoles) == 0 {
				rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", clusterName)
				if permissionsBoundary != "" {
					rolesCMD = fmt.Sprintf("%s --permissions-boundary %s", rolesCMD, permissionsBoundary)
				}
				// HCP Shared VPC
				if route53RoleArn != "" {
					rolesCMD = fmt.Sprintf("%s --%s %s --hosted-cp", rolesCMD, route53RoleArnFlag, route53RoleArn)
				}
				if vpcEndpointRoleArn != "" {
					rolesCMD = fmt.Sprintf("%s --%s %s", rolesCMD, vpcEndpointRoleArnFlag, vpcEndpointRoleArn)
				}
				output = fmt.Sprintf("%s\t%s\n", output, rolesCMD)
			}
			oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()
			oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(oidcEndpointURL,
				r.Creator.Partition, r.Creator.AccountID)
			if err != nil {
				if strings.Contains(err.Error(), "AccessDenied") {
					r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
				} else {
					r.Reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
					os.Exit(1)
				}
			}
			if !oidcProviderExists {
				oidcCMD := "rosa create oidc-provider"
				oidcCMD = fmt.Sprintf("%s --cluster %s", oidcCMD, clusterName)
				output = fmt.Sprintf("%s\t%s\n", output, oidcCMD)
			}
			if output != "" {
				output = fmt.Sprintf("Run the following commands to continue the cluster creation:\n\n%s",
					output)
				r.Reporter.Infof(output)
			}
		}
	}

	if args.watch {
		installLogs.Cmd.Run(installLogs.Cmd, []string{clusterName})
		arguments.DisableRegionDeprecationWarning = false // no longer disable deprecation warning
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

// clusterConfigFor builds the cluster spec for the OCM API from our command-line options.
// TODO: eventually, this method signature should be func(args) ocm.Spec.
func clusterConfigFor(
	reporter *reporter.Object,
	clusterConfig ocm.Spec,
	awsCreator *aws.Creator,
	awsCredentialsGetter aws.AccessKeyGetter,
) (ocm.Spec, error) {
	if clusterConfig.CustomProperties != nil && clusterConfig.CustomProperties[properties.UseLocalCredentials] ==
		strconv.FormatBool(true) {
		reporter.Warnf("Using local AWS access key for '%s'", awsCreator.ARN)
		var err error
		clusterConfig.AWSAccessKey, err = awsCredentialsGetter.GetLocalAWSAccessKeys()
		if err != nil {
			return clusterConfig, fmt.Errorf("Failed to get local AWS credentials: %w", err)
		}
	} else if clusterConfig.RoleARN == "" {
		// Create the access key for the AWS user:
		var err error
		clusterConfig.AWSAccessKey, err = awsCredentialsGetter.GetAWSAccessKeys()
		if err != nil {
			return clusterConfig, fmt.Errorf("Failed to get access keys for user '%s': %w",
				aws.AdminUserName, err)
		}
	}
	return clusterConfig, nil
}

func provideBillingAccount(billingAccounts []string, accountID string, r *rosa.Runtime) (string, error) {
	if !helper.ContainsPrefix(billingAccounts, accountID) {
		return "", fmt.Errorf("A billing account is required for Hosted Control Plane clusters. %s",
			listBillingAccountMessage)
	}

	billingAccount := accountID

	r.Reporter.Infof("Using '%s' as billing account", billingAccount)
	r.Reporter.Infof(
		"To use a different billing account, add --billing-account xxxxxxxxxx to previous command",
	)
	return billingAccount, nil
}

// validateNetworkType ensure user passes a valid network type parameter at creation
func validateNetworkType(networkType string) error {
	if networkType == "" {
		// Parameter not specified, nothing to do
		return nil
	}
	if !helper.Contains(ocm.NetworkTypes, networkType) {
		return fmt.Errorf(fmt.Sprintf("Expected a valid network type. Valid values: %v", ocm.NetworkTypes))
	}
	return nil
}

func GetBillingAccountContracts(cloudAccounts []*accountsv1.CloudAccount,
	billingAccount string) ([]*accountsv1.Contract, bool) {
	var contracts []*accountsv1.Contract
	for _, account := range cloudAccounts {
		if account.CloudAccountID() == billingAccount {
			contracts = account.Contracts()
			if ocm.HasValidContracts(account) {
				return contracts, true
			}
		}
	}
	return contracts, false
}

func GenerateContractDisplay(contract *accountsv1.Contract) string {
	format := "Jan 02, 2006"
	dimensions := contract.Dimensions()

	numberOfVCPUs, numberOfClusters := ocm.GetNumsOfVCPUsAndClusters(dimensions)

	contractDisplay := fmt.Sprintf(`
   +---------------------+----------------+ 
   | Start Date          |%s    | 
   | End Date            |%s    | 
   | Number of vCPUs:    |'%s'             | 
   | Number of clusters: |'%s'             | 
   +---------------------+----------------+ 
`,
		contract.StartDate().Format(format),
		contract.EndDate().Format(format),
		strconv.Itoa(numberOfVCPUs),
		strconv.Itoa(numberOfClusters),
	)

	return contractDisplay

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

func filterPrivateSubnets(initialSubnets []ec2types.Subnet, r *rosa.Runtime) ([]ec2types.Subnet, []string) {
	excludedSubnetsDueToPublic := []string{}
	filteredSubnets := []ec2types.Subnet{}
	publicSubnetMap, err := r.AWSClient.FetchPublicSubnetMap(initialSubnets)
	if err != nil {
		r.Reporter.Errorf("Unable to check if subnet have an IGW: %v", err)
		os.Exit(1)
	}
	for _, subnet := range initialSubnets {
		skip := false
		if isPublic, ok := publicSubnetMap[awssdk.ToString(subnet.SubnetId)]; ok {
			if isPublic {
				excludedSubnetsDueToPublic = append(
					excludedSubnetsDueToPublic,
					awssdk.ToString(subnet.SubnetId),
				)
				skip = true
			}
		}
		if !skip {
			filteredSubnets = append(filteredSubnets, subnet)
		}
	}
	if len(excludedSubnetsDueToPublic) > 0 {
		r.Reporter.Warnf("The following subnets have been excluded"+
			" because they have an Internet Gateway Targetded Route and the Cluster choice is private: %s",
			helper.SliceToSortedString(excludedSubnetsDueToPublic))
	}
	return filteredSubnets, excludedSubnetsDueToPublic
}

// filterCidrRangeSubnets filters the initial set of subnets to those that are part of the machine network,
// and not part of the service network
func filterCidrRangeSubnets(
	initialSubnets []ec2types.Subnet,
	machineNetwork *net.IPNet,
	serviceNetwork *net.IPNet,
	r *rosa.Runtime,
) ([]ec2types.Subnet, error) {
	excludedSubnetsDueToCidr := []string{}
	filteredSubnets := []ec2types.Subnet{}
	for _, subnet := range initialSubnets {
		skip := false
		subnetIP, subnetNetwork, err := net.ParseCIDR(*subnet.CidrBlock)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse subnet CIDR: %w", err)
		}

		if !isValidCidrRange(subnetIP, subnetNetwork, machineNetwork, serviceNetwork) {
			excludedSubnetsDueToCidr = append(excludedSubnetsDueToCidr, awssdk.ToString(subnet.SubnetId))
			skip = true
		}

		if !skip {
			filteredSubnets = append(filteredSubnets, subnet)
		}
	}
	if len(excludedSubnetsDueToCidr) > 0 {
		r.Reporter.Warnf("The following subnets have been excluded"+
			" because they do not fit into chosen CIDR ranges: %s", helper.SliceToSortedString(excludedSubnetsDueToCidr))
	}
	return filteredSubnets, nil
}

func isValidCidrRange(
	subnetIP net.IP,
	subnetNetwork *net.IPNet,
	machineNetwork *net.IPNet,
	serviceNetwork *net.IPNet,
) bool {
	return machineNetwork.Contains(subnetIP) &&
		!subnetNetwork.Contains(serviceNetwork.IP) &&
		!serviceNetwork.Contains(subnetIP)
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

const hostedCPFlag = "--hosted-cp"

func buildCommand(spec ocm.Spec, operatorRolesPrefix string,
	operatorRolePath string, userSelectedAvailabilityZones bool, labels string,
	properties []string) string {
	command := "rosa create cluster"
	command += fmt.Sprintf(" --cluster-name %s", spec.Name)
	if spec.DomainPrefix != "" {
		command += fmt.Sprintf(" --domain-prefix %s", spec.DomainPrefix)
	}

	if spec.IsSTS {
		command += " --sts"
		if spec.Mode != "" {
			command += fmt.Sprintf(" --mode %s", spec.Mode)
		}
	}
	if spec.ClusterAdminUser != "" {
		argAdded := false
		// Checks if admin password is from user (both flag and interactive)
		if args.clusterAdminPassword != "" && spec.ClusterAdminPassword != "" {
			command += fmt.Sprintf(" --cluster-admin-password %s", spec.ClusterAdminPassword)
			argAdded = true
		}
		if spec.ClusterAdminUser != admin.ClusterAdminUsername {
			command += fmt.Sprintf(" --cluster-admin-user %s", spec.ClusterAdminUser)
			argAdded = true
		}
		if !argAdded {
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
	if args.externalAuthProvidersEnabled {
		command += fmt.Sprintf(" --%s", ExternalAuthProvidersEnabledFlag)
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

	if spec.Ec2MetadataHttpTokens != "" {
		command += fmt.Sprintf(" --ec2-metadata-http-tokens %s", spec.Ec2MetadataHttpTokens)
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
		command += fmt.Sprintf(" --%s \"%s\"", arguments.NewDefaultMPLabelsFlag, labels)
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
		// TODO: Change flag names here when we deprecate the old flags
		command += fmt.Sprintf(" --%s %s", ingressPrivateHostedZoneIdFlag, spec.PrivateHostedZoneID)
		command += fmt.Sprintf(" --%s %s", route53RoleArnFlag, spec.SharedVPCRoleArn)
		command += fmt.Sprintf(" --base-domain %s", spec.BaseDomain)
	}
	if spec.InternalCommunicationHostedZoneId != "" {
		command += fmt.Sprintf(" --%s %s", hcpInternalCommunicationHostedZoneIdFlag,
			spec.InternalCommunicationHostedZoneId)
		command += fmt.Sprintf(" --%s %s", vpcEndpointRoleArnFlag, spec.VpcEndpointRoleArn)
	}
	if spec.FIPS {
		command += " --fips"
	} else if spec.EtcdEncryption {
		command += " --etcd-encryption"
		if spec.EtcdEncryptionKMSArn != "" {
			command += fmt.Sprintf(" --etcd-encryption-kms-arn %s", spec.EtcdEncryptionKMSArn)
		}
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
		command += " " + hostedCPFlag
	}

	if spec.AuditLogRoleARN != nil && *spec.AuditLogRoleARN != "" {
		command += fmt.Sprintf(" --audit-log-arn %s", *spec.AuditLogRoleARN)
	}
	if spec.MachinePoolRootDisk != nil {
		machinePoolRootDiskSize := spec.MachinePoolRootDisk.Size
		if machinePoolRootDiskSize != 0 {
			command += fmt.Sprintf(" --%s %dGiB", workerDiskSizeFlag, machinePoolRootDiskSize)
		}
	}

	if !reflect.DeepEqual(spec.DefaultIngress, ocm.NewDefaultIngressSpec()) {
		if len(spec.DefaultIngress.RouteSelectors) != 0 {
			selectors := []string{}
			for k, v := range spec.DefaultIngress.RouteSelectors {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressRouteSelectorFlag, strings.Join(selectors, ","))
		}
		if len(spec.DefaultIngress.ExcludedNamespaces) != 0 {
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressExcludedNamespacesFlag,
				strings.Join(spec.DefaultIngress.ExcludedNamespaces, ","))
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.WildcardPolicy) {
			command += fmt.Sprintf(
				" --%s %s",
				ingress.DefaultIngressWildcardPolicyFlag,
				spec.DefaultIngress.WildcardPolicy,
			)
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.NamespaceOwnershipPolicy) {
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressNamespaceOwnershipPolicyFlag,
				spec.DefaultIngress.NamespaceOwnershipPolicy)
		}
	}

	command += clusterautoscaler.BuildAutoscalerOptions(spec.AutoscalerConfig, clusterAutoscalerFlagsPrefix)
	command += clusterregistryconfig.BuildRegistryConfigOptions(spec)

	if len(spec.AdditionalComputeSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.ComputeSecurityGroupFlag,
			strings.Join(spec.AdditionalComputeSecurityGroupIds, ","))
	}

	if len(spec.AdditionalInfraSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.InfraSecurityGroupFlag,
			strings.Join(spec.AdditionalInfraSecurityGroupIds, ","))
	}

	if len(spec.AdditionalControlPlaneSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.ControlPlaneSecurityGroupFlag,
			strings.Join(spec.AdditionalControlPlaneSecurityGroupIds, ","))
	}

	if spec.BillingAccount != "" {
		command += fmt.Sprintf(" --billing-account %s", spec.BillingAccount)
	}

	if spec.NoCni {
		command += " --no-cni"
	}

	if len(spec.AdditionalAllowedPrincipals) > 0 {
		command += fmt.Sprintf(" --additional-allowed-principals %s",
			strings.Join(spec.AdditionalAllowedPrincipals, ","))
	}

	for _, p := range properties {
		command += fmt.Sprintf(" --properties \"%s\"", p)
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

func getInitialValidSubnets(awsClient aws.Client, ids []string, r *reporter.Object) ([]ec2types.Subnet, error) {
	var initialValidSubnets []ec2types.Subnet
	rhManagedSubnets := []string{}
	localZoneSubnets := []string{}

	validSubnets, err := awsClient.ListSubnets(ids...)

	if err != nil {
		return initialValidSubnets, err
	}

	for _, subnet := range validSubnets {
		hasRHManaged := tags.Ec2ResourceHasTag(subnet.Tags, tags.RedHatManaged, strconv.FormatBool(true))
		if hasRHManaged {
			rhManagedSubnets = append(rhManagedSubnets, awssdk.ToString(subnet.SubnetId))
		} else {
			zoneType, err := awsClient.GetAvailabilityZoneType(awssdk.ToString(subnet.AvailabilityZone))
			if err != nil {
				return initialValidSubnets, err
			}
			if zoneType == aws.LocalZone || zoneType == aws.WavelengthZone {
				localZoneSubnets = append(localZoneSubnets, awssdk.ToString(subnet.SubnetId))
			} else {
				initialValidSubnets = append(initialValidSubnets, subnet)
			}
		}
	}
	if len(rhManagedSubnets) != 0 {
		r.Warnf("The following subnets were excluded because they belong"+
			" to a VPC that is managed by Red Hat: %s", helper.SliceToSortedString(rhManagedSubnets))
	}
	if len(localZoneSubnets) != 0 {
		r.Warnf("The following subnets were excluded because they are on local zone or wavelength zone: %s",
			helper.SliceToSortedString(localZoneSubnets))
	}
	return initialValidSubnets, nil
}

func outputClusterAdminDetails(r *rosa.Runtime, isClusterAdmin bool, createAdminUser, createAdminPassword string) {
	if isClusterAdmin {
		r.Reporter.Infof("cluster admin user is %s", createAdminUser)
		r.Reporter.Infof("cluster admin password is %s", createAdminPassword)
	}
}

func getSecurityGroups(r *rosa.Runtime, cmd *cobra.Command, isVersionCompatibleComputeSgIds bool,
	kind string, useExistingVpc bool, currentSubnets []ec2types.Subnet, subnetIds []string,
	additionalSgIds *[]string) {
	hasChangedSgIdsFlag := cmd.Flags().Changed(securitygroups.SgKindFlagMap[kind])
	if hasChangedSgIdsFlag {
		if !useExistingVpc {
			r.Reporter.Errorf("Setting the `%s` flag is only allowed for BYO VPC clusters",
				securitygroups.SgKindFlagMap[kind])
			os.Exit(1)
		}
		if !isVersionCompatibleComputeSgIds {
			formattedVersion, err := versions.FormatMajorMinorPatch(
				ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay1,
			)
			if err != nil {
				r.Reporter.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
				os.Exit(1)
			}
			r.Reporter.Errorf("Parameter '%s' is not supported prior to version '%s'",
				securitygroups.SgKindFlagMap[kind], formattedVersion)
			os.Exit(1)
		}
	} else if interactive.Enabled() && isVersionCompatibleComputeSgIds && useExistingVpc {
		vpcId := ""
		for _, subnet := range currentSubnets {
			if awssdk.ToString(subnet.SubnetId) == subnetIds[0] {
				vpcId = awssdk.ToString(subnet.VpcId)
			}
		}
		if vpcId == "" {
			r.Reporter.Warnf("Unexpected situation a VPC ID should have been selected based on chosen subnets")
			os.Exit(1)
		}
		*additionalSgIds = interactiveSgs.
			GetSecurityGroupIds(r, cmd, vpcId, kind, "")
	}
	for i, sg := range *additionalSgIds {
		(*additionalSgIds)[i] = strings.TrimSpace(sg)
	}
}

func getMachinePoolRootDisk(r *rosa.Runtime, cmd *cobra.Command, version string,
	isHostedCP bool, defaultMachinePoolRootDiskSize int) (machinePoolRootDisk *ocm.Volume, err error) {
	var isVersionCompatibleMachinePoolRootDisk bool
	if !isHostedCP {
		isVersionCompatibleMachinePoolRootDisk, err = versions.IsGreaterThanOrEqual(
			version, ocm.MinVersionForMachinePoolRootDisk)
		if err != nil {
			return nil, fmt.Errorf("There was a problem checking version compatibility: %v", err)
		}
		if !isVersionCompatibleMachinePoolRootDisk && cmd.Flags().Changed(workerDiskSizeFlag) {
			formattedVersion, err := versions.FormatMajorMinorPatch(ocm.MinVersionForMachinePoolRootDisk)
			if err != nil {
				r.Reporter.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
				os.Exit(1)
			}
			return nil, fmt.Errorf(
				"Updating Worker disk size is not supported for versions prior to '%s'",
				formattedVersion,
			)
		}
	}

	if (isVersionCompatibleMachinePoolRootDisk || isHostedCP) &&
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
			if isHostedCP {
				machinePoolRootDiskSizeStr, err = interactive.GetString(interactive.Input{
					Question: "Machine pool root disk size (GiB or TiB)",
					Help:     cmd.Flags().Lookup(workerDiskSizeFlag).Usage,
					Default:  machinePoolRootDiskSizeStr,
					Validators: []interactive.Validator{
						interactive.NodePoolRootDiskSizeValidator(),
					},
				})
			} else {
				machinePoolRootDiskSizeStr, err = interactive.GetString(interactive.Input{
					Question: "Machine pool root disk size (GiB or TiB)",
					Help:     cmd.Flags().Lookup(workerDiskSizeFlag).Usage,
					Default:  machinePoolRootDiskSizeStr,
					Validators: []interactive.Validator{
						interactive.MachinePoolRootDiskSizeValidator(version),
					},
				})
			}
			if err != nil {
				return nil, fmt.Errorf("Expected a valid machine pool root disk size value: %v", err)
			}
		}

		// Parse the value given by either CLI or interactive mode and return it in GigiBytes
		machinePoolRootDiskSize, err := ocm.ParseDiskSizeToGigibyte(machinePoolRootDiskSizeStr)
		if err != nil {
			return nil, fmt.Errorf("Expected a valid machine pool root disk size value '%s': %v",
				machinePoolRootDiskSizeStr, err)

		}

		if isHostedCP {
			err = diskValidator.ValidateNodePoolRootDiskSize(machinePoolRootDiskSize)
		} else {
			err = diskValidator.ValidateMachinePoolRootDiskSize(version, machinePoolRootDiskSize)
		}
		if err != nil {
			return nil, err
		}

		// If the size given by the user is different than the default, we just let the OCM server
		// handle the default root disk size
		if machinePoolRootDiskSize != defaultMachinePoolRootDiskSize {
			machinePoolRootDisk = &ocm.Volume{
				Size: machinePoolRootDiskSize,
			}
		}
	}

	return machinePoolRootDisk, nil
}

func clusterHasLongNameWithoutDomainPrefix(clusterName, domainPrefix string) bool {
	return domainPrefix == "" && len(clusterName) > ocm.MaxClusterDomainPrefixLength
}

func validateUniqueIamRoleArnsForStsCluster(accountRoles []string, operatorRoles []ocm.OperatorIAMRole) error {
	tempRoleList := []string{}
	tempRoleList = append(tempRoleList, accountRoles...)

	for _, role := range operatorRoles {
		tempRoleList = append(tempRoleList, role.RoleARN)
	}
	duplicate, found := aws.HasDuplicates(tempRoleList)
	if found {
		return fmt.Errorf(duplicateIamRoleArnErrorMsg, duplicate)
	}

	return nil
}
