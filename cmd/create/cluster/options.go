package cluster

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	clustervalidations "github.com/openshift-online/ocm-common/pkg/cluster/validations"
	idputils "github.com/openshift-online/ocm-common/pkg/idp/utils"
	passwordValidator "github.com/openshift-online/ocm-common/pkg/idp/validations"
	diskValidator "github.com/openshift-online/ocm-common/pkg/machinepool/validations"
	kmsArnRegexpValidator "github.com/openshift-online/ocm-common/pkg/resource/validations"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/helper"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/errors"
)

const (
	OidcConfigIdFlag      = "oidc-config-id"
	ClassicOidcConfigFlag = "classic-oidc-config"
	workerDiskSizeFlag    = "worker-disk-size"
	// #nosec G101
	Ec2MetadataHttpTokensFlag = "ec2-metadata-http-tokens"

	clusterAutoscalerFlagsPrefix = "autoscaler-"

	MinReplicasSingleAZ = 2
	MinReplicaMultiAZ   = 3

	listInputMessage = "Format should be a comma-separated list."

	// nolint:lll
	createVpcForHcpDoc = "https://docs.openshift.com/rosa/rosa_hcp/rosa-hcp-sts-creating-a-cluster-quickly.html#rosa-hcp-creating-vpc"
)

type Options struct {
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
	noCni                bool

	// Cluster Admin
	createAdminUser      bool
	clusterAdminPassword string
	// Deprecated Cluster Admin
	clusterAdminUser string

	// Audit Log Forwarding
	auditLogRoleARN string

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

	// Infra machine pool attributes
	additionalInfraSecurityGroupIds []string

	// Control Plane machine pool attributes
	additionalControlPlaneSecurityGroupIds []string

	autoscalerArgs                *clusterautoscaler.AutoscalerArgs
	userSpecifiedAutoscalerValues []*pflag.Flag
}

func NewOptions() *Options {
	return &Options{
		channelGroup:   ocm.DefaultChannelGroup,
		flavour:        "osd-4",
		computeNodes:   2,
		minReplicas:    2,
		maxReplicas:    2,
		autoscalerArgs: clusterautoscaler.NewAutoscalerArgs(),
	}
}

