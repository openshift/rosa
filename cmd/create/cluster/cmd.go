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
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	clusterdescribe "github.com/openshift/rosa/cmd/describe/cluster"
	installLogs "github.com/openshift/rosa/cmd/logs/install"

	"github.com/spf13/cobra"

	awssdk "github.com/aws/aws-sdk-go/aws"
	v "github.com/openshift/rosa/cmd/validations"
	"github.com/openshift/rosa/pkg/aws"

	clusterprovider "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/confirm"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/ocm/machines"
	"github.com/openshift/rosa/pkg/ocm/regions"
	"github.com/openshift/rosa/pkg/ocm/versions"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	// Watch logs during cluster installation
	watch bool

	// Simulate creating a cluster
	dryRun bool

	// Disable SCP checks in the installer
	disableSCPChecks bool

	// Basic options
	private            bool
	multiAZ            bool
	expirationDuration time.Duration
	expirationTime     string
	clusterName        string
	region             string
	version            string
	channelGroup       string

	// Scaling options
	computeMachineType string
	computeNodes       int
	autoscalingEnabled bool
	minReplicas        int
	maxReplicas        int

	// Networking options
	hostPrefix  int
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet

	// The Subnet IDs to use when installing the cluster.
	// SubnetIDs should come in pairs; two per availability zone, one private and one public.
	subnetIDs []string
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Example: `  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2`,
	Run:              run,
	PersistentPreRun: v.Validations,
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
		&args.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	flags.StringVarP(
		&args.region,
		"region",
		"r",
		"",
		"AWS region where your worker pool will be located. (overrides the AWS_REGION environment variable)",
	)
	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to install the cluster, for example \"4.3.10\"",
	)
	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		versions.DefaultChannelGroup,
		"Channel group is the name of the group where this image belongs, for example \"stable\" or \"fast\".",
	)
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

	flags.StringSliceVar(
		&args.subnetIDs,
		"subnet-ids",
		nil,
		"The Subnet IDs to use when installing the cluster. "+
			"SubnetIDs should come in pairs; two per availability zone, one private and one public. "+
			"Subnets are comma separated, for example: --subnet-ids=subnet-1,subnet-2."+
			"Leave empty for installer provisioned subnet IDs.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	var err error

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()
	ocmClient := ocmConnection.ClustersMgmt().V1()

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
		})
		if err != nil {
			reporter.Errorf("Expected a valid cluster name: %s", err)
			os.Exit(1)
		}
	}

	// Trim names to remove any leading/trailing invisible characters
	clusterName = strings.Trim(clusterName, " \t")

	if !clusterprovider.IsValidClusterName(clusterName) {
		reporter.Errorf("Cluster name must consist" +
			" of no more than 15 lowercase alphanumeric characters or '-', " +
			"start with a letter, and end with an alphanumeric character.")
		os.Exit(1)
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
	region, err := aws.GetRegion(args.region)
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	regionList, regionAZ, err := regions.GetRegionList(ocmClient, multiAZ)
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

	// OpenShift version:
	version := args.version
	channelGroup := args.channelGroup
	versionList, err := getVersionList(ocmClient, channelGroup)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
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
	version, err = validateVersion(version, versionList)
	if err != nil {
		reporter.Errorf("Expected a valid OpenShift version: %s", err)
		os.Exit(1)
	}

	// Subnet IDs
	awsClient, err := aws.NewClient().
		Region(region).
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create awsClient: %s", err)
		os.Exit(1)
	}

	subnetIDs := args.subnetIDs
	subnetsProvided := len(subnetIDs) > 0
	useExistingVPC := false
	reporter.Debugf("Received the following subnetIDs: %v", args.subnetIDs)
	if !subnetsProvided && interactive.Enabled() {
		useExistingVPC, err = interactive.GetBool(interactive.Input{
			Question: "Install into an existing VPC",
			Help: "To install into an existing VPC you need to ensure that your VPC is configured " +
				"with a public and a private subnet for each availability zone that you want the " +
				"cluster installed into.",
			Default: useExistingVPC,
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
		if interactive.Enabled() && len(options) > 0 && (!multiAZ || len(mapAZCreated) >= 3) {
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

	// Compute node instance type:
	computeMachineType := args.computeMachineType
	computeMachineTypeList, err := machines.GetMachineTypeList(ocmClient)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	if interactive.Enabled() {
		computeMachineType, err = interactive.GetOption(interactive.Input{
			Question: "Compute nodes instance type",
			Help:     cmd.Flags().Lookup("compute-machine-type").Usage,
			Options:  computeMachineTypeList,
			Default:  computeMachineType,
		})
		if err != nil {
			reporter.Errorf("Expected a valid machine type: %s", err)
			os.Exit(1)
		}
	}
	computeMachineType, err = machines.ValidateMachineType(computeMachineType, computeMachineTypeList)
	if err != nil {
		reporter.Errorf("Expected a valid machine type: %s", err)
		os.Exit(1)
	}

	// Autoscaling
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	autoscaling := args.autoscalingEnabled
	if !autoscaling && !isAutoscalingSet && interactive.Enabled() {
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
		if cmd.Flags().Changed("compute-nodes") {
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
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}

		if multiAZ && minReplicas < 3 {
			reporter.Errorf("Multi AZ cluster requires at least 3 compute nodes")
			os.Exit(1)
		}
		if !multiAZ && minReplicas < 2 {
			reporter.Errorf("Cluster requires at least 2 compute nodes")
			os.Exit(1)
		}

		if minReplicas > maxReplicas {
			reporter.Errorf("max-replicas must be greater or equal to min-replicas")
			os.Exit(1)
		}

		if multiAZ && (minReplicas%3 != 0 || maxReplicas%3 != 0) {
			reporter.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
			os.Exit(1)
		}
	}

	// Compute nodes:
	computeNodes := args.computeNodes
	// Compute node requirements for multi-AZ clusters are higher
	if multiAZ && !autoscaling && !cmd.Flags().Changed("compute-nodes") {
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
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of compute nodes: %s", err)
				os.Exit(1)
			}
		}
		if multiAZ {
			if computeNodes < 3 {
				reporter.Errorf("The number of compute nodes needs to be at least 3")
				os.Exit(1)
			}
			if computeNodes%3 != 0 {
				reporter.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
				os.Exit(1)
			}
		} else {
			if computeNodes < 2 {
				reporter.Errorf("The number of compute nodes needs to be at least 2")
				os.Exit(1)
			}
		}
	}

	// Validate all remaining flags:
	expiration, err := validateExpiration()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	var dMachinecidr *net.IPNet
	var dPodcidr *net.IPNet
	var dServicecidr *net.IPNet
	dMachinecidr, dPodcidr, dServicecidr, dhostPrefix := ocm.GetDefaultClusterFlavors(ocmClient)

	// Machine CIDR:
	machineCIDR := args.machineCIDR
	if interactive.Enabled() {
		machineCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Machine CIDR",
			Help:     cmd.Flags().Lookup("machine-cidr").Usage,
			Default:  *dMachinecidr,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}

	// Service CIDR:
	serviceCIDR := args.serviceCIDR
	if interactive.Enabled() {
		serviceCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Service CIDR",
			Help:     cmd.Flags().Lookup("service-cidr").Usage,
			Default:  *dServicecidr,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}
	// Pod CIDR:
	podCIDR := args.podCIDR
	if interactive.Enabled() {
		podCIDR, err = interactive.GetIPNet(interactive.Input{
			Question: "Pod CIDR",
			Help:     cmd.Flags().Lookup("pod-cidr").Usage,
			Default:  *dPodcidr,
		})
		if err != nil {
			reporter.Errorf("Expected a valid CIDR value: %s", err)
			os.Exit(1)
		}
	}

	// Host prefix:
	hostPrefix := args.hostPrefix
	if interactive.Enabled() {
		hostPrefix, err = interactive.GetInt(interactive.Input{
			Question: "Host prefix",
			Help:     cmd.Flags().Lookup("host-prefix").Usage,
			Default:  dhostPrefix,
		})
		if err != nil {
			reporter.Errorf("Expected a valid host prefix value: %s", err)
			os.Exit(1)
		}
	}

	// Cluster privacy:
	private := args.private
	privateWarning := "You will not be able to access your cluster until you edit network settings in your cloud provider."
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

	clusterConfig := clusterprovider.Spec{
		Name:               clusterName,
		Region:             region,
		MultiAZ:            multiAZ,
		Version:            version,
		ChannelGroup:       channelGroup,
		Expiration:         expiration,
		ComputeMachineType: computeMachineType,
		ComputeNodes:       computeNodes,
		Autoscaling:        autoscaling,
		MinReplicas:        minReplicas,
		MaxReplicas:        maxReplicas,
		MachineCIDR:        machineCIDR,
		ServiceCIDR:        serviceCIDR,
		PodCIDR:            podCIDR,
		HostPrefix:         hostPrefix,
		Private:            &private,
		DryRun:             &args.dryRun,
		DisableSCPChecks:   &args.disableSCPChecks,
		AvailabilityZones:  availabilityZones,
		SubnetIds:          subnetIDs,
	}

	reporter.Infof("Creating cluster '%s'", clusterName)
	reporter.Infof("To view a list of clusters and their status, run 'rosa list clusters'")

	cluster, err := clusterprovider.CreateCluster(ocmClient.Clusters(), clusterConfig)
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

	reporter.Infof("Cluster '%s' has been created.", clusterName)
	reporter.Infof(
		"Once the cluster is installed you will need to add an Identity Provider " +
			"before you can login into the cluster. See 'rosa create idp --help' " +
			"for more information.")

	if args.watch {
		installLogs.Cmd.Run(cmd, []string{cluster.ID()})
	} else {
		reporter.Infof(
			"To determine when your cluster is Ready, run 'rosa describe cluster -c %s'.",
			clusterName,
		)
		reporter.Infof(
			"To watch your cluster installation logs, run 'rosa logs install -c %s --watch'.",
			clusterName,
		)
	}

	clusterdescribe.Cmd.Run(cmd, []string{cluster.ID()})
}

// Validate OpenShift versions
func validateVersion(version string, versionList []string) (string, error) {
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

		version = "openshift-v" + version
	}

	return version, nil
}

func getVersionList(client *cmv1.Client, channelGroup string) (versionList []string, err error) {
	versions, err := versions.GetVersions(client, channelGroup)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	for _, v := range versions {
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
