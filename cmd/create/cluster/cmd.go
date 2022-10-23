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
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/cmd/create/operatorroles"
	clusterdescribe "github.com/openshift/rosa/cmd/describe/cluster"
	installLogs "github.com/openshift/rosa/cmd/logs/install"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

//nolint
var kmsArnRE = regexp.MustCompile(`^arn:aws:kms:[\w-]+:\d{12}:key\/[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var args struct {
	// Watch logs during cluster installation
	watch bool

	// Simulate creating a cluster
	dryRun bool
	// Create a fake cluster with no AWS resources
	fakeCluster bool
	// Set custom properties in cluster spec
	properties []string

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

	//Encryption
	etcdEncryption           bool
	fips                     bool
	enableCustomerManagedKey bool
	kmsKeyARN                string
	// Scaling options
	computeMachineType string
	computeNodes       int
	autoscalingEnabled bool
	minReplicas        int
	maxReplicas        int

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

	// Force STS mode for interactive and validation
	sts bool

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

	// Proxy
	enableProxy               bool
	httpProxy                 string
	httpsProxy                string
	additionalTrustBundleFile string

	tags []string
}

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

	flags.StringSliceVar(
		&args.tags,
		"tags",
		nil,
		"Apply user defined tags to all resources created by ROSA in AWS."+
			"Tags are comma separated, for example: --tags=foo:bar,bar:baz",
	)
	flags.MarkHidden("tags")

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
	flags.MarkHidden("fips")

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

	flags.StringSliceVar(
		&args.subnetIDs,
		"subnet-ids",
		nil,
		"The Subnet IDs to use when installing the cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for installer provisioned subnet IDs.",
	)

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
		"Number of worker nodes to provision per zone. Single zone clusters need at least 2 nodes, "+
			"multizone clusters need at least 3 nodes.",
	)

	flags.BoolVar(
		&args.autoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling of compute nodes.",
	)

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
		&args.networkType,
		"network-type",
		ocm.NetworkTypes[0],
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

	flags.BoolVar(
		&args.watch,
		"watch",
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

	flags.StringVar(
		&args.operatorRolesPermissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the operator "+
			"roles in STS clusters.",
	)

	aws.AddModeFlag(Cmd)
	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
}

func networkTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return ocm.NetworkTypes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	var err error

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	awsClient := aws.GetAWSClientForUserRegion(reporter, logger)

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Unable to get IAM credentials: %v", err)
		os.Exit(1)
	}

	if interactive.Enabled() {
		reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Get cluster name
	clusterName := strings.Trim(args.clusterName, " \t")

	if clusterName == "" && !interactive.Enabled() {
		interactive.Enable()
		reporter.Infof("Enabling interactive mode")
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
			reporter.Errorf("Expected a valid cluster name: %s", err)
			os.Exit(1)
		}
	}

	// Trim names to remove any leading/trailing invisible characters
	clusterName = strings.Trim(clusterName, " \t")

	if !ocm.IsValidClusterName(clusterName) {
		reporter.Errorf("Cluster name must consist" +
			" of no more than 15 lowercase alphanumeric characters or '-', " +
			"start with a letter, and end with an alphanumeric character.")
		os.Exit(1)
	}

	isSTS := args.sts || args.roleARN != ""
	isIAM := cmd.Flags().Changed("sts") && !isSTS

	if interactive.Enabled() && (!isSTS && !isIAM) {
		isSTS, err = interactive.GetBool(interactive.Input{
			Question: "Deploy cluster using AWS STS",
			Help:     cmd.Flags().Lookup("sts").Usage,
			Default:  true,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid --sts value: %s", err)
			os.Exit(1)
		}
		isIAM = !isSTS
	}

	permissionsBoundary := args.operatorRolesPermissionsBoundary
	if permissionsBoundary != "" {
		_, err := arn.Parse(permissionsBoundary)
		if err != nil {
			reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	if isIAM {
		if awsCreator.IsSTS {
			reporter.Errorf("Since your AWS credentials are returning an STS ARN you can only " +
				"create STS clusters. Otherwise, switch to IAM credentials.")
			os.Exit(1)
		}
		err := awsClient.CheckAdminUserExists(aws.AdminUserName)
		if err != nil {
			reporter.Errorf("IAM user '%s' does not exist. Run `rosa init` first", aws.AdminUserName)
			os.Exit(1)
		}
		reporter.Debugf("IAM user is valid!")
	}

	// AWS ARN Role
	roleARN := args.roleARN
	supportRoleARN := args.supportRoleARN
	controlPlaneRoleARN := args.controlPlaneRoleARN
	workerRoleARN := args.workerRoleARN

	isSTS = isSTS || awsCreator.IsSTS

	// OpenShift version:
	version := args.version
	channelGroup := args.channelGroup
	versionList, err := getVersionList(ocmClient, channelGroup, isSTS)
	if err != nil {
		reporter.Errorf("%s", err)
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
			reporter.Errorf("Expected a valid OpenShift version: %s", err)
			os.Exit(1)
		}
	}
	version, err = validateVersion(version, versionList, channelGroup, isSTS)
	if err != nil {
		reporter.Errorf("Expected a valid OpenShift version: %s", err)
		os.Exit(1)
	}

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// warn if mode is used for non sts cluster
	if !isSTS && mode != "" {
		reporter.Warnf("--mode is only valid for STS clusters")
	}

	// validate mode passed is allowed value

	if isSTS && mode != "" {
		isValidMode := arguments.IsValidMode(aws.Modes, mode)
		if !isValidMode {
			reporter.Errorf("Invalid --mode '%s'. Allowed values are %s", mode, aws.Modes)
			os.Exit(1)
		}
	}

	hasRoles := false
	if isSTS && roleARN == "" {
		minor := ocm.GetVersionMinor(version)
		role := aws.AccountRoles[aws.InstallerAccountRole]

		// Find all installer roles in the current account using AWS resource tags
		roleARNs, err := awsClient.FindRoleARNs(aws.InstallerAccountRole, minor)
		if err != nil {
			reporter.Errorf("Failed to find %s role: %s", role.Name, err)
			os.Exit(1)
		}

		if len(roleARNs) > 1 {
			defaultRoleARN := roleARNs[0]
			// Prioritize roles with the default prefix
			for _, rARN := range roleARNs {
				if strings.Contains(rARN, fmt.Sprintf("%s-%s-Role", aws.DefaultPrefix, role.Name)) {
					defaultRoleARN = rARN
				}
			}
			reporter.Warnf("More than one %s role found", role.Name)
			roleARN, err = interactive.GetOption(interactive.Input{
				Question: fmt.Sprintf("%s role ARN", role.Name),
				Help:     cmd.Flags().Lookup(role.Flag).Usage,
				Options:  roleARNs,
				Default:  defaultRoleARN,
				Required: true,
			})
			if err != nil {
				reporter.Errorf("Expected a valid role ARN: %s", err)
				os.Exit(1)
			}
		} else if len(roleARNs) == 1 {
			if !output.HasFlag() || reporter.IsTerminal() {
				reporter.Infof("Using %s for the %s role", roleARNs[0], role.Name)
			}
			roleARN = roleARNs[0]
		} else {
			reporter.Warnf("No account roles found. You will need to manually set them in the " +
				"next steps or run 'rosa create account-roles' to create them first.")
			interactive.Enable()
		}

		if roleARN != "" {
			// Get role prefix
			rolePrefix, err := getAccountRolePrefix(roleARN, role)
			if err != nil {
				reporter.Errorf("Failed to find prefix from %s account role", role.Name)
				os.Exit(1)
			}
			reporter.Debugf("Using '%s' as the role prefix", rolePrefix)

			hasRoles = true
			for roleType, role := range aws.AccountRoles {
				if roleType == aws.InstallerAccountRole {
					// Already dealt with
					continue
				}
				roleARNs, err := awsClient.FindRoleARNs(roleType, minor)
				if err != nil {
					reporter.Errorf("Failed to find %s role: %s", role.Name, err)
					os.Exit(1)
				}
				selectedARN := ""
				for _, rARN := range roleARNs {
					if strings.Contains(rARN, fmt.Sprintf("%s-%s-Role", rolePrefix, role.Name)) {
						selectedARN = rARN
					}
				}
				if selectedARN == "" {
					reporter.Warnf("No %s account roles found. You will need to manually set "+
						"them in the next steps or run 'rosa create account-roles' to create "+
						"them first.", role.Name)
					interactive.Enable()
					hasRoles = false
					break
				}
				if !output.HasFlag() || reporter.IsTerminal() {
					reporter.Infof("Using %s for the %s role", selectedARN, role.Name)
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
			reporter.Errorf("Expected a valid ARN: %s", err)
			os.Exit(1)
		}
	}
	if roleARN != "" {
		_, err = arn.Parse(roleARN)
		if err != nil {
			reporter.Errorf("Expected a valid Role ARN: %s", err)
			os.Exit(1)
		}
		isSTS = true
	}

	if !isSTS && mode != "" {
		reporter.Warnf("--mode is only valid for STS clusters")
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
			reporter.Errorf("Expected a valid External ID: %s", err)
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
			reporter.Errorf("Expected a valid ARN: %s", err)
			os.Exit(1)
		}
	}
	if supportRoleARN != "" {
		_, err = arn.Parse(supportRoleARN)
		if err != nil {
			reporter.Errorf("Expected a valid Support Role ARN: %s", err)
			os.Exit(1)
		}
	} else if roleARN != "" {
		reporter.Errorf("Support Role ARN is required: %s", err)
		os.Exit(1)
	}

	// Instance IAM Roles
	// Ensure interactive mode if missing required role ARNs on STS clusters
	if isSTS && !hasRoles && !interactive.Enabled() {
		if controlPlaneRoleARN == "" || workerRoleARN == "" {
			interactive.Enable()
		}
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
			reporter.Errorf("Expected a valid control plane IAM role ARN: %s", err)
			os.Exit(1)
		}
	}
	if controlPlaneRoleARN != "" {
		_, err = arn.Parse(controlPlaneRoleARN)
		if err != nil {
			reporter.Errorf("Expected a valid control plane instance IAM role ARN: %s", err)
			os.Exit(1)
		}
	} else if roleARN != "" {
		reporter.Errorf("Control plane instance IAM role ARN is required: %s", err)
		os.Exit(1)
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
			reporter.Errorf("Expected a valid worker IAM role ARN: %s", err)
			os.Exit(1)
		}
	}
	if workerRoleARN != "" {
		_, err = arn.Parse(workerRoleARN)
		if err != nil {
			reporter.Errorf("Expected a valid worker instance IAM role ARN: %s", err)
			os.Exit(1)
		}
	} else if roleARN != "" {
		reporter.Errorf("Worker instance IAM role ARN is required: %s", err)
		os.Exit(1)
	}

	// combine role arns to list
	roleARNs := []string{
		roleARN,
		supportRoleARN,
		controlPlaneRoleARN,
		workerRoleARN,
	}

	// iterate and validate role arns against openshift version
	for _, ARN := range roleARNs {
		if ARN == "" {
			continue
		}
		// get role from arn
		role, err := awsClient.GetRoleByARN(ARN)
		if err != nil {
			reporter.Errorf("Could not get Role '%s' : %v", ARN, err)
			os.Exit(1)
		}

		validVersion, err := awsClient.HasCompatibleVersionTags(role.Tags, ocm.GetVersionMinor(version))
		if err != nil {
			reporter.Errorf("Could not validate Role '%s' : %v", ARN, err)
			os.Exit(1)
		}
		if !validVersion {
			reporter.Errorf("Account role '%s' is not compatible with version %s. "+
				"Run 'rosa create account-roles' to create compatible roles and try again.",
				ARN, version)
			os.Exit(1)
		}
	}

	operatorRolesPrefix := args.operatorRolesPrefix
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
				reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
				os.Exit(1)
			}
		}
		if len(operatorRolesPrefix) == 0 {
			reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
			os.Exit(1)
		}
		if len(operatorRolesPrefix) > 32 {
			reporter.Errorf("Expected a prefix with no more than 32 characters")
			os.Exit(1)
		}
		if !aws.RoleNameRE.MatchString(operatorRolesPrefix) {
			reporter.Errorf("Expected valid operator roles prefix matching %s", aws.RoleNameRE.String())
			os.Exit(1)
		}
	}

	operatorIAMRoles := args.operatorIAMRoles
	operatorIAMRoleList := []ocm.OperatorIAMRole{}
	if isSTS {
		for _, operator := range aws.CredentialRequests {
			//If the cluster version is less than the supported operator version
			if operator.MinVersion != "" {
				isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(version), operator.MinVersion)
				if err != nil {
					reporter.Errorf("Error validating operator role '%s' version %s", operator.Name, err)
					os.Exit(1)
				}
				if !isSupported {
					continue
				}
			}
			operatorIAMRoleList = append(operatorIAMRoleList, ocm.OperatorIAMRole{
				Name:      operator.Name,
				Namespace: operator.Namespace,
				RoleARN:   getOperatorRoleArn(operatorRolesPrefix, operator, awsCreator),
			})

		}
		// If user insists on using the deprecated --operator-iam-roles
		// override the values to support the legacy documentation
		if cmd.Flags().Changed("operator-iam-roles") {
			operatorIAMRoleList = []ocm.OperatorIAMRole{}
			for _, role := range operatorIAMRoles {
				if !strings.Contains(role, ",") {
					reporter.Errorf("Expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
					os.Exit(1)
				}
				roleData := strings.Split(role, ",")
				if len(roleData) != 3 {
					reporter.Errorf("Expected operator IAM roles to be a comma-separated " +
						"list of name,namespace,role_arn")
					os.Exit(1)
				}
				operatorIAMRoleList = append(operatorIAMRoleList, ocm.OperatorIAMRole{
					Name:      roleData[0],
					Namespace: roleData[1],
					RoleARN:   roleData[2],
				})
			}
		}
		// Validate the role names are available on AWS
		for _, role := range operatorIAMRoleList {
			name := strings.SplitN(role.RoleARN, "/", 2)[1]
			err := awsClient.ValidateRoleNameAvailable(name)
			if err != nil {
				reporter.Errorf("Error validating role: %v", err)
				os.Exit(1)
			}
		}
	}

	// Custom tags for AWS resources
	tags := args.tags
	tagsList := map[string]string{}
	if len(tags) > 0 && interactive.Enabled() {
		tagsInput, err := interactive.GetString(interactive.Input{
			Question: "Tags",
			Help:     cmd.Flags().Lookup("tags").Usage,
			Default:  strings.Join(tags, ","),
			Validators: []interactive.Validator{
				aws.UserTagValidator,
				aws.UserTagDuplicateValidator,
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid set of tags: %s", err)
			os.Exit(1)
		}
		tags = strings.Split(tagsInput, ",")
	}
	if len(tags) > 0 {
		duplicate, found := aws.HasDuplicateTagKey(tags)
		if found {
			reporter.Errorf("Invalid tags, user tag keys must be unique, duplicate key '%s' found", duplicate)
			os.Exit(1)
		}
		for _, tag := range tags {
			err := aws.UserTagValidator(tag)
			if err != nil {
				reporter.Errorf("%s", err)
				os.Exit(1)
			}
			t := strings.Split(tag, ":")
			tagsList[t[0]] = strings.TrimSpace(t[1])
		}
	}

	// Multi-AZ:
	multiAZ := args.multiAZ
	if interactive.Enabled() {
		multiAZ, err = interactive.GetBool(interactive.Input{
			Question: "Multiple availability zones",
			Help:     cmd.Flags().Lookup("multi-az").Usage,
			Default:  multiAZ,
		})
		if err != nil {
			reporter.Errorf("Expected a valid multi-AZ value: %s", err)
			os.Exit(1)
		}
	}

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	regionList, regionAZ, err := ocmClient.GetRegionList(multiAZ, roleARN, externalID)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
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
			reporter.Errorf("Expected a valid AWS region: %s", err)
			os.Exit(1)
		}
	}

	if region == "" {
		reporter.Errorf("Expected a valid AWS region")
		os.Exit(1)
	} else {
		if supportsMultiAZ, found := regionAZ[region]; found {
			if !supportsMultiAZ && multiAZ {
				reporter.Errorf("Region '%s' does not support multiple availability zones", region)
				os.Exit(1)
			}
		} else {
			reporter.Errorf("Region '%s' is not supported for this AWS account", region)
			os.Exit(1)
		}
	}

	awsClient, err = aws.NewClient().
		Region(region).
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create awsClient: %s", err)
		os.Exit(1)
	}

	// Cluster privacy:
	useExistingVPC := false
	privateLink := args.privateLink
	private := args.private

	privateLinkWarning := "Once the cluster is created, this option cannot be changed."
	if isSTS {
		privateLinkWarning = fmt.Sprintf("STS clusters can only be private if AWS PrivateLink is used. %s ",
			privateLinkWarning)
	}
	if interactive.Enabled() {
		privateLink, err = interactive.GetBool(interactive.Input{
			Question: "PrivateLink cluster",
			Help:     fmt.Sprintf("%s %s", cmd.Flags().Lookup("private-link").Usage, privateLinkWarning),
			Default:  privateLink || (isSTS && args.private),
		})
		if err != nil {
			reporter.Errorf("Expected a valid private-link value: %s", err)
			os.Exit(1)
		}
	} else if privateLink || (isSTS && private) {
		reporter.Warnf("You are choosing to use AWS PrivateLink for your cluster. %s", privateLinkWarning)
		if !confirm.Confirm("use AWS PrivateLink for cluster '%s'", clusterName) {
			os.Exit(0)
		}
		privateLink = true
	}

	if privateLink {
		private = true
	} else if isSTS && private {
		reporter.Errorf("Private STS clusters are only supported through AWS PrivateLink")
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
				reporter.Errorf("Expected a valid private value: %s", err)
				os.Exit(1)
			}
		} else if private {
			reporter.Warnf("You are choosing to make your cluster private. %s", privateWarning)
			if !confirm.Confirm("set cluster '%s' as private", clusterName) {
				os.Exit(0)
			}
		}
	}

	if isSTS && private && !privateLink {
		reporter.Errorf("Private STS clusters are only supported through AWS PrivateLink")
		os.Exit(1)
	}

	if privateLink {
		useExistingVPC = true
	}

	// cluster-wide proxy values set here as we need to know whather to skip the "Install
	// into an existing VPC" question
	enableProxy := false
	httpProxy := args.httpProxy
	httpsProxy := args.httpsProxy
	additionalTrustBundleFile := args.additionalTrustBundleFile
	if httpProxy != "" || httpsProxy != "" || additionalTrustBundleFile != "" {
		useExistingVPC = true
		enableProxy = true
	}

	// Subnet IDs
	subnetIDs := args.subnetIDs
	subnetsProvided := len(subnetIDs) > 0
	reporter.Debugf("Received the following subnetIDs: %v", args.subnetIDs)
	if !useExistingVPC && !subnetsProvided && interactive.Enabled() {
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
			reporter.Errorf("Expected a valid value: %s", err)
			os.Exit(1)
		}
	}

	var availabilityZones []string
	if useExistingVPC || subnetsProvided {
		subnets, err := awsClient.GetSubnetIDs()
		if err != nil {
			reporter.Errorf("Failed to get the list of subnets: %s", err)
			os.Exit(1)
		}

		mapSubnetToAZ := make(map[string]string)
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
					reporter.Errorf("Could not find the following subnet provided: %s", subnetArg)
					os.Exit(1)
				}
			}
		}

		for i, subnet := range subnets {
			subnetID := awssdk.StringValue(subnet.SubnetId)
			availabilityZone := awssdk.StringValue(subnet.AvailabilityZone)

			// Create the options to prompt the user.
			options[i] = setSubnetOption(subnetID, availabilityZone)
			if subnetsProvided {
				for _, subnetArg := range subnetIDs {
					defaultOptions = append(defaultOptions, setSubnetOption(subnetArg, availabilityZone))
				}
			}
			mapSubnetToAZ[subnetID] = availabilityZone
			mapAZCreated[availabilityZone] = false
		}
		if ((privateLink && !subnetsProvided) || interactive.Enabled()) &&
			len(options) > 0 && (!multiAZ || len(mapAZCreated) >= 3) {
			subnetIDs, err = interactive.GetMultipleOptions(interactive.Input{
				Question: "Subnet IDs",
				Help:     cmd.Flags().Lookup("subnet-ids").Usage,
				Required: false,
				Options:  options,
				Default:  defaultOptions,
			})
			if err != nil {
				reporter.Errorf("Expected valid subnet IDs: %s", err)
				os.Exit(1)
			}
			for i, subnet := range subnetIDs {
				subnetIDs[i] = parseSubnet(subnet)
			}
		}

		for _, subnet := range subnetIDs {
			az := mapSubnetToAZ[subnet]
			if !mapAZCreated[az] {
				availabilityZones = append(availabilityZones, az)
				mapAZCreated[az] = true
			}
		}
	}
	reporter.Debugf("Found the following availability zones for the subnets provided: %v", availabilityZones)

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
			reporter.Errorf("Expected a valid value for enable-customer-managed-key: %s", err)
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
			reporter.Errorf("Expected a valid value for kms-key-arn: %s", err)
			os.Exit(1)
		}
	}

	if kmsKeyARN != "" && !kmsArnRE.MatchString(kmsKeyARN) {
		reporter.Errorf("Expected a valid value for kms-key-arn matching %s", kmsArnRE)
		os.Exit(1)
	}

	// Compute node instance type:
	computeMachineType := args.computeMachineType
	computeMachineTypeList, err := ocmClient.GetAvailableMachineTypes()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	if interactive.Enabled() {
		computeMachineType, err = interactive.GetOption(interactive.Input{
			Question: "Compute nodes instance type",
			Help:     cmd.Flags().Lookup("compute-machine-type").Usage,
			Options:  computeMachineTypeList.GetAvailableIDs(multiAZ),
			Default:  computeMachineType,
		})
		if err != nil {
			reporter.Errorf("Expected a valid machine type: %s", err)
			os.Exit(1)
		}
	}
	err = computeMachineTypeList.ValidateMachineType(computeMachineType, multiAZ)
	if err != nil {
		reporter.Errorf("Expected a valid machine type: %s", err)
		os.Exit(1)
	}

	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isReplicasSet := cmd.Flags().Changed("compute-nodes")

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
			reporter.Errorf("Expected a valid value for enable-autoscaling: %s", err)
			os.Exit(1)
		}
	}

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")

	minReplicas := args.minReplicas
	maxReplicas := args.maxReplicas
	if autoscaling {
		// if the user set compute-nodes and enabled autoscaling
		if isReplicasSet {
			reporter.Errorf("Compute-nodes can't be set when autoscaling is enabled")
			os.Exit(1)
		}

		if multiAZ {
			if !isMinReplicasSet {
				minReplicas = 3
			}
			if !isMaxReplicasSet {
				maxReplicas = minReplicas
			}
		}
		if interactive.Enabled() || !isMinReplicasSet {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: true,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ),
				},
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZ)(minReplicas)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					maxReplicaValidator(multiAZ, minReplicas),
				},
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
		err = maxReplicaValidator(multiAZ, minReplicas)(maxReplicas)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	// Compute nodes:
	computeNodes := args.computeNodes
	// Compute node requirements for multi-AZ clusters are higher
	if multiAZ && !autoscaling && !isReplicasSet {
		computeNodes = 3
	}
	if !autoscaling {
		// if the user set min/max replicas and hasn't enabled autoscaling
		if isMinReplicasSet || isMaxReplicasSet {
			reporter.Errorf("Autoscaling must be enabled in order to set min and max replicas")
			os.Exit(1)
		}

		if interactive.Enabled() {
			computeNodes, err = interactive.GetInt(interactive.Input{
				Question: "Compute nodes",
				Help:     cmd.Flags().Lookup("compute-nodes").Usage,
				Default:  computeNodes,
				Validators: []interactive.Validator{
					minReplicaValidator(multiAZ),
				},
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of compute nodes: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZ)(computeNodes)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	// Validate all remaining flags:
	expiration, err := validateExpiration()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
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
			reporter.Errorf("Expected a valid network type: %s", err)
			os.Exit(1)
		}
	}

	dMachinecidr, dPodcidr, dServicecidr, dhostPrefix := ocmClient.GetDefaultClusterFlavors(args.flavour)
	if dMachinecidr == nil || dPodcidr == nil || dServicecidr == nil {
		reporter.Errorf("Error retrieving default cluster flavors")
		os.Exit(1)
	}

	// Machine CIDR:
	machineCIDR := args.machineCIDR
	if interactive.Enabled() {
		if ocm.IsEmptyCIDR(machineCIDR) {
			machineCIDR = *dMachinecidr
		}
		machineCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Machine CIDR",
			Help:     cmd.Flags().Lookup("machine-cidr").Usage,
			Default:  machineCIDR,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}

	// Service CIDR:
	serviceCIDR := args.serviceCIDR
	if interactive.Enabled() {
		if ocm.IsEmptyCIDR(serviceCIDR) {
			serviceCIDR = *dServicecidr
		}
		serviceCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Service CIDR",
			Help:     cmd.Flags().Lookup("service-cidr").Usage,
			Default:  serviceCIDR,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}
	// Pod CIDR:
	podCIDR := args.podCIDR
	if interactive.Enabled() {
		if ocm.IsEmptyCIDR(podCIDR) {
			podCIDR = *dPodcidr
		}
		podCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Pod CIDR",
			Help:     cmd.Flags().Lookup("pod-cidr").Usage,
			Default:  podCIDR,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
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
			reporter.Errorf("Expected a valid host prefix value: %s", err)
			os.Exit(1)
		}
	}
	err = hostPrefixValidator(hostPrefix)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	fips := args.fips
	if interactive.Enabled() && fips {
		fips, err = interactive.GetBool(interactive.Input{
			Question: "Enable FIPS support",
			Help:     cmd.Flags().Lookup("fips").Usage,
			Default:  fips,
		})
		if err != nil {
			reporter.Errorf("Expected a valid fips value: %v", err)
			os.Exit(1)
		}
	}

	etcdEncryption := args.etcdEncryption
	if interactive.Enabled() && !fips {
		etcdEncryption, err = interactive.GetBool(interactive.Input{
			Question: "Encrypt etcd data",
			Help:     cmd.Flags().Lookup("etcd-encryption").Usage,
			Default:  etcdEncryption,
		})
		if err != nil {
			reporter.Errorf("Expected a valid etcd-encryption value: %v", err)
			os.Exit(1)
		}
	}
	if fips {
		if cmd.Flags().Changed("etcd-encryption") && !etcdEncryption {
			reporter.Errorf("etcd encryption cannot be disabled on clusters with FIPS mode")
			os.Exit(1)
		} else {
			etcdEncryption = true
		}
	}

	disableWorkloadMonitoring := args.disableWorkloadMonitoring
	if interactive.Enabled() {
		disableWorkloadMonitoring, err = interactive.GetBool(interactive.Input{
			Question: "Disable Workload monitoring",
			Help:     cmd.Flags().Lookup("disable-workload-monitoring").Usage,
			Default:  disableWorkloadMonitoring,
		})
		if err != nil {
			reporter.Errorf("Expected a valid disable-workload-monitoring value: %v", err)
			os.Exit(1)
		}
	}

	// Cluster-wide proxy configuration
	if useExistingVPC && !enableProxy && interactive.Enabled() {
		enableProxy, err = interactive.GetBool(interactive.Input{
			Question: "Use cluster-wide proxy",
			Help: "To install cluster-wide proxy, you need to set one of the following attributes: 'http-proxy', " +
				"'https-proxy', additional-trust-bundle",
			Default: enableProxy,
		})
		if err != nil {
			reporter.Errorf("Expected a valid proxy-enabled value: %s", err)
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
			reporter.Errorf("Expected a valid http proxy: %s", err)
			os.Exit(1)
		}
	}
	err = ocm.ValidateHTTPProxy(httpProxy)
	if err != nil {
		reporter.Errorf("%s", err)
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
			reporter.Errorf("Expected a valid https proxy: %s", err)
			os.Exit(1)
		}
	}
	err = interactive.IsURL(httpsProxy)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if enableProxy && interactive.Enabled() {
		additionalTrustBundleFile, err = interactive.GetCert(interactive.Input{
			Question: "Additional trust bundle file path",
			Help:     cmd.Flags().Lookup("additional-trust-bundle-file").Usage,
			Default:  additionalTrustBundleFile,
			Validators: []interactive.Validator{
				ocm.ValidateAdditionalTrustBundle,
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid additional trust bundle file name: %s", err)
			os.Exit(1)
		}
	}
	err = ocm.ValidateAdditionalTrustBundle(additionalTrustBundleFile)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Get certificate contents
	var additionalTrustBundle *string
	if additionalTrustBundleFile != "" {
		cert, err := ioutil.ReadFile(additionalTrustBundleFile)
		if err != nil {
			reporter.Errorf("Failed to read additional trust bundle file: %s", err)
			os.Exit(1)
		}
		additionalTrustBundle = new(string)
		*additionalTrustBundle = string(cert)
	}

	if enableProxy && httpProxy == "" && httpsProxy == "" && additionalTrustBundleFile == "" {
		reporter.Errorf("Expected at least one of the following: http-proxy, https-proxy, additional-trust-bundle")
		os.Exit(1)
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
		EnableProxy:               enableProxy,
		AdditionalTrustBundle:     additionalTrustBundle,
		Expiration:                expiration,
		ComputeMachineType:        computeMachineType,
		ComputeNodes:              computeNodes,
		Autoscaling:               autoscaling,
		MinReplicas:               minReplicas,
		MaxReplicas:               maxReplicas,
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
		OperatorIAMRoles:          operatorIAMRoleList,
		ControlPlaneRoleARN:       controlPlaneRoleARN,
		WorkerRoleARN:             workerRoleARN,
		Mode:                      mode,
		Tags:                      tagsList,
		KMSKeyArn:                 kmsKeyARN,
		DisableWorkloadMonitoring: &disableWorkloadMonitoring,
	}

	if httpProxy != "" {
		clusterConfig.HTTPProxy = &httpProxy
	}
	if httpsProxy != "" {
		clusterConfig.HTTPSProxy = &httpsProxy
	}
	if additionalTrustBundleFile != "" {
		clusterConfig.AdditionalTrustBundleFile = &additionalTrustBundleFile
		clusterConfig.AdditionalTrustBundle = additionalTrustBundle
	}

	props := args.properties
	if args.fakeCluster {
		props = append(props, properties.FakeCluster)
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

	if !output.HasFlag() || reporter.IsTerminal() {
		reporter.Infof("Creating cluster '%s'", clusterName)
		if interactive.Enabled() {
			command := buildCommand(clusterConfig, operatorRolesPrefix)
			reporter.Infof("To create this cluster again in the future, you can run:\n   %s", command)
		}
		reporter.Infof("To view a list of clusters and their status, run 'rosa list clusters'")
	}

	_, err = ocmClient.CreateCluster(clusterConfig)
	if err != nil {
		if args.dryRun {
			reporter.Errorf("Creating cluster '%s' should fail: %s", clusterName, err)
		} else {
			reporter.Errorf("Failed to create cluster: %s", err)
		}
		os.Exit(1)
	}

	if args.dryRun {
		reporter.Infof(
			"Creating cluster '%s' should succeed. Run without the '--dry-run' flag to create the cluster.",
			clusterName)
		os.Exit(0)
	}

	if !output.HasFlag() || reporter.IsTerminal() {
		reporter.Infof("Cluster '%s' has been created.", clusterName)
		reporter.Infof(
			"Once the cluster is installed you will need to add an Identity Provider " +
				"before you can login into the cluster. See 'rosa create idp --help' " +
				"for more information.")
	}

	if args.watch {
		installLogs.Cmd.Run(installLogs.Cmd, []string{clusterName})
	} else if !output.HasFlag() || reporter.IsTerminal() {
		reporter.Infof(
			"To determine when your cluster is Ready, run 'rosa describe cluster -c %s'.",
			clusterName,
		)
		reporter.Infof(
			"To watch your cluster installation logs, run 'rosa logs install -c %s --watch'.",
			clusterName,
		)
	}

	clusterdescribe.Cmd.Run(clusterdescribe.Cmd, []string{clusterName})

	if isSTS {
		if mode != "" {
			reporter.Infof("Preparing to create operator roles.")
			operatorroles.Cmd.Run(operatorroles.Cmd, []string{clusterName, mode, permissionsBoundary})
			reporter.Infof("Preparing to create OIDC Provider.")
			oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{clusterName, mode})
		} else {
			rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", clusterName)
			oidcCMD := fmt.Sprintf("rosa create oidc-provider --cluster %s", clusterName)

			if permissionsBoundary != "" {
				rolesCMD = fmt.Sprintf("%s --permissions-boundary %s", rolesCMD, permissionsBoundary)
			}

			reporter.Infof("Run the following commands to continue the cluster creation:\n\n"+
				"\t%s\n"+
				"\t%s\n",
				rolesCMD, oidcCMD)
		}
	}
}

func minReplicaValidator(multiAZ bool) interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if multiAZ {
			if minReplicas < 3 {
				return fmt.Errorf("Multi AZ cluster requires at least 3 compute nodes")
			}
			if minReplicas%3 != 0 {
				return fmt.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
			}
		} else if minReplicas < 2 {
			return fmt.Errorf("Cluster requires at least 2 compute nodes")
		}
		return nil
	}
}

func maxReplicaValidator(multiAZ bool, minReplicas int) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas > maxReplicas {
			return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
		}
		if multiAZ && maxReplicas%3 != 0 {
			return fmt.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
		}
		return nil
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

func getOperatorRoleArn(prefix string, operator aws.Operator, creator *aws.Creator) string {
	role := fmt.Sprintf("%s-%s-%s", prefix, operator.Namespace, operator.Name)
	if len(role) > 64 {
		role = role[0:64]
	}
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", creator.AccountID, role)
}

func getAccountRolePrefix(roleARN string, role aws.AccountRole) (string, error) {
	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return "", err
	}
	roleName := strings.SplitN(parsedARN.Resource, "/", 2)[1]
	rolePrefix := strings.TrimSuffix(roleName, fmt.Sprintf("-%s-Role", role.Name))
	return rolePrefix, nil
}

// Validate OpenShift versions
func validateVersion(version string, versionList []string, channelGroup string, isSTS bool) (string, error) {
	if version != "" {
		// Check and set the cluster version
		hasVersion := false
		for _, v := range versionList {
			if v == version {
				hasVersion = true
			}
		}
		if !hasVersion {
			allVersions := strings.Join(versionList, " ")
			err := fmt.Errorf("A valid version number must be specified\nValid versions: %s", allVersions)
			return version, err
		}

		if isSTS && !ocm.HasSTSSupport(version, channelGroup) {
			err := fmt.Errorf("Version '%s' is not supported for STS clusters", version)
			return version, err
		}

		version = "openshift-v" + version
	}

	return version, nil
}

func getVersionList(ocmClient *ocm.Client, channelGroup string, isSTS bool) (versionList []string, err error) {
	vs, err := ocmClient.GetVersions(channelGroup)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	for _, v := range vs {
		if isSTS && !ocm.HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		versionList = append(versionList, strings.Replace(v.ID(), "openshift-v", "", 1))
	}

	if len(versionList) == 0 {
		err = fmt.Errorf("Could not find versions for the provided channel-group: '%s'", channelGroup)
		return
	}

	return
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
		// round up to the nearest second
		expiration = time.Now().Add(args.expirationDuration).Round(time.Second)
	}

	return
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

const subnetTemplate = "%s (%s)"

// Creates a subnet options using a predefined template.
func setSubnetOption(subnet, zone string) string {
	return fmt.Sprintf(subnetTemplate, subnet, zone)
}

// Parses the subnet from the option chosen by the user.
func parseSubnet(subnetOption string) string {
	return strings.Split(subnetOption, " ")[0]
}

func buildCommand(spec ocm.Spec, operatorRolesPrefix string) string {
	command := "rosa create cluster"
	command += fmt.Sprintf(" --cluster-name %s", spec.Name)
	if spec.IsSTS {
		command += " --sts"
		if spec.Mode != "" {
			command += fmt.Sprintf(" --mode %s", spec.Mode)
		}
	}
	if spec.RoleARN != "" {
		command += fmt.Sprintf(" --role-arn %s", spec.RoleARN)
		command += fmt.Sprintf(" --support-role-arn %s", spec.SupportRoleARN)
		command += fmt.Sprintf(" --controlplane-iam-role %s", spec.ControlPlaneRoleARN)
		command += fmt.Sprintf(" --worker-iam-role %s", spec.WorkerRoleARN)
	}
	if spec.ExternalID != "" {
		command += fmt.Sprintf(" --external-id %s", spec.ExternalID)
	}
	if operatorRolesPrefix != "" {
		command += fmt.Sprintf(" --operator-roles-prefix %s", operatorRolesPrefix)
	}
	if len(spec.Tags) > 0 {
		tags := []string{}
		for k, v := range spec.Tags {
			tags = append(tags, fmt.Sprintf("%s:%s", k, v))
		}
		command += fmt.Sprintf(" --tags %s", strings.Join(tags, ","))
	}
	if spec.MultiAZ {
		command += " --multi-az"
	}
	if spec.Region != "" {
		command += fmt.Sprintf(" --region %s", spec.Region)
	}
	if spec.DisableSCPChecks != nil && *spec.DisableSCPChecks {
		command += " --disable-scp-checks"
	}
	if spec.Version != "" {
		if spec.ChannelGroup != ocm.DefaultChannelGroup {
			command += fmt.Sprintf(" --channel-group %s", spec.ChannelGroup)
		}
		command += fmt.Sprintf(" --version %s", strings.TrimPrefix(spec.Version, "openshift-v"))
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
			command += fmt.Sprintf(" --compute-nodes %d", spec.ComputeNodes)
		}
	}
	if spec.ComputeMachineType != "" {
		command += fmt.Sprintf(" --compute-machine-type %s", spec.ComputeMachineType)
	}

	if spec.NetworkType != ocm.NetworkTypes[0] {
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
	return command
}

func getRolePrefix(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, ocm.RandomLabel(4))
}