func (o *Options) AddFlags(flags *pflag.FlagSet) {
	o.autoscalerArgs.AddClusterAutoscalerFlags(flags, clusterAutoscalerFlagsPrefix)

	// Basic options
	flags.StringVarP(
		&o.clusterName,
		"name",
		"n",
		"",
		"Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.",
	)
	flags.MarkDeprecated("name", "use --cluster-name instead")

	flags.StringVarP(
		&o.clusterName,
		"cluster-name",
		"c",
		"",
		"Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.",
	)

	flags.BoolVar(
		&o.sts,
		"sts",
		false,
		"Use AWS Security Token Service (STS) instead of IAM credentials to deploy your cluster.",
	)
	flags.BoolVar(
		&o.nonSts,
		"non-sts",
		false,
		"Use legacy method of creating clusters (IAM mode).",
	)
	flags.BoolVar(
		&o.nonSts,
		"mint-mode",
		false,
		"Use legacy method of creating clusters (IAM mode). This is an alias for --non-sts.",
	)
	flags.StringVar(
		&o.roleARN,
		"role-arn",
		"",
		"The Amazon Resource Name of the role that OpenShift Cluster Manager will assume to create the cluster.",
	)
	flags.StringVar(
		&o.externalID,
		"external-id",
		"",
		"An optional unique identifier that might be required when you assume a role in another account.",
	)
	flags.StringVar(
		&o.supportRoleARN,
		"support-role-arn",
		"",
		"The Amazon Resource Name of the role used by Red Hat SREs to enable "+
			"access to the cluster account in order to provide support.",
	)

	flags.StringVar(
		&o.controlPlaneRoleARN,
		"controlplane-iam-role",
		"",
		"The IAM role ARN that will be attached to control plane instances.",
	)

	flags.StringVar(
		&o.controlPlaneRoleARN,
		"master-iam-role",
		"",
		"The IAM role ARN that will be attached to master instances.",
	)
	flags.MarkDeprecated("master-iam-role", "use --controlplane-iam-role instead")

	flags.StringVar(
		&o.workerRoleARN,
		"worker-iam-role",
		"",
		"The IAM role ARN that will be attached to worker instances.",
	)

	flags.StringArrayVar(
		&o.operatorIAMRoles,
		"operator-iam-roles",
		nil,
		"List of OpenShift name and namespace, and role ARNs used to perform credential "+
			"requests by operators needed in the OpenShift installer.",
	)
	flags.MarkDeprecated("operator-iam-roles", "use --operator-roles-prefix instead")
	flags.StringVar(
		&o.operatorRolesPrefix,
		"operator-roles-prefix",
		"",
		"Prefix to use for all IAM roles used by the operators needed in the OpenShift installer. "+
			"Leave empty to use an auto-generated one.",
	)

	flags.StringVar(
		&o.oidcConfigId,
		OidcConfigIdFlag,
		"",
		"Registered OIDC Configuration ID to use for cluster creation",
	)

	flags.BoolVar(
		&o.classicOidcConfig,
		ClassicOidcConfigFlag,
		false,
		"Use classic OIDC configuration without registering an ID.",
	)
	flags.MarkHidden(ClassicOidcConfigFlag)

	flags.StringSliceVar(
		&o.tags,
		"tags",
		nil,
		"Apply user defined tags to all resources created by ROSA in AWS. "+
			"Tags are comma separated, for example: 'key value, foo bar'",
	)

	flags.BoolVar(
		&o.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	arguments.AddRegionFlag(flags)
	flags.StringVar(
		&o.version,
		"version",
		"",
		"Version of OpenShift that will be used to install the cluster, for example \"4.3.10\"",
	)
	flags.StringVar(
		&o.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the group where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	flags.StringVar(
		&o.flavour,
		"flavour",
		"osd-4",
		"Set of predefined properties of a cluster",
	)
	flags.MarkHidden("flavour")

	flags.BoolVar(
		&o.etcdEncryption,
		"etcd-encryption",
		false,
		"Add etcd encryption. By default etcd data is encrypted at rest. "+
			"This option configures etcd encryption on top of existing storage encryption.",
	)
	flags.BoolVar(
		&o.fips,
		"fips",
		false,
		"Create cluster that uses FIPS Validated / Modules in Process cryptographic libraries.",
	)

	flags.StringVar(
		&o.httpProxy,
		"http-proxy",
		"",
		"A proxy URL to use for creating HTTP connections outside the cluster. The URL scheme must be http.",
	)

	flags.StringVar(
		&o.httpsProxy,
		"https-proxy",
		"",
		"A proxy URL to use for creating HTTPS connections outside the cluster.",
	)

	flags.StringSliceVar(
		&o.noProxySlice,
		"no-proxy",
		nil,
		"A comma-separated list of destination domain names, domains, IP addresses or "+
			"other network CIDRs to exclude proxying.",
	)

	flags.StringVar(
		&o.additionalTrustBundleFile,
		"additional-trust-bundle-file",
		"",
		"A file contains a PEM-encoded X.509 certificate bundle that will be "+
			"added to the nodes' trusted certificate store.")

	flags.BoolVar(&o.enableCustomerManagedKey,
		"enable-customer-managed-key",
		false,
		"Enable to specify your KMS Key to encrypt EBS instance volumes. By default account’s default "+
			"KMS key for that particular region is used.")

	flags.StringVar(&o.kmsKeyARN,
		"kms-key-arn",
		"",
		"The key ARN is the Amazon Resource Name (ARN) of a CMK. It is a unique, "+
			"fully qualified identifier for the CMK. A key ARN includes the AWS account, Region, and the key ID.")

	flags.StringVar(&o.etcdEncryptionKmsARN,
		"etcd-encryption-kms-arn",
		"",
		"The etcd encryption kms key ARN is the key used to encrypt etcd. "+
			"If set it will override etcd-encryption flag to true. It is a unique, "+
			"fully qualified identifier for the CMK. A key ARN includes the AWS account, Region, and the key ID.")

	flags.StringVar(
		&o.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&o.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	flags.MarkHidden("expiration-time")
	flags.MarkHidden("expiration")

	flags.BoolVar(
		&o.privateLink,
		"private-link",
		false,
		"Provides private connectivity between VPCs, AWS services, and your on-premises networks, "+
			"without exposing your traffic to the public internet.",
	)

	flags.StringVar(
		&o.ec2MetadataHttpTokens,
		Ec2MetadataHttpTokensFlag,
		"",
		"Should cluster nodes use both v1 and v2 endpoints or just v2 endpoint "+
			"of EC2 Instance Metadata Service (IMDS)",
	)

	flags.StringSliceVar(
		&o.subnetIDs,
		"subnet-ids",
		nil,
		"The Subnet IDs to use when installing the cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for installer provisioned subnet IDs.",
	)

	flags.StringSliceVar(
		&o.availabilityZones,
		"availability-zones",
		nil,
		"The availability zones to use when installing a non-BYOVPC cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for the installer to pick availability zones")

	// Scaling options
	flags.StringVar(
		&o.computeMachineType,
		"compute-machine-type",
		"",
		"Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.",
	)

	flags.IntVar(
		&o.computeNodes,
		"compute-nodes",
		2,
		"Number of worker nodes to provision. Single zone clusters need at least 2 nodes, "+
			"multizone clusters need at least 3 nodes.",
	)
	flags.MarkDeprecated("compute-nodes", "use --replicas instead")
	flags.IntVar(
		&o.computeNodes,
		"replicas",
		2,
		"Number of worker nodes to provision. Single zone clusters need at least 2 nodes, "+
			"multizone clusters need at least 3 nodes. Hosted clusters require that the number of worker nodes be a "+
			"multiple of the number of private subnets.",
	)

	flags.BoolVar(
		&o.autoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling of compute nodes.",
	)

	// iterates through all autoscaling flags and stores them in slice to track user input
	flags.VisitAll(func(f *pflag.Flag) {
		if strings.HasPrefix(f.Name, clusterAutoscalerFlagsPrefix) {
			o.userSpecifiedAutoscalerValues = append(o.userSpecifiedAutoscalerValues, f)
		}
	})

	flags.IntVar(
		&o.minReplicas,
		"min-replicas",
		2,
		"Minimum number of compute nodes.",
	)

	flags.IntVar(
		&o.maxReplicas,
		"max-replicas",
		2,
		"Maximum number of compute nodes.",
	)

	flags.SetNormalizeFunc(arguments.NormalizeFlags)
	flags.StringVar(
		&o.defaultMachinePoolLabels,
		arguments.NewDefaultMPLabelsFlag,
		"",
		"Labels for the worker machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&o.networkType,
		"network-type",
		"",
		"The main controller responsible for rendering the core networking components.",
	)
	flags.MarkHidden("network-type")

	flags.IPNetVar(
		&o.machineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	flags.IPNetVar(
		&o.serviceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	flags.IPNetVar(
		&o.podCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	flags.IntVar(
		&o.hostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set "+
			"to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
	)
	flags.BoolVar(
		&o.private,
		"private",
		false,
		"Restrict master API endpoint and application routes to direct, private connectivity.",
	)

	flags.BoolVar(
		&o.disableSCPChecks,
		"disable-scp-checks",
		false,
		"Indicates if cloud permission checks are disabled when attempting installation of the cluster.",
	)
	flags.BoolVar(
		&o.disableWorkloadMonitoring,
		"disable-workload-monitoring",
		false,
		"Enables you to monitor your own projects in isolation from Red Hat Site Reliability Engineer (SRE) "+
			"platform metrics.",
	)

	flags.BoolVarP(
		&o.watch,
		"watch",
		"w",
		false,
		"Watch cluster installation logs.",
	)

	flags.BoolVar(
		&o.dryRun,
		"dry-run",
		false,
		"Simulate creating the cluster.",
	)

	flags.BoolVar(
		&o.fakeCluster,
		"fake-cluster",
		false,
		"Create a fake cluster that uses no AWS resources.",
	)
	flags.MarkHidden("fake-cluster")

	flags.StringArrayVar(
		&o.properties,
		"properties",
		nil,
		"User defined properties for tagging and querying.",
	)
	flags.MarkHidden("properties")

	flags.BoolVar(
		&o.useLocalCredentials,
		"use-local-credentials",
		false,
		"Use local AWS credentials instead of the 'osdCcsAdmin' user. This is not supported.",
	)
	flags.MarkHidden("use-local-credentials")

	flags.StringVar(
		&o.operatorRolesPermissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the operator "+
			"roles in STS clusters.",
	)

	// Options related to HyperShift:
	flags.BoolVar(
		&o.hostedClusterEnabled,
		"hosted-cp",
		false,
		"Enable the use of Hosted Control Planes",
	)

	flags.StringVar(&o.machinePoolRootDiskSize,
		workerDiskSizeFlag,
		"",
		"Default worker machine pool root disk size with a **unit suffix** like GiB or TiB, "+
			"e.g. 200GiB.")

	flags.StringVar(
		&o.billingAccount,
		"billing-account",
		"",
		"Account used for billing subscriptions purchased via the AWS marketplace",
	)

	flags.BoolVar(
		&o.createAdminUser,
		"create-admin-user",
		false,
		`Create cluster admin named "cluster-admin"`,
	)

	flags.BoolVar(
		&o.noCni,
		"no-cni",
		false,
		"Disable CNI creation to let users bring their own CNI.",
	)

	flags.StringVar(
		&o.clusterAdminUser,
		"cluster-admin-user",
		"",
		`Deprecated cluster admin flag. Please use --create-admin-user.`,
	)
	flags.StringVar(
		&o.clusterAdminPassword,
		"cluster-admin-password",
		"",
		`The password must
		- Be at least 14 characters (ASCII-standard) without whitespaces
		- Include uppercase letters, lowercase letters, and numbers or symbols (ASCII-standard characters only)`,
	)
	// cluster admin flags deprecated to be removed
	flags.MarkHidden("cluster-admin-user")

	flags.StringVar(
		&o.auditLogRoleARN,
		"audit-log-arn",
		"",
		"The ARN of the role that is used to forward audit logs to AWS CloudWatch.",
	)

	flags.StringVar(
		&o.defaultIngressRouteSelectors,
		ingress.DefaultIngressRouteSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	flags.StringVar(
		&o.defaultIngressExcludedNamespaces,
		ingress.DefaultIngressExcludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&o.defaultIngressWildcardPolicy,
		ingress.DefaultIngressWildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ingress.ValidWildcardPolicies, ","), ingress.DefaultWildcardPolicy),
	)

	flags.StringVar(
		&o.defaultIngressNamespaceOwnershipPolicy,
		ingress.DefaultIngressNamespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ingress.ValidNamespaceOwnershipPolicies, ","), ingress.DefaultNamespaceOwnershipPolicy),
	)

	flags.StringVar(
		&o.privateHostedZoneID,
		"private-hosted-zone-id",
		"",
		"ID assigned by AWS to private Route 53 hosted zone associated with intended shared VPC, "+
			"e.g., 'Z05646003S02O1ENCDCSN'.",
	)

	flags.StringVar(
		&o.sharedVPCRoleARN,
		"shared-vpc-role-arn",
		"",
		"AWS IAM role ARN with a policy attached, granting permissions necessary to create and manage Route 53 DNS records "+
			"in private Route 53 hosted zone associated with intended shared VPC.",
	)

	flags.StringVar(
		&o.baseDomain,
		"base-domain",
		"",
		"Base DNS domain name previously reserved and matching the hosted zone name of the private Route 53 hosted zone "+
			"associated with intended shared VPC, e.g., '1vo8.p1.openshiftapps.com'.",
	)

	flags.StringSliceVar(
		&o.additionalComputeSecurityGroupIds,
		securitygroups.ComputeSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the default worker machine pool. "+
			listInputMessage,
	)

	flags.StringSliceVar(
		&o.additionalInfraSecurityGroupIds,
		securitygroups.InfraSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the infra worker nodes. "+
			listInputMessage,
	)

	flags.StringSliceVar(
		&o.additionalControlPlaneSecurityGroupIds,
		securitygroups.ControlPlaneSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the control plane nodes. "+
			listInputMessage,
	)
}

func (o *Options) Complete(flags *pflag.FlagSet, r *rosa.Runtime) (*CompletedOptions, error) {

	// Validate mode
	mode, err := aws.GetMode()
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	for _, val := range o.userSpecifiedAutoscalerValues {
		if val.Changed && !o.autoscalingEnabled {
			return nil, fmt.Errorf("using autoscaling flag '%s', requires flag '--enable-autoscaling'. "+
				"Please try again with flag", val.Name)
		}
	}

	// validate flags for cluster admin
	isHostedCP := o.hostedClusterEnabled
	createAdminUser := o.createAdminUser
	clusterAdminPassword := strings.Trim(o.clusterAdminPassword, " \t")
	if (createAdminUser || clusterAdminPassword != "") && isHostedCP {
		return nil, fmt.Errorf("setting Cluster Admin is only supported in classic ROSA clusters")
	}
	// error when using deprecated admin flags
	clusterAdminUser := strings.Trim(o.clusterAdminUser, " \t")
	if clusterAdminUser != "" {
		return nil, fmt.Errorf("'--cluster-admin-user' flag has been deprecated " +
			"and replaced with '--create-admin-user'")
	}

	supportedRegions, err := r.OCMClient.GetDatabaseRegionList()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve supported regions: %v", err)
	}
	awsClient := aws.GetAWSClientForUserRegion(r.Reporter, r.Logger, supportedRegions, o.useLocalCredentials)
	r.AWSClient = awsClient

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		return nil, fmt.Errorf("unable to get IAM credentials: %v", err)
	}

	shardPinningEnabled := false
	for _, value := range o.properties {
		if strings.HasPrefix(value, properties.ProvisionShardId) {
			shardPinningEnabled = true
			break
		}
	}

	isBYOVPC := flags.Changed("subnet-ids")
	isAvailabilityZonesSet := flags.Changed("availability-zones")
	// Setting subnet IDs is choosing BYOVPC implicitly,
	// and selecting availability zones is only allowed for non-BYOVPC clusters
	if isBYOVPC && isAvailabilityZonesSet {
		return nil, fmt.Errorf("setting availability zones is not supported for BYO VPC. " +
			"ROSA autodetects availability zones from subnet IDs provided")
	}

	// Select a multi-AZ cluster implicitly by providing three availability zones
	if len(o.availabilityZones) == clustervalidations.MultiAZCount {
		o.multiAZ = true
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Get cluster name
	clusterName := strings.Trim(o.clusterName, " \t")

	if clusterName == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	if interactive.Enabled() {
		clusterName, err = interactive.GetString(interactive.Input{
			Question: "Cluster name",
			Help:     flags.Lookup("cluster-name").Usage,
			Default:  clusterName,
			Required: true,
			Validators: []interactive.Validator{
				ocm.ClusterNameValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid cluster name: %w", err)
		}
	}

	// Trim names to remove any leading/trailing invisible characters
	clusterName = strings.Trim(clusterName, " \t")

	if !ocm.IsValidClusterName(clusterName) {
		return nil, fmt.Errorf("cluster name must consist" +
			" of no more than 15 lowercase alphanumeric characters or '-', " +
			"start with a letter, and end with an alphanumeric character")
	}

	if interactive.Enabled() {
		isHostedCP, err = interactive.GetBool(interactive.Input{
			Question: "Deploy cluster with Hosted Control Plane",
			Help:     flags.Lookup("hosted-cp").Usage,
			Default:  isHostedCP,
			Required: false,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid --hosted-cp value: %w", err)
		}
	}

	if isHostedCP && r.Reporter.IsTerminal() {
		techPreviewMsg, err := r.OCMClient.GetTechnologyPreviewMessage(ocm.HcpProduct, time.Now())
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}
		if techPreviewMsg != "" {
			r.Reporter.Infof(techPreviewMsg)
		}
	}

	if isHostedCP && flags.Changed(Ec2MetadataHttpTokensFlag) {
		return nil, fmt.Errorf("'%s' is not available for Hosted Control Plane clusters", Ec2MetadataHttpTokensFlag)
	}

	// Errors when users elects for cluster admin via flags and elects for hosted control plane via interactive prompt"
	if isHostedCP && (createAdminUser || clusterAdminPassword != "") {
		return nil, fmt.Errorf("setting Cluster Admin is only supported in classic ROSA clusters")
	}

	// isClusterAdmin is a flag indicating if user wishes to create cluster admin
	isClusterAdmin := false
	if !isHostedCP {
		if createAdminUser {
			isClusterAdmin = true
			// user supplies create-admin-user flag without cluster-admin-password will generate random password
			if clusterAdminPassword == "" {
				r.Reporter.Debugf(admin.GeneratingRandomPasswordString)
				clusterAdminPassword, err = idputils.GenerateRandomPassword()
				if err != nil {
					return nil, fmt.Errorf("failed to generate a random password")
				}
			}
		}
		// validates both user inputted custom password and randomly generated password
		if clusterAdminPassword != "" {
			err = passwordValidator.PasswordValidator(clusterAdminPassword)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
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
				return nil, fmt.Errorf("expected a valid value: %w", err)
			}
			if isClusterAdmin {
				isCustomAdminPassword, err := interactive.GetBool(interactive.Input{
					Question: "Create custom password for cluster admin",
					Default:  false,
					Required: true,
				})
				if err != nil {
					return nil, fmt.Errorf("expected a valid value: %w", err)
				}
				if !isCustomAdminPassword {
					clusterAdminPassword, err = idputils.GenerateRandomPassword()
					if err != nil {
						return nil, fmt.Errorf("failed to generate a random password")
					}
				} else {
					clusterAdminPassword = idp.GetIdpPasswordFromPrompt(flags, r,
						"cluster-admin-password", clusterAdminPassword)
					o.clusterAdminPassword = clusterAdminPassword
				}
			}
		}
		outputClusterAdminDetails(r, isClusterAdmin, clusterAdminPassword)
	}

	if isHostedCP && flags.Changed(arguments.NewDefaultMPLabelsFlag) {
		return nil, fmt.Errorf("setting the worker machine pool labels is not supported for hosted clusters")
	}

	// Billing Account
	billingAccount := o.billingAccount
	if isHostedCP {
		isHcpBillingTechPreview, err := r.OCMClient.IsTechnologyPreview(ocm.HcpBillingAccount, time.Now())
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}

		if !isHcpBillingTechPreview {

			if billingAccount != "" && !ocm.IsValidAWSAccount(billingAccount) {
				return nil, fmt.Errorf("billing account is invalid. Run the command again with a valid billing account")
			}

			cloudAccounts, err := r.OCMClient.GetBillingAccounts()
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}

			billingAccounts := ocm.GenerateBillingAccountsList(cloudAccounts)

			if billingAccount == "" && !interactive.Enabled() {
				// if a billing account is not provided we will try to use the infrastructure account as default
				if helper.ContainsPrefix(billingAccounts, awsCreator.AccountID) {
					billingAccount = awsCreator.AccountID
					r.Reporter.Infof("Using '%s' as billing account", billingAccount)
					r.Reporter.Infof(
						"To use a different billing account, add --billing-account xxxxxxxxxx to previous command",
					)
				} else {
					return nil, fmt.Errorf("a billing account is required for Hosted Control Plane clusters")
				}
			}

			if interactive.Enabled() {
				if len(billingAccounts) > 0 {
					billingAccount, err = interactive.GetOption(interactive.Input{
						Question: "Billing Account",
						Help:     flags.Lookup("billing-account").Usage,
						Default:  billingAccount,
						Required: true,
						Options:  billingAccounts,
					})

					if err != nil {
						return nil, fmt.Errorf("expected a valid billing account: '%s'", err)
					}

					billingAccount = aws.ParseOption(billingAccount)
				}

				if billingAccount == "" || !ocm.IsValidAWSAccount(billingAccount) {
					return nil, fmt.Errorf("expected a valid billing account")
				}

				// Get contract info
				contracts, isContractEnabled := GetBillingAccountContracts(cloudAccounts, billingAccount)

				if billingAccount != awsCreator.AccountID {
					r.Reporter.Infof(
						"The selected AWS billing account is a different account than your AWS infrastructure account." +
							"The AWS billing account will be charged for subscription usage. " +
							"The AWS infrastructure account will be used for managing the cluster.",
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
		return nil, fmt.Errorf("billing accounts are only supported for Hosted Control Plane clusters")
	}

	etcdEncryptionKmsARN := o.etcdEncryptionKmsARN

	if etcdEncryptionKmsARN != "" && !isHostedCP {
		return nil, fmt.Errorf("etcd encryption kms arn is only allowed for hosted cp")
	}

	// all hosted clusters are sts
	isSTS := o.sts || o.roleARN != "" || fedramp.Enabled() || isHostedCP
	isIAM := (flags.Changed("sts") && !isSTS) || o.nonSts

	if isSTS && isIAM {
		return nil, fmt.Errorf("can't use both STS and mint mode at the same time")
	}

	if interactive.Enabled() && (!isSTS && !isIAM) {
		isSTS, err = interactive.GetBool(interactive.Input{
			Question: "Deploy cluster using AWS STS",
			Help:     flags.Lookup("sts").Usage,
			Default:  true,
			Required: true,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid --sts value: %w", err)
		}
		isIAM = !isSTS
	}

	isSTS = isSTS || awsCreator.IsSTS

	if r.Reporter.IsTerminal() && !isHostedCP {
		r.Reporter.Warnf("In a future release STS will be the default mode.")
		r.Reporter.Warnf("--sts flag won't be necessary if you wish to use STS.")
		r.Reporter.Warnf("--non-sts/--mint-mode flag will be necessary if you do not wish to use STS.")
	}

	permissionsBoundary := o.operatorRolesPermissionsBoundary
	if permissionsBoundary != "" {
		err = aws.ARNValidator(permissionsBoundary)
		if err != nil {
			return nil, fmt.Errorf("expected a valid policy ARN for permissions boundary: %w", err)
		}
	}

	if isIAM {
		if awsCreator.IsSTS {
			return nil, fmt.Errorf("since your AWS credentials are returning an STS ARN you can only " +
				"create STS clusters. Otherwise, switch to IAM credentials")
		}
		err := awsClient.CheckAdminUserExists(aws.AdminUserName)
		if err != nil {
			return nil, fmt.Errorf("iAM user '%s' does not exist. Run `rosa init` first", aws.AdminUserName)
		}
		r.Reporter.Debugf("IAM user is valid!")
	}

	// AWS ARN Role
	roleARN := o.roleARN
	supportRoleARN := o.supportRoleARN
	controlPlaneRoleARN := o.controlPlaneRoleARN
	workerRoleARN := o.workerRoleARN

	// OpenShift version:
	version := o.version
	channelGroup := o.channelGroup
	versionList, err := versions.GetVersionList(r, channelGroup, isSTS, isHostedCP, isHostedCP, true)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	if version == "" {
		version = versionList[0]
	}
	if interactive.Enabled() {
		version, err = interactive.GetOption(interactive.Input{
			Question: "OpenShift version",
			Help:     flags.Lookup("version").Usage,
			Options:  versionList,
			Default:  version,
			Required: true,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid OpenShift version: %w", err)
		}
	}
	version, err = r.OCMClient.ValidateVersion(version, versionList, channelGroup, isSTS, isHostedCP)
	if err != nil {
		return nil, fmt.Errorf("expected a valid OpenShift version: %w", err)
	}
	if err := r.OCMClient.IsVersionCloseToEol(ocm.CloseToEolDays, version, channelGroup); err != nil {
		r.Reporter.Warnf("%v", err)
		if !confirm.Confirm("continue with version '%s'", ocm.GetRawVersionId(version)) {
			return nil, nil
		}
	}

	httpTokens := o.ec2MetadataHttpTokens
	if !isHostedCP {
		if httpTokens == "" {
			httpTokens = string(v1.Ec2MetadataHttpTokensOptional)
		}
		if interactive.Enabled() {
			httpTokens, err = interactive.GetOption(interactive.Input{
				Question: "Configure the use of IMDSv2 for ec2 instances",
				Options:  []string{string(v1.Ec2MetadataHttpTokensOptional), string(v1.Ec2MetadataHttpTokensRequired)},
				Help:     flags.Lookup(Ec2MetadataHttpTokensFlag).Usage,
				Required: true,
				Default:  httpTokens,
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid http tokens value : %v", err)
			}
		}
		if err = ocm.ValidateHttpTokensValue(httpTokens); err != nil {
			return nil, fmt.Errorf("expected a valid http tokens value : %v", err)
		}
		if err := ocm.ValidateHttpTokensVersion(ocm.GetVersionMinor(version), httpTokens); err != nil {
			return nil, fmt.Errorf(err.Error())
		}
	}

	// warn if mode is used for non sts cluster
	if !isSTS && mode != "" {
		r.Reporter.Warnf("--mode is only valid for STS clusters")
	}

	// validate mode passed is allowed value

	if isSTS && mode != "" {
		isValidMode := arguments.IsValidMode(aws.Modes, mode)
		if !isValidMode {
			return nil, fmt.Errorf("invalid --mode '%s'. Allowed values are %s", mode, aws.Modes)
		}
	}

	if o.watch && isSTS && mode == aws.ModeAuto && !confirm.Yes() {
		return nil, fmt.Errorf("cannot watch for STS cluster installation logs in mode 'auto' " +
			"without also supplying '--yes' option." +
			"To watch your cluster installation logs, run 'rosa logs install' instead after the cluster has began creating")
	}

	if o.watch && isSTS && mode == aws.ModeManual {
		return nil, fmt.Errorf("cannot watch for STS cluster installation logs in mode 'manual'." +
			"It requires manual commands to be performed as part of the process." +
			"To watch your cluster installation logs, run 'rosa logs install' after the cluster has began creating")
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
			return nil, fmt.Errorf("failed to find %s role: %s", role.Name, err)
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
					Help:     flags.Lookup(role.Flag).Usage,
					Options:  roleARNs,
					Default:  defaultRoleARN,
					Required: true,
				})
				if err != nil {
					return nil, fmt.Errorf("expected a valid role ARN: %w", err)
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
			r.Reporter.Warnf(fmt.Sprintf("No compatible account roles with version '%s' found. "+
				"You will need to manually set them in the next steps or run '%s' to create them first.",
				minor, createAccountRolesCommand))
			interactive.Enable()
		}

		if roleARN != "" {
			// check if role has hosted cp policy via AWS tag value
			hostedCPPolicies, err := awsClient.HasHostedCPPolicies(roleARN)
			if err != nil {
				return nil, fmt.Errorf("failed to determine if cluster has hosted CP policies: %v", err)
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
					return nil, fmt.Errorf("failed to find %s role: %s", role.Name, err)
				}
				selectedARN := ""
				expectedResourceIDForAccRole, rolePrefix, err := getExpectedResourceIDForAccRole(
					hostedCPPolicies, roleARN, roleType)
				if err != nil {
					return nil, fmt.Errorf("failed to get the expected resource ID for role type: %s", roleType)
				}
				r.Reporter.Debugf(
					"Using '%s' as the role prefix to retrieve the expected resource ID for role type '%s'",
					rolePrefix,
					roleType,
				)

				for _, rARN := range roleARNs {
					resourceId, err := aws.GetResourceIdFromARN(rARN)
					if err != nil {
						return nil, fmt.Errorf("failed to get resource ID from arn. %s", err)
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
			Help:     flags.Lookup("role-arn").Usage,
			Default:  roleARN,
			Required: isSTS,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid ARN: %w", err)
		}
	}

	if roleARN != "" {
		err = aws.ARNValidator(roleARN)
		if err != nil {
			return nil, fmt.Errorf("expected a valid Role ARN: %w", err)
		}
		isSTS = true
	}

	if !isSTS && mode != "" {
		r.Reporter.Warnf("--mode is only valid for STS clusters")
	}

	externalID := o.externalID
	if isSTS && interactive.Enabled() {
		externalID, err = interactive.GetString(interactive.Input{
			Question: "External ID",
			Help:     flags.Lookup("external-id").Usage,
			Validators: []interactive.Validator{
				interactive.RegExp(`^[\w+=,.@:\/-]*$`),
				interactive.MaxLength(1224),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid External ID: %w", err)
		}
	}

	// Ensure interactive mode if missing required role ARNs on STS clusters
	if isSTS && !hasRoles && !interactive.Enabled() && supportRoleARN == "" {
		interactive.Enable()
	}
	if isSTS && !hasRoles && interactive.Enabled() {
		supportRoleARN, err = interactive.GetString(interactive.Input{
			Question: "Support Role ARN",
			Help:     flags.Lookup("support-role-arn").Usage,
			Default:  supportRoleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid ARN: %w", err)
		}
	}
	if supportRoleARN != "" {
		err = aws.ARNValidator(supportRoleARN)
		if err != nil {
			return nil, fmt.Errorf("expected a valid Support Role ARN: %w", err)
		}
	} else if roleARN != "" {
		return nil, fmt.Errorf("support Role ARN is required: %w", err)
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
				Help:     flags.Lookup("controlplane-iam-role").Usage,
				Default:  controlPlaneRoleARN,
				Required: true,
				Validators: []interactive.Validator{
					aws.ARNValidator,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid control plane IAM role ARN: %w", err)
			}
		}
		if controlPlaneRoleARN != "" {
			err = aws.ARNValidator(controlPlaneRoleARN)
			if err != nil {
				return nil, fmt.Errorf("expected a valid control plane instance IAM role ARN: %w", err)
			}
		} else if roleARN != "" {
			return nil, fmt.Errorf("control plane instance IAM role ARN is required: %w", err)
		}
	}

	// Ensure interactive mode if missing required role ARNs on STS clusters
	if isSTS && !hasRoles && !interactive.Enabled() && workerRoleARN == "" {
		interactive.Enable()
	}

	if isSTS && !hasRoles && interactive.Enabled() {
		workerRoleARN, err = interactive.GetString(interactive.Input{
			Question: "Worker IAM Role ARN",
			Help:     flags.Lookup("worker-iam-role").Usage,
			Default:  workerRoleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid worker IAM role ARN: %w", err)
		}
	}
	if workerRoleARN != "" {
		err = aws.ARNValidator(workerRoleARN)
		if err != nil {
			return nil, fmt.Errorf("expected a valid worker instance IAM role ARN: %w", err)
		}
	} else if roleARN != "" {
		return nil, fmt.Errorf("worker instance IAM role ARN is required: %w", err)
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
		return nil, fmt.Errorf("failed to determine if cluster has managed policies: %v", err)
	}
	// check if role has hosted cp policy via AWS tag value
	hostedCPPolicies, err := awsClient.HasHostedCPPolicies(roleARN)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if cluster has hosted CP policies: %v", err)
	}

	if managedPolicies {
		rolePrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
		if err != nil {
			return nil, fmt.Errorf("failed to find prefix from account role: %w", err)
		}

		err = roles.ValidateAccountRolesManagedPolicies(r, rolePrefix, hostedCPPolicies)
		if err != nil {
			return nil, fmt.Errorf("failed while validating account roles: %w", err)
		}
	} else {
		err = roles.ValidateUnmanagedAccountRoles(roleARNs, awsClient, version)
		if err != nil {
			return nil, fmt.Errorf("failed while validating account roles: %w", err)
		}
	}

	operatorRolesPrefix := o.operatorRolesPrefix
	var operatorRoles []string
	expectedOperatorRolePath, _ := aws.GetPathFromARN(roleARN)
	operatorIAMRoles := o.operatorIAMRoles
	var computedOperatorIamRoleList []ocm.OperatorIAMRole
	if isSTS {
		if operatorRolesPrefix == "" {
			operatorRolesPrefix = getRolePrefix(clusterName)
		}
		if interactive.Enabled() {
			operatorRolesPrefix, err = interactive.GetString(interactive.Input{
				Question: "Operator roles prefix",
				Help:     flags.Lookup("operator-roles-prefix").Usage,
				Required: true,
				Default:  operatorRolesPrefix,
				Validators: []interactive.Validator{
					interactive.RegExp(aws.RoleNameRE.String()),
					interactive.MaxLength(32),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a prefix for the operator IAM roles: %w", err)
			}
		}
		if len(operatorRolesPrefix) == 0 {
			return nil, fmt.Errorf("expected a prefix for the operator IAM roles: %w", err)
		}
		if len(operatorRolesPrefix) > 32 {
			return nil, fmt.Errorf("expected a prefix with no more than 32 characters")
		}
		if !aws.RoleNameRE.MatchString(operatorRolesPrefix) {
			return nil, fmt.Errorf("expected valid operator roles prefix matching %s", aws.RoleNameRE.String())
		}

		credRequests, err := r.OCMClient.GetAllCredRequests()
		if err != nil {
			return nil, fmt.Errorf("error getting operator credential request from OCM %v", err)
		}
		operatorRoles, err = r.AWSClient.GetOperatorRolesFromAccountByPrefix(operatorRolesPrefix, credRequests)
		if err != nil {
			return nil, fmt.Errorf("there was a problem retrieving the Operator Roles from AWS: %v", err)
		}
	}

	var oidcConfig *v1.OidcConfig
	if isSTS {
		credRequests, err := r.OCMClient.GetCredRequests(isHostedCP)
		if err != nil {
			return nil, fmt.Errorf("error getting operator credential request from OCM %s", err)
		}
		accRolesPrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
		if err != nil {
			return nil, fmt.Errorf("failed to find prefix from account role: %w", err)
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
					return nil, fmt.Errorf("error validating operator role '%s' version %s", operator.Name(), err)
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
		if flags.Changed("operator-iam-roles") {
			computedOperatorIamRoleList = []ocm.OperatorIAMRole{}
			for _, role := range operatorIAMRoles {
				if !strings.Contains(role, ",") {
					return nil, fmt.Errorf("expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
				}
				roleData := strings.Split(role, ",")
				if len(roleData) != 3 {
					return nil, fmt.Errorf("expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
				}
				computedOperatorIamRoleList = append(computedOperatorIamRoleList, ocm.OperatorIAMRole{
					Name:      roleData[0],
					Namespace: roleData[1],
					RoleARN:   roleData[2],
				})
			}
		}
		oidcConfig, err = o.handleOidcConfigOptions(r, flags, isSTS, isHostedCP)
		if err != nil {
			return nil, err
		}
		err = validateOperatorRolesAvailabilityUnderUserAwsAccount(awsClient, computedOperatorIamRoleList)
		if err != nil {
			if !oidcConfig.Reusable() {
				return nil, fmt.Errorf("%v", err)
			} else {
				err = ocm.ValidateOperatorRolesMatchOidcProvider(r.Reporter, awsClient, computedOperatorIamRoleList,
					oidcConfig.IssuerUrl(), ocm.GetVersionMinor(version), expectedOperatorRolePath, managedPolicies)
				if err != nil {
					return nil, fmt.Errorf("%v", err)
				}
			}
		}
	}

	// Custom tags for AWS resources
	_tags := o.tags
	tagsList := map[string]string{}
	if interactive.Enabled() {
		tagsInput, err := interactive.GetString(interactive.Input{
			Question: "Tags",
			Help:     flags.Lookup("tags").Usage,
			Default:  strings.Join(_tags, ","),
			Validators: []interactive.Validator{
				aws.UserTagValidator,
				aws.UserTagDuplicateValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid set of tags: %w", err)
		}
		if len(tagsInput) > 0 {
			_tags = strings.Split(tagsInput, ",")
		}
	}
	if len(_tags) > 0 {
		if err := aws.UserTagValidator(_tags); err != nil {
			return nil, fmt.Errorf("%s", err)
		}
		delim := aws.GetTagsDelimiter(_tags)
		for _, tag := range _tags {
			t := strings.Split(tag, delim)
			tagsList[t[0]] = strings.TrimSpace(t[1])
		}
	}

	// Multi-AZ:
	multiAZ := o.multiAZ
	if interactive.Enabled() && !isHostedCP {
		multiAZ, err = interactive.GetBool(interactive.Input{
			Question: "Multiple availability zones",
			Help:     flags.Lookup("multi-az").Usage,
			Default:  multiAZ,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid multi-AZ value: %w", err)
		}
	}

	// Hosted clusters will be multiAZ by definition
	if isHostedCP {
		multiAZ = true
		if flags.Changed("multi-az") {
			r.Reporter.Warnf("Hosted clusters deprecate the --multi-az flag. " +
				"The hosted control plane will be MultiAZ, machinepools will be created in the different private " +
				"subnets provided under --subnet-ids flag.")
		}
	}

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		return nil, fmt.Errorf("error getting region: %v", err)
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
		return nil, fmt.Errorf(fmt.Sprintf("%s", err))
	}
	if region == "" {
		return nil, fmt.Errorf("expected a valid AWS region")
	} else if found := helper.Contains(regionList, region); isHostedCP && !shardPinningEnabled && !found {
		r.Reporter.Warnf("Region '%s' not currently available for Hosted Control Plane cluster.", region)
		interactive.Enable()
	}

	if interactive.Enabled() {
		region, err = interactive.GetOption(interactive.Input{
			Question: "AWS region",
			Help:     flags.Lookup("region").Usage,
			Options:  regionList,
			Default:  region,
			Required: true,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid AWS region: %w", err)
		}
	}
	if supportsMultiAZ, found := regionAZ[region]; found {
		if !supportsMultiAZ && multiAZ {
			return nil, fmt.Errorf("region '%s' does not support multiple availability zones", region)
		}
	} else {
		return nil, fmt.Errorf("region '%s' is not supported for this AWS account", region)
	}

	awsClient, err = aws.NewClient().
		Region(region).
		Logger(r.Logger).
		UseLocalCredentials(o.useLocalCredentials).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create awsClient: %w", err)
	}
	r.AWSClient = awsClient

	// Cluster privacy:
	useExistingVPC := false
	private := o.private
	isPrivateHostedCP := isHostedCP && private // all private hosted clusters are private-link
	privateLink := o.privateLink || fedramp.Enabled() || isPrivateHostedCP

	privateLinkWarning := "Once the cluster is created, this option cannot be changed."
	if isSTS {
		privateLinkWarning = fmt.Sprintf("STS clusters can only be private if AWS PrivateLink is used. %s ",
			privateLinkWarning)
	}
	if interactive.Enabled() && !fedramp.Enabled() && !isPrivateHostedCP {
		privateLink, err = interactive.GetBool(interactive.Input{
			Question: "PrivateLink cluster",
			Help:     fmt.Sprintf("%s %s", flags.Lookup("private-link").Usage, privateLinkWarning),
			Default:  privateLink || (isSTS && o.private),
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid private-link value: %w", err)
		}
	} else if (privateLink || (isSTS && private)) && !fedramp.Enabled() && !isPrivateHostedCP {
		// do not prompt users for privatelink if it is private hosted cluster
		r.Reporter.Warnf("You are choosing to use AWS PrivateLink for your cluster. %s", privateLinkWarning)
		if !confirm.Confirm("use AWS PrivateLink for cluster '%s'", clusterName) {
			return nil, nil
		}
		privateLink = true
	}

	if privateLink {
		private = true
	} else if isSTS && private {
		return nil, fmt.Errorf("private STS clusters are only supported through AWS PrivateLink")
	} else if !isSTS {
		privateWarning := "You will not be able to access your cluster until " +
			"you edit network settings in your cloud provider."
		if interactive.Enabled() {
			private, err = interactive.GetBool(interactive.Input{
				Question: "Private cluster",
				Help:     fmt.Sprintf("%s %s", flags.Lookup("private").Usage, privateWarning),
				Default:  private,
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid private value: %w", err)
			}
		} else if private {
			r.Reporter.Warnf("You are choosing to make your cluster private. %s", privateWarning)
			if !confirm.Confirm("set cluster '%s' as private", clusterName) {
				return nil, nil
			}
		}
	}

	if isSTS && private && !privateLink {
		return nil, fmt.Errorf("private STS clusters are only supported through AWS PrivateLink")
	}

	if privateLink || isHostedCP {
		useExistingVPC = true
	}

	// cluster-wide proxy values set here as we need to know whather to skip the "Install
	// into an existing VPC" question
	enableProxy := false
	httpProxy := o.httpProxy
	httpsProxy := o.httpsProxy
	noProxySlice := o.noProxySlice
	additionalTrustBundleFile := o.additionalTrustBundleFile
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
		GetDefaultClusterFlavors(o.flavour)
	if dMachinecidr == nil || dPodcidr == nil || dServicecidr == nil {
		return nil, fmt.Errorf("error retrieving default cluster flavors")
	}

	// Machine CIDR:
	machineCIDR := o.machineCIDR
	if ocm.IsEmptyCIDR(machineCIDR) {
		machineCIDR = *dMachinecidr
	}
	if interactive.Enabled() {
		machineCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Machine CIDR",
			Help:     flags.Lookup("machine-cidr").Usage,
			Default:  machineCIDR,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid CIDR value: %w", err)
		}
	}

	// Service CIDR:
	serviceCIDR := o.serviceCIDR
	if ocm.IsEmptyCIDR(serviceCIDR) {
		serviceCIDR = *dServicecidr
	}
	if interactive.Enabled() {
		serviceCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Service CIDR",
			Help:     flags.Lookup("service-cidr").Usage,
			Default:  serviceCIDR,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid CIDR value: %w", err)
		}
	}
	// Pod CIDR:
	podCIDR := o.podCIDR
	if ocm.IsEmptyCIDR(podCIDR) {
		podCIDR = *dPodcidr
	}
	if interactive.Enabled() {
		podCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Pod CIDR",
			Help:     flags.Lookup("pod-cidr").Usage,
			Default:  podCIDR,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid CIDR value: %w", err)
		}
	}

	// Subnet IDs
	subnetIDs := o.subnetIDs
	subnetsProvided := len(subnetIDs) > 0
	r.Reporter.Debugf("Received the following subnetIDs: %v", o.subnetIDs)
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
			return nil, fmt.Errorf("expected a valid value: %w", err)
		}
	}

	if isHostedCP && !subnetsProvided && !useExistingVPC {
		return nil, fmt.Errorf("all hosted clusters need a pre-configured VPC. Make sure to specify the subnet ids")
	}

	// For hosted cluster we will need the number of the private subnets the users has selected
	privateSubnetsCount := 0

	var availabilityZones []string
	var subnets []*ec2.Subnet
	mapSubnetIDToSubnet := make(map[string]aws.Subnet)
	if useExistingVPC || subnetsProvided {
		initialSubnets, err := getInitialValidSubnets(awsClient, o.subnetIDs, r.Reporter)
		if err != nil {
			return nil, fmt.Errorf("failed to get the list of subnets: %w", err)
		}
		if subnetsProvided {
			useExistingVPC = true
		}
		_, machineNetwork, err := net.ParseCIDR(machineCIDR.String())
		if err != nil {
			return nil, fmt.Errorf("unable to parse machine CIDR")
		}
		_, serviceNetwork, err := net.ParseCIDR(serviceCIDR.String())
		if err != nil {
			return nil, fmt.Errorf("unable to parse service CIDR")
		}
		subnets, err = filterCidrRangeSubnets(initialSubnets, machineNetwork, serviceNetwork, r)
		if err != nil {
			return nil, err
		}
		if privateLink {
			subnets, err = filterPrivateSubnets(subnets, r)
			if err != nil {
				return nil, err
			}
		}
		if len(subnets) == 0 {
			r.Reporter.Warnf("No subnets found in current region that are valid for the chosen CIDR ranges")
			if isHostedCP {
				return nil, fmt.Errorf(
					"all Hosted Control Plane clusters need a pre-configured VPC. Please check: %s",
					createVpcForHcpDoc,
				)
			}
			if ok := confirm.Prompt(false, "Continue with default? A new RH Managed VPC will be created for your cluster"); !ok {
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
					return nil, fmt.Errorf("could not find the following subnet provided in region '%s': %s",
						r.AWSClient.GetRegion(), subnetArg)
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
				Help:     flags.Lookup("subnet-ids").Usage,
				Required: false,
				Options:  options,
				Default:  defaultOptions,
				Validators: []interactive.Validator{
					interactive.SubnetsCountValidator(multiAZ, privateLink, isHostedCP),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected valid subnet IDs: %w", err)
			}
			for i, subnet := range subnetIDs {
				subnetIDs[i] = aws.ParseOption(subnet)
			}
		}

		// Validate subnets in the case the user has provided them using the `o.Subnets`
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
				return nil, fmt.Errorf("%s", err)
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
	privateHostedZoneID := strings.Trim(o.privateHostedZoneID, " \t")
	sharedVPCRoleARN := strings.Trim(o.sharedVPCRoleARN, " \t")
	baseDomain := strings.Trim(o.baseDomain, " \t")
	if privateHostedZoneID != "" ||
		sharedVPCRoleARN != "" {
		isSharedVPC = true
	}

	if len(subnetIDs) == 0 && isSharedVPC {
		return nil, fmt.Errorf("installing a cluster into a shared VPC is only supported for BYO VPC clusters")
	}

	if isSubnetBelongToSharedVpc(r, awsCreator.AccountID, subnetIDs, mapSubnetIDToSubnet) {
		isSharedVPC = true
		if privateHostedZoneID == "" || sharedVPCRoleARN == "" || baseDomain == "" {
			if !interactive.Enabled() {
				interactive.Enable()
			}

			privateHostedZoneID, err = getPrivateHostedZoneID(flags, privateHostedZoneID)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}

			sharedVPCRoleARN, err = getSharedVpcRoleArn(flags, sharedVPCRoleARN)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}

			baseDomain, err = getBaseDomain(r, flags, baseDomain)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}
		}
	}

	// Select availability zones for a non-BYOVPC cluster
	var selectAvailabilityZones bool
	if !useExistingVPC && !subnetsProvided {
		if isAvailabilityZonesSet {
			availabilityZones = o.availabilityZones
		}

		if !isAvailabilityZonesSet && interactive.Enabled() {
			selectAvailabilityZones, err = interactive.GetBool(interactive.Input{
				Question: "Select availability zones",
				Help:     flags.Lookup("availability-zones").Usage,
				Default:  false,
				Required: false,
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid value for select-availability-zones: %w", err)
			}

			if selectAvailabilityZones {
				optionsAvailabilityZones, err := awsClient.DescribeAvailabilityZones()
				if err != nil {
					return nil, fmt.Errorf("failed to get the list of the availability zone: %w", err)
				}

				availabilityZones, err = selectAvailabilityZonesInteractively(flags, optionsAvailabilityZones, multiAZ)
				if err != nil {
					return nil, fmt.Errorf("%s", err)
				}
			}
		}

		if isAvailabilityZonesSet || selectAvailabilityZones {
			err = validateAvailabilityZones(multiAZ, availabilityZones, awsClient)
			if err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("%s", err))
			}
		}
	}

	enableCustomerManagedKey := o.enableCustomerManagedKey
	kmsKeyARN := o.kmsKeyARN

	if kmsKeyARN != "" {
		enableCustomerManagedKey = true
	}
	if interactive.Enabled() && !enableCustomerManagedKey {
		enableCustomerManagedKey, err = interactive.GetBool(interactive.Input{
			Question: "Enable Customer Managed key",
			Help:     flags.Lookup("enable-customer-managed-key").Usage,
			Default:  enableCustomerManagedKey,
			Required: false,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value for enable-customer-managed-key: %w", err)
		}
	}

	if enableCustomerManagedKey && (kmsKeyARN == "" || interactive.Enabled()) {
		kmsKeyARN, err = interactive.GetString(interactive.Input{
			Question: "KMS Key ARN",
			Help:     flags.Lookup("kms-key-arn").Usage,
			Default:  kmsKeyARN,
			Required: enableCustomerManagedKey,
			Validators: []interactive.Validator{
				interactive.RegExp(kmsArnRegexpValidator.KmsArnRE.String()),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value for kms-key-arn: %w", err)
		}
	}

	err = kmsArnRegexpValidator.ValidateKMSKeyARN(&kmsKeyARN)
	if err != nil {
		return nil, fmt.Errorf("expected a valid value for kms-key-arn: %w", err)
	}

	// Compute node instance type:
	computeMachineType := o.computeMachineType
	computeMachineTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(region, availabilityZones, roleARN,
		awsClient)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("%s", err))
	}
	if computeMachineType == "" {
		computeMachineType = defaultComputeMachineType
	}
	if interactive.Enabled() {
		computeMachineType, err = interactive.GetOption(interactive.Input{
			Question: "Compute nodes instance type",
			Help:     flags.Lookup("compute-machine-type").Usage,
			Options:  computeMachineTypeList.GetAvailableIDs(multiAZ),
			Default:  computeMachineType,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid machine type: %w", err)
		}
	}
	err = computeMachineTypeList.ValidateMachineType(computeMachineType, multiAZ)
	if err != nil {
		return nil, fmt.Errorf("expected a valid machine type: %w", err)
	}

	isAutoscalingSet := flags.Changed("enable-autoscaling")
	isReplicasSet := flags.Changed("compute-nodes") || flags.Changed("replicas")

	// Autoscaling
	autoscaling := o.autoscalingEnabled
	if !isReplicasSet && !autoscaling && !isAutoscalingSet && interactive.Enabled() {
		autoscaling, err = interactive.GetBool(interactive.Input{
			Question: "Enable autoscaling",
			Help:     flags.Lookup("enable-autoscaling").Usage,
			Default:  autoscaling,
			Required: false,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value for enable-autoscaling: %w", err)
		}
	}

	isMinReplicasSet := flags.Changed("min-replicas")
	isMaxReplicasSet := flags.Changed("max-replicas")

	minReplicas, maxReplicas := calculateReplicas(
		isMinReplicasSet,
		isMaxReplicasSet,
		o.minReplicas,
		o.maxReplicas,
		privateSubnetsCount,
		isHostedCP,
		multiAZ)

	var clusterAutoscaler *clusterautoscaler.AutoscalerArgs
	if !autoscaling {
		clusterAutoscaler = nil
	} else {
		// if the user set compute-nodes and enabled autoscaling
		if isReplicasSet {
			return nil, fmt.Errorf("compute-nodes can't be set when autoscaling is enabled")
		}
		if interactive.Enabled() || !isMinReplicasSet {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     flags.Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: true,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid number of min replicas: %w", err)
			}
		}
		err = minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount)(minReplicas)
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}

		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     flags.Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					maxReplicaValidator(multiAZ, minReplicas, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid number of max replicas: %w", err)
			}
		}
		err = maxReplicaValidator(multiAZ, minReplicas, isHostedCP, privateSubnetsCount)(maxReplicas)
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}

		if isHostedCP {
			if clusterautoscaler.IsAutoscalerSetViaCLI(flags, clusterAutoscalerFlagsPrefix) {
				return nil, fmt.Errorf("hosted Control Plane clusters do not support cluster-autoscaler configuration")
			}
		} else {
			clusterAutoscaler, err = clusterautoscaler.GetAutoscalerOptions(
				flags, clusterAutoscalerFlagsPrefix, true, o.autoscalerArgs)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}
		}
	}

	// Compute nodes:
	computeNodes := o.computeNodes
	// Compute node requirements for multi-AZ clusters are higher
	if multiAZ && !autoscaling && !isReplicasSet {
		computeNodes = minReplicas
	}
	if !autoscaling {
		// if the user set min/max replicas and hasn't enabled autoscaling
		if isMinReplicasSet || isMaxReplicasSet {
			return nil, fmt.Errorf("autoscaling must be enabled in order to set min and max replicas")
		}

		if interactive.Enabled() {
			computeNodes, err = interactive.GetInt(interactive.Input{
				Question: "Compute nodes",
				Help:     flags.Lookup("compute-nodes").Usage,
				Default:  computeNodes,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid number of compute nodes: %w", err)
			}
		}
		err = minReplicaValidator(multiAZ, isHostedCP, privateSubnetsCount)(computeNodes)
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}
	}

	// Worker machine pool labels
	labels := o.defaultMachinePoolLabels
	if interactive.Enabled() && !isHostedCP {
		labels, err = interactive.GetString(interactive.Input{
			Question: "Worker machine pool labels",
			Help:     flags.Lookup(arguments.NewDefaultMPLabelsFlag).Usage,
			Default:  labels,
			Validators: []interactive.Validator{
				mpHelpers.LabelValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid comma-separated list of attributes: %w", err)
		}
	}
	labelMap, err := mpHelpers.ParseLabels(labels)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	isVersionCompatibleComputeSgIds, err := versions.IsGreaterThanOrEqual(
		version, ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay1)
	if err != nil {
		return nil, fmt.Errorf("there was a problem checking version compatibility: %v", err)
	}
	additionalComputeSecurityGroupIds := o.additionalComputeSecurityGroupIds
	if err := getSecurityGroups(r, flags, isVersionCompatibleComputeSgIds,
		securitygroups.ComputeKind, useExistingVPC, isHostedCP, subnets,
		subnetIDs, &additionalComputeSecurityGroupIds); err != nil {
		return nil, fmt.Errorf("could not get security groups for %s: %w", securitygroups.ComputeKind, err)
	}

	additionalInfraSecurityGroupIds := o.additionalInfraSecurityGroupIds
	if err := getSecurityGroups(r, flags, isVersionCompatibleComputeSgIds,
		securitygroups.InfraKind, useExistingVPC, isHostedCP, subnets,
		subnetIDs, &additionalInfraSecurityGroupIds); err != nil {
		return nil, fmt.Errorf("could not get security groups for %s: %w", securitygroups.InfraKind, err)
	}

	additionalControlPlaneSecurityGroupIds := o.additionalControlPlaneSecurityGroupIds
	if err := getSecurityGroups(r, flags, isVersionCompatibleComputeSgIds,
		securitygroups.ControlPlaneKind, useExistingVPC, isHostedCP, subnets,
		subnetIDs, &additionalControlPlaneSecurityGroupIds); err != nil {
		return nil, fmt.Errorf("could not get security groups for %s: %w", securitygroups.ControlPlaneKind, err)
	}

	// Validate all remaining flags:
	expiration, err := validateExpiration(o.expirationTime, o.expirationDuration)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("%s", err))
	}

	// Network Type:
	networkType, err := validateNetworkType(r, o.networkType)
	if err != nil {
		return nil, err
	}
	if flags.Changed("network-type") && interactive.Enabled() {
		networkType, err = interactive.GetOption(interactive.Input{
			Question: "Network Type",
			Help:     flags.Lookup("network-type").Usage,
			Options:  ocm.NetworkTypes,
			Default:  networkType,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid network type: %w", err)
		}
	}

	// Host prefix:
	hostPrefix := o.hostPrefix
	if interactive.Enabled() {
		if hostPrefix == 0 {
			hostPrefix = dhostPrefix
		}
		hostPrefix, err = interactive.GetInt(interactive.Input{
			Question: "Host prefix",
			Help:     flags.Lookup("host-prefix").Usage,
			Default:  hostPrefix,
			Validators: []interactive.Validator{
				hostPrefixValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid host prefix value: %w", err)
		}
	}
	err = hostPrefixValidator(hostPrefix)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	isVersionCompatibleMachinePoolRootDisk, err := versions.IsGreaterThanOrEqual(
		version, ocm.MinVersionForMachinePoolRootDisk)
	if err != nil {
		return nil, fmt.Errorf("there was a problem checking version compatibility: %v", err)
	}
	if !isVersionCompatibleMachinePoolRootDisk && flags.Changed(workerDiskSizeFlag) {
		formattedVersion, err := versions.FormatMajorMinorPatch(ocm.MinVersionForMachinePoolRootDisk)
		if err != nil {
			return nil, fmt.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
		}
		return nil, fmt.Errorf(
			"updating Worker disk size is not supported for versions prior to '%s'",
			formattedVersion,
		)
	}
	var machinePoolRootDisk *ocm.Volume
	if isVersionCompatibleMachinePoolRootDisk && !isHostedCP &&
		(o.machinePoolRootDiskSize != "" || interactive.Enabled()) {
		var machinePoolRootDiskSizeStr string
		if o.machinePoolRootDiskSize == "" {
			// We don't need to parse the default since it's returned from the OCM API and AWS
			// always defaults to GiB
			machinePoolRootDiskSizeStr = helper.GigybyteStringer(defaultMachinePoolRootDiskSize)
		} else {
			machinePoolRootDiskSizeStr = o.machinePoolRootDiskSize
		}
		if interactive.Enabled() {
			// In order to avoid confusion, we want to display to the user what was passed as an
			// argument
			// Even if it was not valid, we want to display it to the user, then the CLI will show an
			// error and the value can be corrected
			// Also, if nothing is given, we want to display the default value fetched from the OCM API
			machinePoolRootDiskSizeStr, err = interactive.GetString(interactive.Input{
				Question: "Machine pool root disk size (GiB or TiB)",
				Help:     flags.Lookup(workerDiskSizeFlag).Usage,
				Default:  machinePoolRootDiskSizeStr,
				Validators: []interactive.Validator{
					interactive.MachinePoolRootDiskSizeValidator(version),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid machine pool root disk size value: %v", err)
			}
		}

		// Parse the value given by either CLI or interactive mode and return it in GigiBytes
		machinePoolRootDiskSize, err := ocm.ParseDiskSizeToGigibyte(machinePoolRootDiskSizeStr)
		if err != nil {
			return nil, fmt.Errorf("expected a valid machine pool root disk size value: %v", err)
		}

		err = diskValidator.ValidateMachinePoolRootDiskSize(version, machinePoolRootDiskSize)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}

		// If the size given by the user is different than the default, we just let the OCM server
		// handle the default root disk size
		if machinePoolRootDiskSize != defaultMachinePoolRootDiskSize {
			machinePoolRootDisk = &ocm.Volume{
				Size: machinePoolRootDiskSize,
			}
		}
	}

	// No CNI
	if flags.Changed("no-cni") && !isHostedCP {
		return nil, fmt.Errorf("disabling CNI is supported only for Hosted Control Planes")
	}
	if flags.Changed("no-cni") && flags.Changed("network-type") {
		return nil, fmt.Errorf("--no-cni and --network-type are mutually exclusive parameters")
	}
	noCni := o.noCni
	if flags.Changed("no-cni") && interactive.Enabled() {
		noCni, err = interactive.GetBool(interactive.Input{
			Question: "Disable CNI",
			Help:     flags.Lookup("no-cni").Usage,
			Default:  noCni,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value for no CNI: %w", err)
		}
	}

	if flags.Changed("fips") && isHostedCP {
		return nil, fmt.Errorf("fIPS support not available for Hosted Control Plane clusters")
	}
	fips := o.fips || fedramp.Enabled()
	if interactive.Enabled() && !fedramp.Enabled() && !isHostedCP {
		fips, err = interactive.GetBool(interactive.Input{
			Question: "Enable FIPS support",
			Help:     flags.Lookup("fips").Usage,
			Default:  fips,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid FIPS value: %v", err)
		}
	}

	etcdEncryption := o.etcdEncryption

	// validate and force etcd encryption
	if etcdEncryptionKmsARN != "" {
		if flags.Changed("etcd-encryption") && !etcdEncryption {
			return nil, fmt.Errorf("etcd encryption cannot be disabled when encryption kms arn is provided")
		} else {
			etcdEncryption = true
		}
	}

	if interactive.Enabled() && !(fips || etcdEncryptionKmsARN != "") {
		etcdEncryption, err = interactive.GetBool(interactive.Input{
			Question: "Encrypt etcd data",
			Help:     flags.Lookup("etcd-encryption").Usage,
			Default:  etcdEncryption,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid etcd-encryption value: %v", err)
		}
	}
	if fips {
		if flags.Changed("etcd-encryption") && !etcdEncryption {
			return nil, fmt.Errorf("etcd encryption cannot be disabled on clusters with FIPS mode")
		} else {
			etcdEncryption = true
		}
	}

	if etcdEncryption && isHostedCP && (etcdEncryptionKmsARN == "" || interactive.Enabled()) {
		etcdEncryptionKmsARN, err = interactive.GetString(interactive.Input{
			Question: "Etcd encryption KMS ARN",
			Help:     flags.Lookup("etcd-encryption-kms-arn").Usage,
			Default:  etcdEncryptionKmsARN,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(kmsArnRegexpValidator.KmsArnRE.String()),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value for etcd-encryption-kms-arn: %w", err)
		}
	}

	err = kmsArnRegexpValidator.ValidateKMSKeyARN(&etcdEncryptionKmsARN)
	if err != nil {
		return nil, fmt.Errorf(
			"expected a valid value for etcd-encryption-kms-arn matching %s",
			kmsArnRegexpValidator.KmsArnRE,
		)
	}

	disableWorkloadMonitoring := o.disableWorkloadMonitoring
	if interactive.Enabled() {
		disableWorkloadMonitoring, err = interactive.GetBool(interactive.Input{
			Question: "Disable Workload monitoring",
			Help:     flags.Lookup("disable-workload-monitoring").Usage,
			Default:  disableWorkloadMonitoring,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid disable-workload-monitoring value: %v", err)
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
			return nil, fmt.Errorf("expected a valid proxy-enabled value: %w", err)
		}
	}

	if enableProxy && interactive.Enabled() {
		httpProxy, err = interactive.GetString(interactive.Input{
			Question: "HTTP proxy",
			Help:     flags.Lookup("http-proxy").Usage,
			Default:  httpProxy,
			Validators: []interactive.Validator{
				ocm.ValidateHTTPProxy,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid http proxy: %w", err)
		}
	}
	err = ocm.ValidateHTTPProxy(httpProxy)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	if enableProxy && interactive.Enabled() {
		httpsProxy, err = interactive.GetString(interactive.Input{
			Question: "HTTPS proxy",
			Help:     flags.Lookup("https-proxy").Usage,
			Default:  httpsProxy,
			Validators: []interactive.Validator{
				interactive.IsURL,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid https proxy: %w", err)
		}
	}
	err = interactive.IsURL(httpsProxy)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	if enableProxy && interactive.Enabled() {
		noProxyInput, err := interactive.GetString(interactive.Input{
			Question: "No proxy",
			Help:     flags.Lookup("no-proxy").Usage,
			Default:  strings.Join(noProxySlice, ","),
			Validators: []interactive.Validator{
				aws.UserNoProxyValidator,
				aws.UserNoProxyDuplicateValidator,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid set of no proxy domains/CIDR's: %w", err)
		}
		noProxySlice = helper.HandleEmptyStringOnSlice(strings.Split(noProxyInput, ","))
	}

	if len(noProxySlice) > 0 {
		duplicate, found := aws.HasDuplicates(noProxySlice)
		if found {
			return nil, fmt.Errorf("invalid no-proxy list, duplicate key '%s' found", duplicate)
		}
		for _, domain := range noProxySlice {
			err := aws.UserNoProxyValidator(domain)
			if err != nil {
				return nil, fmt.Errorf("%s", err)
			}
		}
	}

	if httpProxy == "" && httpsProxy == "" && len(noProxySlice) > 0 {
		return nil, fmt.Errorf("expected at least one of the following: http-proxy, https-proxy")
	}

	if useExistingVPC && interactive.Enabled() {
		additionalTrustBundleFile, err = interactive.GetCert(interactive.Input{
			Question: "Additional trust bundle file path",
			Help:     flags.Lookup("additional-trust-bundle-file").Usage,
			Default:  additionalTrustBundleFile,
			Validators: []interactive.Validator{
				ocm.ValidateAdditionalTrustBundle,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid additional trust bundle file name: %w", err)
		}
	}
	err = ocm.ValidateAdditionalTrustBundle(additionalTrustBundleFile)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	// Get certificate contents
	var additionalTrustBundle *string
	if additionalTrustBundleFile != "" {
		cert, err := os.ReadFile(additionalTrustBundleFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read additional trust bundle file: %w", err)
		}
		additionalTrustBundle = new(string)
		*additionalTrustBundle = string(cert)
	}

	if enableProxy && httpProxy == "" && httpsProxy == "" && additionalTrustBundleFile == "" {
		return nil, fmt.Errorf("expected at least one of the following: http-proxy, https-proxy, additional-trust-bundle")
	}

	// Audit Log Forwarding
	auditLogRoleARN := o.auditLogRoleARN

	if auditLogRoleARN != "" && !isHostedCP {
		return nil, fmt.Errorf("audit log forwarding to AWS CloudWatch is only supported for Hosted Control Plane clusters")
	}

	if interactive.Enabled() && isHostedCP {
		requestAuditLogForwarding, err := interactive.GetBool(interactive.Input{
			Question: "Enable audit log forwarding to AWS CloudWatch",
			Default:  false,
			Required: true,
		})
		if err != nil {
			return nil, fmt.Errorf("expected a valid value: %w", err)
		}
		if requestAuditLogForwarding {

			r.Reporter.Infof("To configure the audit log forwarding role in your AWS account, " +
				"please refer to steps 1 through 6: https://access.redhat.com/solutions/7002219")

			auditLogRoleARN, err = interactive.GetString(interactive.Input{
				Question: "Audit log forwarding role ARN",
				Help:     flags.Lookup("audit-log-arn").Usage,
				Default:  auditLogRoleARN,
				Required: true,
				Validators: []interactive.Validator{
					interactive.RegExp(aws.RoleArnRE.String()),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid value for audit-log-arn: %w", err)
			}
		} else {
			auditLogRoleARN = ""
		}
	}

	if auditLogRoleARN != "" && !aws.RoleArnRE.MatchString(auditLogRoleARN) {
		return nil, fmt.Errorf("expected a valid value for audit log arn matching %s", aws.RoleArnRE)
	}

	isVersionCompatibleManagedIngressV2, err := versions.IsGreaterThanOrEqual(
		version, ocm.MinVersionForManagedIngressV2)
	if err != nil {
		return nil, fmt.Errorf("there was a problem checking version compatibility: %v", err)
	}
	if ingress.IsDefaultIngressSetViaCLI(flags) {
		if isHostedCP {
			return nil, fmt.Errorf("updating default ingress settings is not supported for Hosted Control Plane clusters")
		}
		if !isVersionCompatibleManagedIngressV2 {
			formattedVersion, err := versions.FormatMajorMinorPatch(ocm.MinVersionForManagedIngressV2)
			if err != nil {
				return nil, fmt.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
			}
			return nil, fmt.Errorf(
				"updating default ingress settings is not supported for versions prior to '%s'",
				formattedVersion,
			)
		}
	}
	routeSelector := ""
	routeSelectors := map[string]string{}
	excludedNamespaces := ""
	var sliceExcludedNamespaces []string
	wildcardPolicy := ""
	namespaceOwnershipPolicy := ""
	if isVersionCompatibleManagedIngressV2 {
		shouldAskCustomIngress := false
		if interactive.Enabled() && !confirm.Yes() && !isHostedCP {
			shouldAskCustomIngress = confirm.Prompt(false, "Customize the default Ingress Controller?")
		}
		if flags.Changed(ingress.DefaultIngressRouteSelectorFlag) {
			if isHostedCP {
				return nil, fmt.Errorf("updating route selectors is not supported for Hosted Control Plane clusters")
			}
			routeSelector = o.defaultIngressRouteSelectors
		} else if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
			routeSelectorArg, err := interactive.GetString(interactive.Input{
				Question: "Router Ingress Sharding: Route Selector (e.g. 'route=external')",
				Help:     flags.Lookup(ingress.DefaultIngressRouteSelectorFlag).Usage,
				Default:  o.defaultIngressRouteSelectors,
				Validators: []interactive.Validator{
					func(routeSelector interface{}) error {
						_, err := ingress.GetRouteSelector(routeSelector.(string))
						return err
					},
				},
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid comma-separated list of attributes: %w", err)
			}
			routeSelector = routeSelectorArg
		}
		routeSelectors, err = ingress.GetRouteSelector(routeSelector)
		if err != nil {
			return nil, fmt.Errorf("%s", err)
		}

		if flags.Changed(ingress.DefaultIngressExcludedNamespacesFlag) {
			if isHostedCP {
				return nil, fmt.Errorf("updating excluded namespace is not supported for Hosted Control Plane clusters")
			}
			excludedNamespaces = o.defaultIngressExcludedNamespaces
		} else if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
			excludedNamespacesArg, err := interactive.GetString(interactive.Input{
				Question: "Router Ingress Sharding: Namespace exclusion",
				Help:     flags.Lookup(ingress.DefaultIngressExcludedNamespacesFlag).Usage,
				Default:  o.defaultIngressExcludedNamespaces,
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid comma-separated list of attributes: %w", err)
			}
			excludedNamespaces = excludedNamespacesArg
		}
		sliceExcludedNamespaces = ingress.GetExcludedNamespaces(excludedNamespaces)

		if flags.Changed(ingress.DefaultIngressWildcardPolicyFlag) {
			if isHostedCP {
				return nil, fmt.Errorf("updating Wildcard Policy is not supported for Hosted Control Plane clusters")
			}
			wildcardPolicy = o.defaultIngressWildcardPolicy
		} else {
			if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
				defaultIngressWildcardSelection := string(v1.WildcardPolicyWildcardsDisallowed)
				if o.defaultIngressWildcardPolicy != "" {
					defaultIngressWildcardSelection = o.defaultIngressWildcardPolicy
				}
				wildcardPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Route Admission: Wildcard Policy",
					Options:  ingress.ValidWildcardPolicies,
					Help:     flags.Lookup(ingress.DefaultIngressWildcardPolicyFlag).Usage,
					Default:  defaultIngressWildcardSelection,
					Required: true,
				})
				if err != nil {
					return nil, fmt.Errorf("expected a valid Wildcard Policy: %w", err)
				}
				wildcardPolicy = wildcardPolicyArg
			}
		}

		if flags.Changed(ingress.DefaultIngressNamespaceOwnershipPolicyFlag) {
			if isHostedCP {
				return nil, fmt.Errorf(
					"updating Namespace Ownership Policy is not supported for Hosted Control Plane clusters",
				)
			}
			namespaceOwnershipPolicy = o.defaultIngressNamespaceOwnershipPolicy
		} else {
			if interactive.Enabled() && !isHostedCP && shouldAskCustomIngress {
				defaultIngressNamespaceOwnershipSelection := string(v1.NamespaceOwnershipPolicyStrict)
				if o.defaultIngressNamespaceOwnershipPolicy != "" {
					defaultIngressNamespaceOwnershipSelection = o.defaultIngressNamespaceOwnershipPolicy
				}
				namespaceOwnershipPolicyArg, err := interactive.GetOption(interactive.Input{
					Question: "Route Admission: Namespace Ownership Policy",
					Options:  ingress.ValidNamespaceOwnershipPolicies,
					Help:     flags.Lookup(ingress.DefaultIngressNamespaceOwnershipPolicyFlag).Usage,
					Default:  defaultIngressNamespaceOwnershipSelection,
					Required: true,
				})
				if err != nil {
					return nil, fmt.Errorf("expected a valid Namespace Ownership Policy: %w", err)
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
		Flavour:                   o.flavour,
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
		DryRun:                    &o.dryRun,
		DisableSCPChecks:          &o.disableSCPChecks,
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
			return nil, fmt.Errorf("failed creating autoscaler configuration: %w", err)
		}

		clusterConfig.AutoscalerConfig = autoscalerConfig
	}

	props := o.properties
	if o.fakeCluster {
		props = append(props, properties.FakeCluster)
	}
	if o.useLocalCredentials {
		if isSTS {
			return nil, fmt.Errorf("local credentials are not supported for STS clusters")
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

	return &CompletedOptions{
		completedOptions: &completedOptions{
			OperatorRolesPrefix:      operatorRolesPrefix,
			ExpectedOperatorRolePath: expectedOperatorRolePath,
			IsAvailabilityZonesSet:   isAvailabilityZonesSet,
			SelectAvailabilityZones:  selectAvailabilityZones,
			Labels:                   labels,
			Properties:               o.properties,
			ClusterAdminPassword:     clusterAdminPassword,
			ClassicOidcConfig:        o.classicOidcConfig,
			ExpirationDuration:       o.expirationDuration,

			OperatorRoles:       operatorRoles,
			PermissionsBoundary: permissionsBoundary,
			IsSTS:               isSTS,
			DryRun:              o.dryRun,
			Watch:               o.watch,
			ClusterName:         clusterName,
			AWSMode:             mode,
			Spec:                clusterConfig,
		},
	}, nil
}

func (o *CompletedOptions) Validate() error {
	var errs []error
	// TODO: move the rest of the validation logic here
	return errors.NewAggregate(errs)
}

type completedOptions struct {
	IsSTS                    bool
	PermissionsBoundary      string
	OperatorRoles            []string
	OperatorRolesPrefix      string
	ExpectedOperatorRolePath string
	IsAvailabilityZonesSet   bool
	SelectAvailabilityZones  bool
	Labels                   string
	Properties               []string
	ClusterAdminPassword     string
	ClassicOidcConfig        bool

	AWSMode     string
	ClusterName string
	Spec        ocm.Spec

	Watch              bool
	DryRun             bool
	ExpirationDuration time.Duration
}

type CompletedOptions struct {
	*completedOptions
}
