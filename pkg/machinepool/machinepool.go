package machinepool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	diskValidator "github.com/openshift-online/ocm-common/pkg/machinepool/validations"
	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/helper/features"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	clusterAutoscalerLimitMessage = "Cluster Autoscaler limit (MaxNodes)"
)

var fetchMessage string = "Fetching %s '%s' for cluster '%s'"
var notFoundMessage string = "Machine pool '%s' not found"

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

type ListMachinePoolArgs struct {
	ShowAZType    bool
	ShowDedicated bool
	ShowWindowsLI bool
	ShowAll       bool
}

type MachinePoolService interface {
	DescribeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, machinePoolId string) error
	ListMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster, args ListMachinePoolArgs) error
	DeleteMachinePool(r *rosa.Runtime, machinePoolId string, clusterKey string, cluster *cmv1.Cluster) error
	EditMachinePool(cmd *cobra.Command, machinePoolID string, clusterKey string, cluster *cmv1.Cluster,
		r *rosa.Runtime) error
	CreateMachinePoolBasedOnClusterType(r *rosa.Runtime, cmd *cobra.Command,
		clusterKey string, cluster *cmv1.Cluster, clusterAutoscaler *cmv1.ClusterAutoscaler,
		options *mpOpts.CreateMachinepoolUserOptions) error
}

type machinePool struct {
}

var _ MachinePoolService = &machinePool{}

func NewMachinePoolService() MachinePoolService {
	return &machinePool{}
}

// Create machine pool based on cluster type
func (m machinePool) CreateMachinePoolBasedOnClusterType(r *rosa.Runtime,
	cmd *cobra.Command, clusterKey string, cluster *cmv1.Cluster, clusterAutoscaler *cmv1.ClusterAutoscaler,
	options *mpOpts.CreateMachinepoolUserOptions) error {
	if cluster.Hypershift().Enabled() {
		return m.CreateNodePools(r, cmd, clusterKey, cluster, clusterAutoscaler, options)
	}
	return m.CreateMachinePool(r, cmd, clusterKey, cluster, options)
}

func (m *machinePool) CreateMachinePool(r *rosa.Runtime, cmd *cobra.Command, clusterKey string, cluster *cmv1.Cluster,
	args *mpOpts.CreateMachinepoolUserOptions) error {

	// Validate flags that are only allowed for multi-AZ clusters
	isMultiAvailabilityZoneSet := cmd.Flags().Changed("multi-availability-zone")
	if isMultiAvailabilityZoneSet && !cluster.MultiAZ() {
		return fmt.Errorf("setting the `multi-availability-zone` flag is only allowed for multi-AZ clusters")
	}
	isAvailabilityZoneSet := cmd.Flags().Changed("availability-zone")
	if isAvailabilityZoneSet && !cluster.MultiAZ() {
		return fmt.Errorf("setting the `availability-zone` flag is only allowed for multi-AZ clusters")
	}

	// Validate flags that are only allowed for BYOVPC cluster
	isSubnetSet := cmd.Flags().Changed("subnet")
	isByoVpc := helper.IsBYOVPC(cluster)
	if !isByoVpc && isSubnetSet {
		return fmt.Errorf("setting the `subnet` flag is only allowed for BYO VPC clusters")
	}

	isSecurityGroupIdsSet := cmd.Flags().Changed(securitygroups.MachinePoolSecurityGroupFlag)
	isVersionCompatibleComputeSgIds, err := versions.IsGreaterThanOrEqual(
		cluster.Version().RawID(), ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay2)
	if err != nil {
		return fmt.Errorf("there was a problem checking version compatibility: %v", err)
	}
	if isSecurityGroupIdsSet {
		if !isByoVpc {
			return fmt.Errorf("setting the `%s` flag is only allowed for BYOVPC clusters",
				securitygroups.MachinePoolSecurityGroupFlag)
		}
		if !isVersionCompatibleComputeSgIds {
			formattedVersion, err := versions.FormatMajorMinorPatch(
				ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay2,
			)
			if err != nil {
				return fmt.Errorf("%s", fmt.Sprintf(versions.MajorMinorPatchFormattedErrorOutput, err))
			}
			return fmt.Errorf("parameter '%s' is not supported prior to version '%s'",
				securitygroups.MachinePoolSecurityGroupFlag, formattedVersion)
		}
	}

	if isSubnetSet && isAvailabilityZoneSet {
		return fmt.Errorf("setting both `subnet` and `availability-zone` flag is not supported." +
			" Please select `subnet` or `availability-zone` to create a single availability zone machine pool")
	}

	// Validate `subnet` or `availability-zone` flags are set for a single AZ machine pool
	if isAvailabilityZoneSet && isMultiAvailabilityZoneSet && args.MultiAvailabilityZone {
		return fmt.Errorf("setting the `availability-zone` flag is only supported for creating a single AZ " +
			"machine pool in a multi-AZ cluster")
	}
	if isSubnetSet && isMultiAvailabilityZoneSet && args.MultiAvailabilityZone {
		return fmt.Errorf("setting the `subnet` flag is only supported for creating a single AZ machine pool")
	}

	rosa.HostedClusterOnlyFlag(r, cmd, "version")
	rosa.HostedClusterOnlyFlag(r, cmd, "autorepair")
	rosa.HostedClusterOnlyFlag(r, cmd, "tuning-configs")
	rosa.HostedClusterOnlyFlag(r, cmd, "kubelet-configs")
	rosa.HostedClusterOnlyFlag(r, cmd, "ec2-metadata-http-tokens")

	// Machine pool name:
	name := strings.Trim(args.Name, " \t")
	if name == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}
	if name == "" || interactive.Enabled() {
		name, err = interactive.GetString(interactive.Input{
			Question: "Machine pool name",
			Default:  name,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(machinePoolKeyRE.String()),
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid name for the machine pool: %s", err)
		}
	}
	name = strings.Trim(name, " \t")
	if !machinePoolKeyRE.MatchString(name) {
		return fmt.Errorf("expected a valid name for the machine pool")
	}

	// Allow the user to select subnet for a single AZ BYOVPC cluster
	var subnet string
	if !cluster.MultiAZ() && isByoVpc {
		subnet, err = getSubnetFromUser(cmd, r, isSubnetSet, cluster, args)
		if err != nil {
			return err
		}
	}

	// Single AZ machine pool for a multi-AZ cluster
	var multiAZMachinePool bool
	var availabilityZone string
	if cluster.MultiAZ() {
		// Choosing a single AZ machine pool implicitly (providing availability zone or subnet)
		if isAvailabilityZoneSet || isSubnetSet {
			isMultiAvailabilityZoneSet = true
			args.MultiAvailabilityZone = false
		}

		if !isMultiAvailabilityZoneSet && interactive.Enabled() && !confirm.Yes() {
			multiAZMachinePool, err = interactive.GetBool(interactive.Input{
				Question: "Create multi-AZ machine pool",
				Help:     cmd.Flags().Lookup("multi-availability-zone").Usage,
				Default:  true,
				Required: false,
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for create multi-AZ machine pool")
			}
		} else {
			multiAZMachinePool = args.MultiAvailabilityZone
		}

		if !multiAZMachinePool {
			// Allow to create a single AZ machine pool providing the subnet
			if isByoVpc && args.AvailabilityZone == "" {
				subnet, err = getSubnetFromUser(cmd, r, isSubnetSet, cluster, args)
				if err != nil {
					return err
				}
			}

			// Select availability zone if the user didn't select subnet
			if subnet == "" {
				availabilityZone = cluster.Nodes().AvailabilityZones()[0]
				if !isAvailabilityZoneSet && interactive.Enabled() {
					availabilityZone, err = interactive.GetOption(interactive.Input{
						Question: "AWS availability zone",
						Help:     cmd.Flags().Lookup("availability-zone").Usage,
						Options:  cluster.Nodes().AvailabilityZones(),
						Default:  availabilityZone,
						Required: true,
					})
					if err != nil {
						return fmt.Errorf("expected a valid AWS availability zone: %s", err)
					}
				} else if isAvailabilityZoneSet {
					availabilityZone = args.AvailabilityZone
				}

				if !helper.Contains(cluster.Nodes().AvailabilityZones(), availabilityZone) {
					return fmt.Errorf("availability zone '%s' doesn't belong to the cluster's availability zones",
						availabilityZone)
				}
			}
		}
	}

	replicaSizeValidation := &ReplicaSizeValidation{
		ClusterVersion: cluster.OpenshiftVersion(),
		MultiAz:        multiAZMachinePool,
		IsHostedCp:     cluster.Hypershift().Enabled(),
	}
	minReplicas, maxReplicas, replicas, autoscaling, err := manageReplicas(cmd, args, replicaSizeValidation)
	if err != nil {
		return err
	}

	securityGroupIds := args.SecurityGroupIds
	if interactive.Enabled() && isVersionCompatibleComputeSgIds &&
		isByoVpc && !isSecurityGroupIdsSet {
		securityGroupIds, err = getSecurityGroupsOption(r, cmd, cluster)
		if err != nil {
			return err
		}
	}
	for i, sg := range securityGroupIds {
		securityGroupIds[i] = strings.TrimSpace(sg)
	}

	// Machine pool instance type:
	instanceType := args.InstanceType
	if instanceType == "" && !interactive.Enabled() {
		return fmt.Errorf("you must supply a valid instance type")
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() && !output.HasFlag() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Checking available instance types for machine pool '%s'", name)
		spin.Start()
	}

	// Determine machine pool availability zones to filter supported machine types
	availabilityZonesFilter, err := getMachinePoolAvailabilityZones(r, cluster, multiAZMachinePool, availabilityZone,
		subnet)
	if err != nil {
		return err
	}

	instanceTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(
		cluster.Region().ID(),
		availabilityZonesFilter,
		cluster.AWS().STS().RoleARN(),
		r.AWSClient,
		cluster.AWS().STS().ExternalID(),
	)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	if spin != nil {
		spin.Stop()
	}

	if interactive.Enabled() {
		if instanceType == "" {
			instanceType = instanceTypeList.Items[0].MachineType.ID()
		}
		instanceType, err = interactive.GetOption(interactive.Input{
			Question: "Instance type",
			Help:     cmd.Flags().Lookup("instance-type").Usage,
			Options:  instanceTypeList.GetAvailableIDs(cluster.MultiAZ()),
			Default:  instanceType,
			Required: true,
		})
		if err != nil {
			return fmt.Errorf("expected a valid instance type: %s", err)
		}
	}

	err = instanceTypeList.ValidateMachineType(instanceType, cluster.MultiAZ())
	if err != nil {
		return fmt.Errorf("expected a valid instance type: %s", err)
	}

	existingLabels := make(map[string]string, 0)
	labelMap := mpHelpers.GetLabelMap(cmd, r, existingLabels, args.Labels)

	existingTaints := make([]*cmv1.Taint, 0)
	taintBuilders := mpHelpers.GetTaints(cmd, r, existingTaints, args.Taints)

	// Spot instances
	isSpotSet := cmd.Flags().Changed("use-spot-instances")
	isSpotMaxPriceSet := cmd.Flags().Changed("spot-max-price")

	useSpotInstances := args.UseSpotInstances
	spotMaxPrice := args.SpotMaxPrice
	if isSpotMaxPriceSet && isSpotSet && !useSpotInstances {
		return fmt.Errorf("can't set max price when not using spot instances")
	}

	// Validate spot instance are supported
	var isLocalZone bool
	if subnet != "" {
		isLocalZone, err = r.AWSClient.IsLocalAvailabilityZone(availabilityZonesFilter[0])
		if err != nil {
			return err
		}
	}
	if isLocalZone && useSpotInstances {
		return fmt.Errorf("spot instances are not supported for local zones")
	}

	if !isSpotSet && !isSpotMaxPriceSet && !isLocalZone && interactive.Enabled() {
		useSpotInstances, err = interactive.GetBool(interactive.Input{
			Question: "Use spot instances",
			Help:     cmd.Flags().Lookup("use-spot-instances").Usage,
			Default:  useSpotInstances,
			Required: false,
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for use spot instances: %s", err)
		}
	}

	if useSpotInstances && !isSpotMaxPriceSet && interactive.Enabled() {
		spotMaxPrice, err = interactive.GetString(interactive.Input{
			Question: "Spot instance max price",
			Help:     cmd.Flags().Lookup("spot-max-price").Usage,
			Required: false,
			Default:  spotMaxPrice,
			Validators: []interactive.Validator{
				spotMaxPriceValidator,
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for spot max price: %s", err)
		}
	}

	var maxPrice *float64

	err = spotMaxPriceValidator(spotMaxPrice)
	if err != nil {
		return err
	}
	if spotMaxPrice != "on-demand" {
		price, _ := strconv.ParseFloat(spotMaxPrice, commonUtils.MaxByteSize)
		maxPrice = &price
	}

	awsTags := mpHelpers.GetAwsTags(cmd, r, args.Tags)

	mpBuilder := cmv1.NewMachinePool().
		ID(name).
		InstanceType(instanceType).
		Labels(labelMap).
		Taints(taintBuilders...)

	if autoscaling {
		mpBuilder = mpBuilder.Autoscaling(
			cmv1.NewMachinePoolAutoscaling().
				MinReplicas(minReplicas).
				MaxReplicas(maxReplicas))
	} else {
		mpBuilder = mpBuilder.Replicas(replicas)
	}

	awsMpBuilder := cmv1.NewAWSMachinePool()
	if useSpotInstances {
		spotBuilder := cmv1.NewAWSSpotMarketOptions()
		if maxPrice != nil {
			spotBuilder = spotBuilder.MaxPrice(*maxPrice)
		}
		awsMpBuilder.SpotMarketOptions(spotBuilder)
	}
	if len(securityGroupIds) > 0 {
		awsMpBuilder.AdditionalSecurityGroupIds(securityGroupIds...)
	}
	if len(awsTags) > 0 {
		awsMpBuilder.Tags(awsTags)
	}
	mpBuilder.AWS(awsMpBuilder)

	// Create a single AZ machine pool for a multi-AZ cluster
	if cluster.MultiAZ() && !multiAZMachinePool && availabilityZone != "" {
		mpBuilder.AvailabilityZones(availabilityZone)
	}

	// Create a single AZ machine pool for a BYOVPC cluster
	if subnet != "" {
		mpBuilder.Subnets(subnet)
	}

	_, _, _, _, defaultRootDiskSize, _ :=
		r.OCMClient.GetDefaultClusterFlavors(cluster.Flavour().ID())

	if args.RootDiskSize != "" || interactive.Enabled() {
		var rootDiskSizeStr string
		if args.RootDiskSize == "" {
			// We don't need to parse the default since it's returned from the OCM API and AWS
			// always defaults to GiB
			rootDiskSizeStr = helper.GigybyteStringer(defaultRootDiskSize)
		} else {
			rootDiskSizeStr = args.RootDiskSize
		}
		if interactive.Enabled() {
			// In order to avoid confusion, we want to display to the user what was passed as an
			// argument
			// Even if it was not valid, we want to display it to the user, then the CLI will show an
			// error and the value can be corrected
			// Also, if nothing is given, we want to display the default value fetched from the OCM API
			rootDiskSizeStr, err = interactive.GetString(interactive.Input{
				Question: "Root disk size (GiB or TiB)",
				Help:     cmd.Flags().Lookup("disk-size").Usage,
				Default:  rootDiskSizeStr,
				Validators: []interactive.Validator{
					interactive.MachinePoolRootDiskSizeValidator(cluster.Version().RawID()),
				},
			})
			if err != nil {
				return fmt.Errorf("expected a valid machine pool root disk size value: %v", err)
			}
		}

		// Parse the value given by either CLI or interactive mode and return it in GigiBytes
		rootDiskSize, err := ocm.ParseDiskSizeToGigibyte(rootDiskSizeStr)
		if err != nil {
			return fmt.Errorf("expected a valid machine pool root disk size value '%s': %v", rootDiskSizeStr, err)
		}

		err = diskValidator.ValidateMachinePoolRootDiskSize(cluster.Version().RawID(), rootDiskSize)
		if err != nil {
			return err
		}

		// If the size given by the user is different than the default, we just let the OCM server
		// handle the default root disk size
		if rootDiskSize != defaultRootDiskSize {
			mpBuilder.RootVolume(cmv1.NewRootVolume().AWS(cmv1.NewAWSVolume().Size(rootDiskSize)))
		}
	}

	machinePool, err := mpBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create machine pool for cluster '%s': %v", clusterKey, err)
	}

	createdMachinePool, err := r.OCMClient.CreateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		return fmt.Errorf("failed to add machine pool to cluster '%s': %v", clusterKey, err)
	}

	if output.HasFlag() {
		if err = output.Print(createdMachinePool); err != nil {
			return fmt.Errorf("unable to print machine pool: %v", err)
		}
	} else {
		r.Reporter.Infof("Machine pool '%s' created successfully on cluster '%s'", name, clusterKey)
		r.Reporter.Infof("To view the machine pool details, run 'rosa describe machinepool --cluster %s --machinepool %s'",
			clusterKey, name)
		r.Reporter.Infof("To view all machine pools, run 'rosa list machinepools --cluster %s'", clusterKey)
	}

	return nil
}

func (m *machinePool) CreateNodePools(r *rosa.Runtime, cmd *cobra.Command, clusterKey string, cluster *cmv1.Cluster,
	clusterAutoscaler *cmv1.ClusterAutoscaler, args *mpOpts.CreateMachinepoolUserOptions) error {

	var err error
	isMultiAvailabilityZoneSet := cmd.Flags().Changed("multi-availability-zone")
	if isMultiAvailabilityZoneSet {
		return fmt.Errorf("setting `multi-availability-zone` flag is not supported for HCP clusters")
	}

	isAvailabilityZoneSet := cmd.Flags().Changed("availability-zone")
	isSubnetSet := cmd.Flags().Changed("subnet")
	if isSubnetSet && isAvailabilityZoneSet {
		return fmt.Errorf("setting both `subnet` and `availability-zone` flag is not supported." +
			" Please select `subnet` or `availability-zone` to create a single availability zone machine pool")
	}

	// Machine pool name:
	name := strings.Trim(args.Name, " \t")
	if name == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}
	if name == "" || interactive.Enabled() {
		name, err = interactive.GetString(interactive.Input{
			Question: "Machine pool name",
			Default:  name,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(machinePoolKeyRE.String()),
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid name for the machine pool: %s", err)
		}
	}
	name = strings.Trim(name, " \t")
	if !machinePoolKeyRE.MatchString(name) {
		return fmt.Errorf("expected a valid name for the machine pool")
	}

	imageType := ""
	if args.Type != "" && cluster.Hypershift().Enabled() {
		imageType = args.Type
	}

	// Image type (supports things such as WinLI // Windows VMs)
	if interactive.Enabled() && cluster.Hypershift().Enabled() {
		imageType, err = interactive.GetOption(interactive.Input{
			Question: "Image Type",
			Default:  cmv1.ImageTypeDefault,
			Required: false,
			Options:  mpHelpers.ImageTypes,
		})
		if err != nil {
			return fmt.Errorf("expected a valid image type: '%s'", err)
		}
	}

	if imageType != "" && !mpHelpers.IsValidImageType(imageType) {
		return fmt.Errorf("expected a valid image type for the machine pool: '%s'", imageType)
	}

	// OpenShift version:
	isVersionSet := cmd.Flags().Changed("version")
	version := args.Version
	if isVersionSet || interactive.Enabled() {
		// NodePool will take channel group from the cluster
		channelGroup := cluster.Version().ChannelGroup()
		clusterVersion := cluster.Version().RawID()
		// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
		// so we pass the relative parameter as false
		_, versionList, err := versions.GetVersionList(r, channelGroup, true, true, false, false)
		if err != nil {
			return err
		}

		// Calculate the minimal version for a new hosted machine pool
		minVersion, err := versions.GetMinimalHostedMachinePoolVersion(clusterVersion)
		if err != nil {
			return err
		}

		// Filter the available list of versions for a hosted machine pool
		filteredVersionList := versions.GetFilteredVersionList(versionList, minVersion, clusterVersion)

		if version == "" {
			version = clusterVersion
		}
		if interactive.Enabled() {
			version, err = interactive.GetOption(interactive.Input{
				Question: "OpenShift version",
				Help:     cmd.Flags().Lookup("version").Usage,
				Options:  filteredVersionList,
				Default:  version,
				Required: true,
			})
			if err != nil {
				return fmt.Errorf("expected a valid OpenShift version: %s", err)
			}
		}
		// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
		// so we pass the relative parameter as false
		version, err = r.OCMClient.ValidateVersion(version, filteredVersionList, channelGroup, true, false)
		if err != nil {
			return fmt.Errorf("expected a valid OpenShift version: %s", err)
		}
	}

	// Allow the user to select subnet for a single AZ BYOVPC cluster
	subnet, err := getSubnetFromUser(cmd, r, isSubnetSet, cluster, args)
	if err != nil {
		return err
	}

	// Select availability zone if the user didn't select subnet
	if subnet == "" {
		subnet, err = getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster, args)
		if err != nil {
			return err
		}
	}

	replicaSizeValidation := &ReplicaSizeValidation{
		ClusterVersion: cluster.OpenshiftVersion(),
		MultiAz:        false,
		IsHostedCp:     cluster.Hypershift().Enabled(),
	}

	minReplicas, maxReplicas, replicas, autoscaling, err := manageReplicas(cmd, args, replicaSizeValidation)
	if err != nil {
		return err
	}

	existingLabels := make(map[string]string, 0)
	labelMap := mpHelpers.GetLabelMap(cmd, r, existingLabels, args.Labels)

	existingTaints := make([]*cmv1.Taint, 0)
	taintBuilders := mpHelpers.GetTaints(cmd, r, existingTaints, args.Taints)

	isSecurityGroupIdsSet := cmd.Flags().Changed(securitygroups.MachinePoolSecurityGroupFlag)
	securityGroupIds := args.SecurityGroupIds
	isVersionCompatibleSecurityGroupIds, err := features.IsFeatureSupported(
		features.AdditionalDay2SecurityGroupsHcpFeature, version)
	if err != nil {
		return err
	}
	if interactive.Enabled() && !isSecurityGroupIdsSet && isVersionCompatibleSecurityGroupIds {
		securityGroupIds, err = getSecurityGroupsOption(r, cmd, cluster)
		if err != nil {
			return err
		}
	}
	for i, sg := range securityGroupIds {
		securityGroupIds[i] = strings.TrimSpace(sg)
	}

	awsTags := mpHelpers.GetAwsTags(cmd, r, args.Tags)

	npBuilder := cmv1.NewNodePool()
	npBuilder.ID(name).Labels(labelMap).
		Taints(taintBuilders...)

	if autoscaling {
		npBuilder = npBuilder.Autoscaling(
			cmv1.NewNodePoolAutoscaling().
				MinReplica(minReplicas).
				MaxReplica(maxReplicas))
	} else {
		npBuilder = npBuilder.Replicas(replicas)
	}

	if subnet != "" {
		npBuilder.Subnet(subnet)
	}

	// Machine pool instance type:
	// NodePools don't support MultiAZ yet, so the availabilityZonesFilters is calculated from the cluster
	instanceType := args.InstanceType
	if instanceType == "" && !interactive.Enabled() {
		return fmt.Errorf("you must supply a valid instance type")
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() && !output.HasFlag() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Checking available instance types for machine pool '%s'", name)
		spin.Start()
	}

	availabilityZonesFilter := cluster.Nodes().AvailabilityZones()

	// If the user selects a subnet which is in a different AZ than day 1, the instance type list should be filter
	// by the new AZ not the cluster ones
	if subnet != "" {
		availabilityZone, err := r.AWSClient.GetSubnetAvailabilityZone(subnet)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		availabilityZonesFilter = []string{availabilityZone}
	}

	instanceTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(cluster.Region().ID(),
		availabilityZonesFilter, cluster.AWS().STS().RoleARN(), r.AWSClient, cluster.AWS().STS().ExternalID())
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	if spin != nil {
		spin.Stop()
	}

	if interactive.Enabled() {
		if instanceType == "" {
			instanceType = instanceTypeList.Items[0].MachineType.ID()
		}
		instanceType, err = interactive.GetOption(interactive.Input{
			Question: "Instance type",
			Help:     cmd.Flags().Lookup("instance-type").Usage,
			Options:  instanceTypeList.GetAvailableIDs(cluster.MultiAZ()),
			Default:  instanceType,
			Required: true,
		})
		if err != nil {
			return fmt.Errorf("expected a valid instance type: %s", err)
		}
	}

	err = instanceTypeList.ValidateMachineType(instanceType, cluster.MultiAZ())
	if err != nil {
		return fmt.Errorf("expected a valid instance type: %s", err)
	}

	autorepair := args.Autorepair
	if interactive.Enabled() {
		autorepair, err = interactive.GetBool(interactive.Input{
			Question: "Autorepair",
			Help:     cmd.Flags().Lookup("autorepair").Usage,
			Default:  autorepair,
			Required: false,
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for autorepair: %s", err)
		}
	}

	npBuilder.AutoRepair(autorepair)

	var inputTuningConfig []string
	tuningConfigs := args.TuningConfigs
	// Get the list of available tuning configs
	availableTuningConfigs, err := r.OCMClient.GetTuningConfigsName(cluster.ID())
	if err != nil {
		return err
	}
	if tuningConfigs != "" {
		if len(availableTuningConfigs) > 0 {
			inputTuningConfig = strings.Split(tuningConfigs, ",")
		} else {
			// Parameter will be ignored
			r.Reporter.Warnf("No tuning config available for cluster '%s'. "+
				"Any tuning config in input will be ignored", cluster.ID())
		}
	}
	if interactive.Enabled() {
		// Skip if no tuning configs are available
		if len(availableTuningConfigs) > 0 {
			inputTuningConfig, err = interactive.GetMultipleOptions(interactive.Input{
				Question: "Tuning configs",
				Help:     cmd.Flags().Lookup("tuning-configs").Usage,
				Options:  availableTuningConfigs,
				Default:  inputTuningConfig,
				Required: false,
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for tuning configs: %s", err)
			}
		}
	}

	if len(inputTuningConfig) != 0 {
		npBuilder.TuningConfigs(inputTuningConfig...)
	}

	// Capacity reservation ID
	capacityReservationId := args.CapacityReservationId

	if interactive.Enabled() && !autoscaling && !fedramp.Enabled() {
		capacityReservationId, err = interactive.GetString(interactive.Input{
			Question: "Capacity Reservation ID",
			Help:     cmd.Flags().Lookup("capacity-reservation-id").Usage,
			Default:  capacityReservationId,
			Required: false,
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for Capacity Reservation ID: %s", err)
		}
	}

	capacityReservationPreference := args.CapacityReservationPreference

	if interactive.Enabled() && !autoscaling && !fedramp.Enabled() {
		options := []string{mpHelpers.CapacityReservationPreferenceOnly}
		if capacityReservationId == "" {
			options = []string{mpHelpers.CapacityReservationPreferenceNone,
				mpHelpers.CapacityReservationPreferenceOnly, mpHelpers.CapacityReservationPreferenceOpen}
		}
		capacityReservationPreference, err = interactive.GetOption(interactive.Input{
			Question: "Capacity Reservation Preference",
			Help:     cmd.Flags().Lookup("capacity-reservation-preference").Usage,
			Options:  options,
			Required: false,
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for Capacity Reservation Preference: %s", err)
		}
	}

	if capacityReservationPreference != "" {
		err = mpHelpers.ValidateCapacityReservationPreference(capacityReservationPreference, capacityReservationId)
		if err != nil {
			return fmt.Errorf("expected a valid value for Capacity Reservation Preference: %s", err)
		}
	}

	capacityReservation := cmv1.NewAWSCapacityReservation().Id(capacityReservationId).Preference(
		cmv1.CapacityReservationPreference(capacityReservationPreference))

	kubeletConfigs := args.KubeletConfigs

	if (kubeletConfigs != "" || interactive.Enabled()) && !fedramp.Enabled() {
		var inputKubeletConfigs []string
		// Get the list of available kubelet configs
		availableKubeletConfigs, err := r.OCMClient.ListKubeletConfigNames(cluster.ID())
		if err != nil {
			return err
		}

		if len(availableKubeletConfigs) > 0 {
			inputKubeletConfigs = strings.Split(kubeletConfigs, ",")
		} else {
			// Parameter will be ignored
			r.Reporter.Warnf("No kubelet configs available for cluster '%s'. "+
				"Any kubelet config in input will be ignored", cluster.ID())
		}

		if interactive.Enabled() {
			// Skip if no kubelet configs are available
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfigs, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Kubelet config",
					Help:     cmd.Flags().Lookup("kubelet-configs").Usage,
					Options:  availableKubeletConfigs,
					Default:  inputKubeletConfigs,
					Required: false,
					Validators: []interactive.Validator{
						ValidateKubeletConfig,
					},
				})
				if err != nil {
					return fmt.Errorf("expected a valid value for kubelet config: %s", err)
				}
			}
		}

		err = ValidateKubeletConfig(inputKubeletConfigs)
		if err != nil {
			return fmt.Errorf("%s", err.Error())
		}

		if len(inputKubeletConfigs) != 0 {
			npBuilder.KubeletConfigs(inputKubeletConfigs...)
		}
	}

	httpTokens := args.EC2MetadataHttpTokens
	if httpTokens == "" {
		httpTokens = string(cmv1.Ec2MetadataHttpTokensOptional)
	}
	if interactive.Enabled() {
		httpTokens, err = interactive.GetOption(interactive.Input{
			Question: "Configure the use of IMDSv2 for ec2 instances",
			Options:  []string{string(cmv1.Ec2MetadataHttpTokensOptional), string(cmv1.Ec2MetadataHttpTokensRequired)},
			Help:     cmd.Flags().Lookup("ec2-metadata-http-tokens").Usage,
			Required: true,
			Default:  httpTokens,
		})
		if err != nil {
			return fmt.Errorf("expected a valid http tokens value : %v", err)
		}
	}

	if err = ocm.ValidateHttpTokensValue(httpTokens); err != nil {
		return fmt.Errorf("expected a valid http tokens value : %v", err)
	}

	var rootDiskSize *int
	_, _, _, _, defaultRootDiskSize, _ :=
		r.OCMClient.GetDefaultClusterFlavors(cluster.Flavour().ID())

	if args.RootDiskSize != "" || interactive.Enabled() {
		var rootDiskSizeStr string
		if args.RootDiskSize == "" {
			// We don't need to parse the default since it's returned from the OCM API and AWS
			// always defaults to GiB
			rootDiskSizeStr = helper.GigybyteStringer(defaultRootDiskSize)
		} else {
			rootDiskSizeStr = args.RootDiskSize
		}
		if interactive.Enabled() {
			// In order to avoid confusion, we want to display to the user what was passed as an
			// argument
			// Even if it was not valid, we want to display it to the user, then the CLI will show an
			// error and the value can be corrected
			// Also, if nothing is given, we want to display the default value fetched from the OCM API
			rootDiskSizeStr, err = interactive.GetString(interactive.Input{
				Question: "Root disk size (GiB or TiB)",
				Help:     cmd.Flags().Lookup("disk-size").Usage,
				Default:  rootDiskSizeStr,
				Validators: []interactive.Validator{
					interactive.NodePoolRootDiskSizeValidator(),
				},
			})
			if err != nil {
				return fmt.Errorf("expected a valid node pool root disk size value: %v", err)
			}
		}

		// Parse the value given by either CLI or interactive mode and return it in GigiBytes
		parsedRootDiskSize, err := ocm.ParseDiskSizeToGigibyte(rootDiskSizeStr)
		if err != nil {
			return fmt.Errorf("expected a valid node pool root disk size value '%s': %v", rootDiskSizeStr, err)
		}

		err = diskValidator.ValidateNodePoolRootDiskSize(parsedRootDiskSize)
		if err != nil {
			return err
		}

		if parsedRootDiskSize != defaultRootDiskSize {
			rootDiskSize = &parsedRootDiskSize
		}
	}

	awsNodepoolBuilder := createAwsNodePoolBuilder(
		instanceType,
		securityGroupIds,
		httpTokens,
		awsTags,
		rootDiskSize,
	)

	if capacityReservationId != "" && !fedramp.Enabled() && !autoscaling {
		awsNodepoolBuilder = awsNodepoolBuilder.CapacityReservation(capacityReservation)
	}

	npBuilder.AWSNodePool(awsNodepoolBuilder)

	nodeDrainGracePeriod := args.NodeDrainGracePeriod
	if interactive.Enabled() {
		nodeDrainGracePeriod, err = interactive.GetString(interactive.Input{
			Question: "Node drain grace period",
			Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
			Default:  nodeDrainGracePeriod,
			Required: false,
			Validators: []interactive.Validator{
				mpHelpers.ValidateNodeDrainGracePeriod,
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid value for Node drain grace period: %s", err)
		}
	}
	if nodeDrainGracePeriod != "" {
		nodeDrainBuilder, err := mpHelpers.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
		if err != nil {
			return fmt.Errorf("%v", err.Error())
		}
		npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
	}

	isMaxSurgeSet := cmd.Flags().Changed("max-surge")
	isMaxUnavailableSet := cmd.Flags().Changed("max-unavailable")
	if isMaxSurgeSet || isMaxUnavailableSet || interactive.Enabled() {
		maxSurge := args.MaxSurge
		if interactive.Enabled() {
			maxSurge, err = interactive.GetString(interactive.Input{
				Question: "Max surge",
				Help:     cmd.Flags().Lookup("max-surge").Usage,
				Default:  maxSurge,
				Required: false,
				Validators: []interactive.Validator{
					mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for max surge: %s", err)
			}
		}

		maxUnavailable := args.MaxUnavailable
		if interactive.Enabled() {
			maxUnavailable, err = interactive.GetString(interactive.Input{
				Question: "Max unavailable",
				Help:     cmd.Flags().Lookup("max-unavailable").Usage,
				Default:  maxUnavailable,
				Required: false,
				Validators: []interactive.Validator{
					mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for max unavailable: %s", err)
			}
		}
		if maxSurge != "" || maxUnavailable != "" {
			mgmtUpgradeBuilder := cmv1.NewNodePoolManagementUpgrade()
			if maxSurge != "" {
				mgmtUpgradeBuilder.MaxSurge(maxSurge)
			}
			if maxUnavailable != "" {
				mgmtUpgradeBuilder.MaxUnavailable(maxUnavailable)
			}
			npBuilder.ManagementUpgrade(mgmtUpgradeBuilder)
		}
	}

	sumOfReplicas := replicas
	sumOfMaxReplicas := maxReplicas
	sumOfMinReplicas := minReplicas

	nodepools, err := r.OCMClient.GetNodePools(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("error getting node pools from cluster '%s': %s", cluster.ID(), err)
	}

	for _, np := range nodepools {
		// If autoscaling, calculate min and max, use min and max in separate messages below
		npAutoscaling, ok := np.GetAutoscaling()
		if !ok || npAutoscaling == nil {
			npReplicas, _ := np.GetReplicas()
			sumOfReplicas += npReplicas
		} else {
			sumOfMaxReplicas += npAutoscaling.MaxReplica()
			sumOfMinReplicas += npAutoscaling.MinReplica()
		}
	}

	maxNodesTotal := 0
	if clusterAutoscaler != nil && clusterAutoscaler.ResourceLimits() != nil {
		maxNodesTotal = clusterAutoscaler.ResourceLimits().MaxNodesTotal()
	}

	if maxNodesTotal > 0 { // Do not perform if maxNodesTotal == 0
		// Informational message for cluster autoscaler + scaling out to max nodes
		if autoscaling {
			r.Reporter.Infof("Scaling max replicas to the maximum allowed value is subject to cluster autoscaler" +
				" configuration")
		}

		// Informational message for sum of replicas > MaxNodesTotal
		if sumOfReplicas+sumOfMaxReplicas > maxNodesTotal {
			r.Reporter.Infof("Actual maximum replicas can be lowered, since the replicas defined exceeds "+
				"%s", clusterAutoscalerLimitMessage)
		}

		// Informational message for min-replicas or replicas > MaxNodesTotal
		if sumOfReplicas+sumOfMinReplicas > maxNodesTotal {
			r.Reporter.Infof("Actual total nodes in the cluster will be more than the maximum nodes configured " +
				"in the cluster autoscaler")
		}
	}

	if imageType != "" {
		npBuilder.ImageType(cmv1.ImageType(imageType))
	}

	if version != "" {
		npBuilder.Version(cmv1.NewVersion().ID(version))
	}

	nodePool, err := npBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create machine pool for hosted cluster '%s': %v", clusterKey, err)
	}

	createdNodePool, err := r.OCMClient.CreateNodePool(cluster.ID(), nodePool)
	if err != nil {
		return fmt.Errorf("failed to add machine pool to hosted cluster '%s': %v", clusterKey, err)
	}

	if output.HasFlag() {
		if err = output.Print(createdNodePool); err != nil {
			return fmt.Errorf("unable to print machine pool: %v", err)
		}
	} else {
		r.Reporter.Infof("Machine pool '%s' created successfully on hosted cluster '%s'", createdNodePool.ID(), clusterKey)
		r.Reporter.Infof("To view the machine pool details, run 'rosa describe machinepool --cluster %s --machinepool %s'",
			clusterKey, name)
		r.Reporter.Infof("To view all machine pools, run 'rosa list machinepools --cluster %s'", clusterKey)
	}

	return nil
}

// ListMachinePools lists all machinepools (or, nodepools if hypershift) in a cluster
func (m *machinePool) ListMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster,
	args ListMachinePoolArgs) error {
	// Load any existing machine pools for this cluster
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	isHypershift := cluster.Hypershift().Enabled()
	var err error
	var machinePools []*cmv1.MachinePool
	var nodePools []*cmv1.NodePool
	if isHypershift {
		nodePools, err = r.OCMClient.GetNodePools(cluster.ID())
		if err != nil {
			return err
		}
	} else {
		machinePools, err = r.OCMClient.GetMachinePools(cluster.ID())
		if err != nil {
			return err
		}
	}

	if output.HasFlag() {
		if isHypershift {
			return output.Print(nodePools)
		}
		return output.Print(machinePools)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	finalStringToOutput := getMachinePoolsString(r, machinePools, args)
	if isHypershift {
		finalStringToOutput = getNodePoolsString(nodePools)
	}
	fmt.Fprint(writer, finalStringToOutput)
	writer.Flush()
	return nil
}

// DescribeMachinePool describes either a machinepool, or, a nodepool (if hypershift)
func (m *machinePool) DescribeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string,
	machinePoolId string) error {
	if cluster.Hypershift().Enabled() {
		return m.describeNodePool(r, cluster, clusterKey, machinePoolId)
	}

	r.Reporter.Debugf(fetchMessage, "machine pool", machinePoolId, clusterKey)
	machinePool, exists, err := r.OCMClient.GetMachinePool(cluster.ID(), machinePoolId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf(notFoundMessage, machinePoolId)
	}

	if output.HasFlag() {
		return output.Print(machinePool)
	}

	fmt.Print(machinePoolOutput(cluster.ID(), machinePool))

	return nil
}

func (m *machinePool) describeNodePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string,
	nodePoolId string) error {
	r.Reporter.Debugf(fetchMessage, "node pool", nodePoolId, clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf(notFoundMessage, nodePoolId)
	}

	_, scheduledUpgrade, err := r.OCMClient.GetHypershiftNodePoolUpgrade(cluster.ID(), clusterKey, nodePoolId)
	if err != nil {
		return err
	}

	if output.HasFlag() {
		var formattedOutput map[string]interface{}
		formattedOutput, err = formatNodePoolOutput(nodePool, scheduledUpgrade)
		if err != nil {
			return err
		}
		return output.Print(formattedOutput)
	}

	// Attach and print scheduledUpgrades if they exist, otherwise, print output normally
	fmt.Print(appendUpgradesIfExist(scheduledUpgrade, nodePoolOutput(cluster.ID(), nodePool)))

	return nil
}

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var MachinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

// DeleteMachinePool deletes a machinepool from a cluster if it is possible- this function also calls the hypershift
// equivalent, deleteNodePool if it is a hypershift cluster
func (m *machinePool) DeleteMachinePool(r *rosa.Runtime, machinePoolId string, clusterKey string,
	cluster *cmv1.Cluster) error {
	if cluster.Hypershift().Enabled() {
		return deleteNodePool(r, machinePoolId, clusterKey, cluster)
	}

	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get machine pools for cluster '%s': %v", clusterKey, err)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolId {
			machinePool = item
		}
	}
	if machinePool == nil {
		return fmt.Errorf("failed to get machine pool '%s' for cluster '%s'", machinePoolId, clusterKey)
	}

	if confirm.Confirm("delete machine pool '%s' on cluster '%s'", machinePoolId, clusterKey) {
		r.Reporter.Debugf("Deleting machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
		err = r.OCMClient.DeleteMachinePool(cluster.ID(), machinePool.ID())
		if err != nil {
			return fmt.Errorf("failed to delete machine pool '%s' on cluster '%s': %s",
				machinePool.ID(), clusterKey, err)
		}
		r.Reporter.Infof("Successfully deleted machine pool '%s' from cluster '%s'", machinePoolId, clusterKey)
	}
	return nil
}

// deleteNodePool is the hypershift version of DeleteMachinePool - deleteNodePool is called in DeleteMachinePool
// if the cluster is hypershift
func deleteNodePool(r *rosa.Runtime, nodePoolID string, clusterKey string, cluster *cmv1.Cluster) error {
	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for hosted cluster '%s'", clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		return fmt.Errorf("failed to get machine pools for hosted cluster '%s': %v", clusterKey,
			err)
	}
	if !exists {
		return fmt.Errorf("machine pool '%s' does not exist for hosted cluster '%s'", nodePoolID,
			clusterKey)
	}

	if confirm.Confirm("delete machine pool '%s' on hosted cluster '%s'", nodePoolID, clusterKey) {
		r.Reporter.Debugf("Deleting machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
		err = r.OCMClient.DeleteNodePool(cluster.ID(), nodePool.ID())
		if err != nil {
			return fmt.Errorf("failed to delete machine pool '%s' on hosted cluster '%s': %s",
				nodePool.ID(), clusterKey, err)
		}
		r.Reporter.Infof("Successfully deleted machine pool '%s' from hosted cluster '%s'", nodePoolID,
			clusterKey)
	}
	return nil
}

func formatNodePoolOutput(nodePool *cmv1.NodePool,
	scheduledUpgrade *cmv1.NodePoolUpgradePolicy) (map[string]interface{}, error) {

	var b bytes.Buffer
	err := cmv1.MarshalNodePool(nodePool, &b)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(b.Bytes(), &ret)
	if err != nil {
		return nil, err
	}
	if scheduledUpgrade != nil &&
		scheduledUpgrade.State() != nil &&
		len(scheduledUpgrade.Version()) > 0 &&
		len(scheduledUpgrade.State().Value()) > 0 {
		upgrade := make(map[string]interface{})
		upgrade["version"] = scheduledUpgrade.Version()
		upgrade["state"] = scheduledUpgrade.State().Value()
		upgrade["nextRun"] = scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST")
		ret["scheduledUpgrade"] = upgrade
	}

	return ret, nil
}

func appendUpgradesIfExist(scheduledUpgrade *cmv1.NodePoolUpgradePolicy, output string) string {
	if scheduledUpgrade != nil {
		return fmt.Sprintf("%s"+
			"Scheduled upgrade:                     %s %s on %s\n",
			output,
			scheduledUpgrade.State().Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
	}
	return output
}

func getMachinePoolsString(
	runtime *rosa.Runtime,
	machinePools []*cmv1.MachinePool,
	args ListMachinePoolArgs,
) string {
	type columnDefinition struct {
		header      string
		isVisible   bool
		extractData func(*cmv1.MachinePool) string
	}

	allColumnDefinitions := []columnDefinition{
		{"ID", true, func(mp *cmv1.MachinePool) string { return mp.ID() }},
		{"AUTOSCALING", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintMachinePoolAutoscaling(mp.Autoscaling())
		}},
		{"REPLICAS", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintMachinePoolReplicas(mp.Autoscaling(), mp.Replicas())
		}},
		{"INSTANCE TYPE", true, func(mp *cmv1.MachinePool) string {
			return mp.InstanceType()
		}},
		{"LABELS", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintLabels(mp.Labels())
		}},
		{"TAINTS", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintTaints(mp.Taints())
		}},
		{"AVAILABILITY ZONES", true, func(mp *cmv1.MachinePool) string {
			return output.PrintStringSlice(mp.AvailabilityZones())
		}},
		{"SUBNETS", true, func(mp *cmv1.MachinePool) string {
			return output.PrintStringSlice(mp.Subnets())
		}},
		{"SPOT INSTANCES", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintMachinePoolSpot(mp)
		}},
		{"DISK SIZE", true, func(mp *cmv1.MachinePool) string {
			return ocmOutput.PrintMachinePoolDiskSize(mp)
		}},
		{"SG IDS", true, func(mp *cmv1.MachinePool) string {
			return output.PrintStringSlice(mp.AWS().AdditionalSecurityGroupIds())
		}},
		{"AZ TYPE", args.ShowAZType || args.ShowAll, func(mp *cmv1.MachinePool) string {
			return getZoneType(mp)
		}},
		{"WIN-LI ENABLED", args.ShowWindowsLI || args.ShowAll,
			func(mp *cmv1.MachinePool) string { return isWinLIEnabled(mp.Labels()) }},
		{"DEDICATED HOST", args.ShowDedicated || args.ShowAll,
			func(mp *cmv1.MachinePool) string { return isDedicatedHost(mp, runtime) }},
	}

	var visibleColumnHeaders []string
	var visibleColumnData [][]string
	numPools := len(machinePools)

	for _, column := range allColumnDefinitions {
		if !column.isVisible {
			continue
		}

		columnValues := make([]string, numPools)
		hasNonEmptyValue := false

		for i, pool := range machinePools {
			if pool == nil {
				columnValues[i] = "-"
				continue
			}

			value := column.extractData(pool)
			columnValues[i] = value
			if value != "" && value != "-" {
				hasNonEmptyValue = true
			}
		}

		if args.ShowAll || hasNonEmptyValue || numPools == 0 {
			visibleColumnHeaders = append(visibleColumnHeaders, column.header)
			visibleColumnData = append(visibleColumnData, columnValues)
		}
	}

	var tableBuilder strings.Builder

	// Write header
	if len(visibleColumnHeaders) > 0 {
		tableBuilder.WriteString(strings.Join(visibleColumnHeaders, "\t") + "\n")
	}

	// Write data rows
	for rowIndex := range numPools {
		for colIndex, columnValues := range visibleColumnData {
			if colIndex > 0 {
				tableBuilder.WriteString("\t")
			}
			tableBuilder.WriteString(columnValues[rowIndex])
		}
		tableBuilder.WriteString("\n")
	}

	return tableBuilder.String()
}

func getNodePoolsString(nodePools []*cmv1.NodePool) string {
	outputString := "ID\tAUTOSCALING\tREPLICAS\t" +
		"INSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONE\tSUBNET\tDISK SIZE\tVERSION\tAUTOREPAIR\t\n"
	for _, nodePool := range nodePools {
		outputString += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t%s\t%s\t%s\t%s\t\n",
			nodePool.ID(),
			ocmOutput.PrintNodePoolAutoscaling(nodePool.Autoscaling()),
			ocmOutput.PrintNodePoolReplicasShort(
				ocmOutput.PrintNodePoolCurrentReplicas(nodePool.Status()),
				ocmOutput.PrintNodePoolReplicasInline(nodePool.Autoscaling(), nodePool.Replicas()),
			),
			ocmOutput.PrintNodePoolInstanceType(nodePool.AWSNodePool()),
			ocmOutput.PrintLabels(nodePool.Labels()),
			ocmOutput.PrintTaints(nodePool.Taints()),
			nodePool.AvailabilityZone(),
			nodePool.Subnet(),
			ocmOutput.PrintNodePoolDiskSize(nodePool.AWSNodePool()),
			ocmOutput.PrintNodePoolVersion(nodePool.Version()),
			ocmOutput.PrintNodePoolAutorepair(nodePool.AutoRepair()),
		)
	}
	return outputString
}

func (m *machinePool) EditMachinePool(cmd *cobra.Command, machinePoolId string, clusterKey string,
	cluster *cmv1.Cluster, r *rosa.Runtime) error {
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}

	if !MachinePoolKeyRE.MatchString(machinePoolId) {
		return fmt.Errorf("expected a valid identifier for the machine pool")
	}
	if cluster.Hypershift().Enabled() {
		clusterAutoscaler, err := r.OCMClient.GetClusterAutoscaler(cluster.ID())
		if err != nil {
			return errors.UserErrorf("failed to fetch cluster autoscaler for cluster '%s'", cluster.ID())
		}
		return editNodePool(cmd, machinePoolId, clusterKey, cluster, clusterAutoscaler, r)
	}
	return editMachinePool(cmd, machinePoolId, clusterKey, cluster, r)
}

// fillAutoScalingAndReplicas is filling either autoscaling or replicas value in the builder
func fillAutoScalingAndReplicas(npBuilder *cmv1.NodePoolBuilder, autoscaling bool, existingNodepool *cmv1.NodePool,
	minReplicas int, maxReplicas int, replicas int) {
	if autoscaling {
		npBuilder.Autoscaling(editAutoscaling(existingNodepool, minReplicas, maxReplicas))
	} else {
		npBuilder.Replicas(replicas)
	}
}

func getMachinePoolReplicas(cmd *cobra.Command,
	machinePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.MachinePoolAutoscaling,
	askForScalingParams bool,
	multiAZ bool,
	clusterVersion string,
) (autoscaling bool,
	replicas, minReplicas, maxReplicas int, err error) {
	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")

	replicas, err = cmd.Flags().GetInt("replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted replicas: %s", err)
		return
	}
	minReplicas, err = cmd.Flags().GetInt("min-replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted min replicas: %s", err)
		return
	}
	maxReplicas, err = cmd.Flags().GetInt("max-replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted max replicas: %s", err)
		return
	}
	autoscaling, err = cmd.Flags().GetBool("enable-autoscaling")
	if err != nil {
		err = fmt.Errorf("failed to get inputted autoscaling: %s", err)
		return
	}
	replicasRequired := existingAutoscaling == nil

	// if the user set min/max replicas and hasn't enabled autoscaling, or existing is disabled
	if (isMinReplicasSet || isMaxReplicasSet) && !autoscaling && existingAutoscaling == nil {
		err = fmt.Errorf("autoscaling is not enabled on machine pool '%s'. can't set min or max replicas",
			machinePoolID)
		return
	}

	// if the user set replicas but enabled autoscaling or hasn't disabled existing autoscaling
	if isReplicasSet && existingAutoscaling != nil && (!isAutoscalingSet || autoscaling) {
		err = fmt.Errorf("autoscaling enabled on machine pool '%s'. can't set replicas",
			machinePoolID)
		return
	}

	if !isAutoscalingSet {
		autoscaling = existingAutoscaling != nil
		if interactive.Enabled() {
			autoscaling, err = interactive.GetBool(interactive.Input{
				Question: "Enable autoscaling",
				Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
				Default:  autoscaling,
				Required: false,
			})
			if err != nil {
				err = fmt.Errorf("expected a valid value for enable-autoscaling: %s", err)
				return
			}
		}
	}

	replicaSizeValidation := &ReplicaSizeValidation{
		MinReplicas:    minReplicas,
		ClusterVersion: clusterVersion,
		MultiAz:        multiAZ,
		Autoscaling:    autoscaling,
		IsHostedCp:     false,
	}

	if autoscaling {
		// set default values from previous autoscaling values
		if !isMinReplicasSet {
			minReplicas = existingAutoscaling.MinReplicas()
		}
		if !isMaxReplicasSet {
			maxReplicas = existingAutoscaling.MaxReplicas()
		}

		// Prompt for min replicas if neither min or max is set or interactive mode
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet &&
			askForScalingParams) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: replicasRequired,
				Validators: []interactive.Validator{
					replicaSizeValidation.MinReplicaValidator(),
				},
			})
			if err != nil {
				err = fmt.Errorf("expected a valid number of min replicas: %s", err)
				return
			}
			replicaSizeValidation.MinReplicas = minReplicas
		}

		// Prompt for max replicas if neither min or max is set or interactive mode
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet &&
			askForScalingParams) {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: replicasRequired,
				Validators: []interactive.Validator{
					replicaSizeValidation.MaxReplicaValidator(),
				},
			})
			if err != nil {
				err = fmt.Errorf("expected a valid number of max replicas: %s", err)
				return
			}
		}
	} else {
		if !isReplicasSet {
			replicas = existingReplicas
		}
		if interactive.Enabled() || !isReplicasSet && askForScalingParams {
			replicas, err = interactive.GetInt(interactive.Input{
				Question: "Replicas",
				Help:     cmd.Flags().Lookup("replicas").Usage,
				Default:  replicas,
				Required: true,
				Validators: []interactive.Validator{
					replicaSizeValidation.MinReplicaValidator(),
				},
			})
			if err != nil {
				err = fmt.Errorf("expected a valid number of replicas: %s", err)
				return
			}
		}
	}

	err = validateEditInput("machine", autoscaling, minReplicas, maxReplicas, replicas, isReplicasSet,
		isAutoscalingSet, isMinReplicasSet, isMaxReplicasSet, machinePoolID)

	return
}

// Single-AZ: AvailabilityZones == []string{"us-east-1a"}
func isMultiAZMachinePool(machinePool *cmv1.MachinePool) bool {
	return len(machinePool.AvailabilityZones()) != 1
}

func editMachinePoolAutoscaling(machinePool *cmv1.MachinePool,
	minReplicas int, maxReplicas int) *cmv1.MachinePoolAutoscalingBuilder {
	asBuilder := cmv1.NewMachinePoolAutoscaling()
	changed := machinePool.Autoscaling().MinReplicas() != minReplicas && minReplicas >= 0

	if machinePool.Autoscaling().MaxReplicas() != maxReplicas && maxReplicas >= 0 {
		changed = true
	}

	if changed {
		asBuilder = asBuilder.MinReplicas(minReplicas).MaxReplicas(maxReplicas)
		return asBuilder
	}
	return nil
}

func editMachinePool(cmd *cobra.Command, machinePoolId string,
	clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) error {
	rosa.HostedClusterOnlyFlag(r, cmd, "autorepair")
	rosa.HostedClusterOnlyFlag(r, cmd, "tuning-configs")
	rosa.HostedClusterOnlyFlag(r, cmd, "kubelet-configs")

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")

	// if no value set enter interactive mode
	//nolint:staticcheck
	if !(isMinReplicasSet || isMaxReplicasSet || isReplicasSet || isAutoscalingSet || isLabelsSet || isTaintsSet) {
		interactive.Enable()
	}

	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get machine pools for cluster '%s': %v", clusterKey, err)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolId {
			machinePool = item
		}
	}
	if machinePool == nil {
		return fmt.Errorf("failed to get machine pool '%s' for cluster '%s'", machinePoolId, clusterKey)
	}

	autoscaling, replicas, minReplicas, maxReplicas, err :=
		getMachinePoolReplicas(cmd, machinePoolId, machinePool.Replicas(), machinePool.Autoscaling(),
			!isLabelsSet && !isTaintsSet, isMultiAZMachinePool(machinePool), cluster.OpenshiftVersion())

	if err != nil {
		return fmt.Errorf("failed to get autoscaling or replicas: '%s'", err)
	}

	if cluster.MultiAZ() && isMultiAZMachinePool(machinePool) &&
		(!autoscaling && replicas%3 != 0 ||
			(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
		return fmt.Errorf("multi AZ clusters require that the number of MachinePool replicas be a multiple of 3")
	}

	labels := cmd.Flags().Lookup("labels").Value.String()
	labelMap := mpHelpers.GetLabelMap(cmd, r, machinePool.Labels(), labels)

	taints := cmd.Flags().Lookup("taints").Value.String()
	taintBuilders := mpHelpers.GetTaints(cmd, r, machinePool.Taints(), taints)

	mpBuilder := cmv1.NewMachinePool().
		ID(machinePool.ID())

	// Check either for an explicit flag or interactive mode. Since
	// interactive will always show both labels and taints we can safely
	// assume that the value entered is the same as the value desired.
	if isLabelsSet || interactive.Enabled() {
		mpBuilder = mpBuilder.Labels(labelMap)
	}
	if isTaintsSet || interactive.Enabled() {
		mpBuilder = mpBuilder.Taints(taintBuilders...)
	}

	if autoscaling {
		mpBuilder.Autoscaling(editMachinePoolAutoscaling(machinePool, minReplicas, maxReplicas))
	} else {
		if machinePool.Replicas() != replicas {
			mpBuilder.Replicas(replicas)
		}
	}

	machinePool, err = mpBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create machine pool for cluster '%s': %v", clusterKey, err)
	}

	r.Reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		return fmt.Errorf("failed to update machine pool '%s' on cluster '%s': %s",
			machinePool.ID(), clusterKey, err)
	}
	r.Reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	return nil
}

func editNodePool(cmd *cobra.Command, nodePoolID string,
	clusterKey string, cluster *cmv1.Cluster, clusterAutoscaler *cmv1.ClusterAutoscaler, r *rosa.Runtime) error {
	var err error

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")
	isAutorepairSet := cmd.Flags().Changed("autorepair")
	isTuningsConfigSet := cmd.Flags().Changed("tuning-configs")
	isKubeletConfigSet := cmd.Flags().Changed("kubelet-configs")
	isNodeDrainGracePeriodSet := cmd.Flags().Changed("node-drain-grace-period")
	isUpgradeMaxSurgeSet := cmd.Flags().Changed("max-surge")
	isUpgradeMaxUnavailableSet := cmd.Flags().Changed("max-unavailable")

	// isAnyAdditionalParameterSet is true if at least one parameter not related to replicas and autoscaling is set
	isAnyAdditionalParameterSet := isLabelsSet || isTaintsSet || isAutorepairSet || isTuningsConfigSet ||
		isKubeletConfigSet || isUpgradeMaxSurgeSet || isUpgradeMaxUnavailableSet
	isAnyParameterSet := isMinReplicasSet || isMaxReplicasSet || isReplicasSet ||
		isAutoscalingSet || isAnyAdditionalParameterSet

	// if no value set enter interactive mode
	if !isAnyParameterSet {
		interactive.Enable()
	}

	// Try to find the node pool
	r.Reporter.Debugf("Loading machine pool for hosted cluster '%s'", clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		return fmt.Errorf("failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
	}
	if !exists {
		return fmt.Errorf("machine pool '%s' does not exist for hosted cluster '%s'", nodePoolID, clusterKey)
	}

	autoscaling, replicas, minReplicas, maxReplicas, err := getNodePoolReplicas(cmd, nodePoolID,
		nodePool.Replicas(), nodePool.Autoscaling(), isAnyAdditionalParameterSet, cluster.OpenshiftVersion())
	if err != nil {
		return fmt.Errorf("failed to get autoscaling or replicas: '%s'", err)
	}

	err = validateNodePoolEdit(cmd, autoscaling, replicas, minReplicas, maxReplicas)
	if err != nil {
		return err
	}

	labels := cmd.Flags().Lookup("labels").Value.String()
	labelMap := mpHelpers.GetLabelMap(cmd, r, nodePool.Labels(), labels)

	taints := cmd.Flags().Lookup("taints").Value.String()
	taintBuilders := mpHelpers.GetTaints(cmd, r, nodePool.Taints(), taints)

	npBuilder := cmv1.NewNodePool().
		ID(nodePool.ID())

	// Check either for an explicit flag or interactive mode. Since
	// interactive will always show both labels and taints we can safely
	// assume that the value entered is the same as the value desired.
	if isLabelsSet || interactive.Enabled() {
		npBuilder = npBuilder.Labels(labelMap)
	}
	if isTaintsSet || interactive.Enabled() {
		npBuilder = npBuilder.Taints(taintBuilders...)
	}

	fillAutoScalingAndReplicas(npBuilder, autoscaling, nodePool, minReplicas, maxReplicas, replicas)

	if isAutorepairSet || interactive.Enabled() {
		autorepair, err := strconv.ParseBool(cmd.Flags().Lookup("autorepair").Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse autorepair flag: %s", err)
		}
		if interactive.Enabled() {
			autorepair, err = interactive.GetBool(interactive.Input{
				Question: "Autorepair",
				Help:     cmd.Flags().Lookup("autorepair").Usage,
				Default:  autorepair,
				Required: false,
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for autorepair: %s", err)
			}
		}

		npBuilder.AutoRepair(autorepair)
	}

	if isTuningsConfigSet || interactive.Enabled() {
		var inputTuningConfig []string
		tuningConfigs := cmd.Flags().Lookup("tuning-configs").Value.String()
		// Get the list of available tuning configs
		availableTuningConfigs, err := r.OCMClient.GetTuningConfigsName(cluster.ID())
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		if tuningConfigs != "" {
			if len(availableTuningConfigs) > 0 {
				inputTuningConfig = strings.Split(tuningConfigs, ",")
			} else {
				// Parameter will be ignored
				r.Reporter.Warnf("No tuning config available for cluster '%s'. "+
					"Any tuning config in input will be ignored", cluster.ID())
			}
		}

		if interactive.Enabled() {
			if !isTuningsConfigSet {
				// Interactive mode without explicit input parameter. Take the existing value
				inputTuningConfig = nodePool.TuningConfigs()
			}

			// Skip if no tuning configs are available
			if len(availableTuningConfigs) > 0 {
				inputTuningConfig, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Tuning configs",
					Help:     cmd.Flags().Lookup("tuning-configs").Usage,
					Options:  availableTuningConfigs,
					Default:  inputTuningConfig,
					Required: false,
				})
				if err != nil {
					return fmt.Errorf("expected a valid value for tuning configs: %s", err)
				}
			}
		}

		npBuilder.TuningConfigs(inputTuningConfig...)
	}

	if (isKubeletConfigSet || interactive.Enabled()) && !fedramp.Enabled() {
		var inputKubeletConfig []string
		kubeletConfigs := cmd.Flags().Lookup("kubelet-configs").Value.String()
		// Get the list of available tuning configs
		availableKubeletConfigs, err := r.OCMClient.ListKubeletConfigNames(cluster.ID())
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		if kubeletConfigs != "" {
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfig = strings.Split(kubeletConfigs, ",")
			} else {
				// Parameter will be ignored
				r.Reporter.Warnf("No kubelet config available for cluster '%s'. "+
					"Any kubelet config in input will be ignored", cluster.ID())
			}
		}

		if interactive.Enabled() {
			if !isKubeletConfigSet {
				// Interactive mode without explicit input parameter. Take the existing value
				inputKubeletConfig = nodePool.KubeletConfigs()
			}

			// Skip if no kubelet configs are available
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfig, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Kubelet config",
					Help:     cmd.Flags().Lookup("kubelet-configs").Usage,
					Options:  availableKubeletConfigs,
					Default:  inputKubeletConfig,
					Required: false,
					Validators: []interactive.Validator{
						ValidateKubeletConfig,
					},
				})
				if err != nil {
					return fmt.Errorf("expected a valid value for kubelet config: %s", err)
				}
			}
		}
		err = ValidateKubeletConfig(inputKubeletConfig)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
		npBuilder.KubeletConfigs(inputKubeletConfig...)
		isKubeletConfigSet = true
	}

	if isNodeDrainGracePeriodSet || interactive.Enabled() {
		nodeDrainGracePeriod := cmd.Flags().Lookup("node-drain-grace-period").Value.String()
		if nodeDrainGracePeriod == "" && nodePool.NodeDrainGracePeriod() != nil &&
			nodePool.NodeDrainGracePeriod().Value() != 0 {
			nodeDrainGracePeriod = fmt.Sprintf("%d minutes", int(nodePool.NodeDrainGracePeriod().Value()))
		}

		if interactive.Enabled() {
			nodeDrainGracePeriod, err = interactive.GetString(interactive.Input{
				Question: "Node drain grace period",
				Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
				Default:  nodeDrainGracePeriod,
				Required: false,
				Validators: []interactive.Validator{
					mpHelpers.ValidateNodeDrainGracePeriod,
				},
			})
			if err != nil {
				return fmt.Errorf("expected a valid value for Node drain grace period: %s", err)
			}
		}

		if nodeDrainGracePeriod != "" {
			nodeDrainBuilder, err := mpHelpers.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
			if err != nil {
				return fmt.Errorf("%v", err.Error())
			}
			npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
		}
	}

	if isUpgradeMaxSurgeSet || isUpgradeMaxUnavailableSet || interactive.Enabled() {
		maxSurge := cmd.Flags().Lookup("max-surge").Value.String()
		if maxSurge == "" && nodePool.ManagementUpgrade().MaxSurge() != "" {
			maxSurge = nodePool.ManagementUpgrade().MaxSurge()
		}

		maxUnavailable := cmd.Flags().Lookup("max-unavailable").Value.String()
		if maxUnavailable == "" && nodePool.ManagementUpgrade().MaxUnavailable() != "" {
			maxUnavailable = nodePool.ManagementUpgrade().MaxUnavailable()
		}
		if interactive.Enabled() {
			maxSurge, err = interactive.GetString(interactive.Input{
				Question: "Max surge",
				Help:     cmd.Flags().Lookup("max-surge").Usage,
				Default:  maxSurge,
				Required: false,
				Validators: []interactive.Validator{
					mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("expected a valid value for max surge: %s", err)
				os.Exit(1)
			}

			maxUnavailable, err = interactive.GetString(interactive.Input{
				Question: "Max unavailable",
				Help:     cmd.Flags().Lookup("max-unavailable").Usage,
				Default:  maxUnavailable,
				Required: false,
				Validators: []interactive.Validator{
					mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("expected a valid value for max unavailable: %s", err)
				os.Exit(1)
			}
		}

		if maxSurge != "" || maxUnavailable != "" {
			mgmtUpgradeBuilder := cmv1.NewNodePoolManagementUpgrade()
			if maxSurge != "" {
				mgmtUpgradeBuilder.MaxSurge(maxSurge)
			}
			if maxUnavailable != "" {
				mgmtUpgradeBuilder.MaxUnavailable(maxUnavailable)
			}
			npBuilder.ManagementUpgrade(mgmtUpgradeBuilder)
		}
	}

	sumOfReplicas := replicas
	sumOfMaxReplicas := maxReplicas
	sumOfMinReplicas := minReplicas

	nodepools, err := r.OCMClient.GetNodePools(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("error getting node pools from cluster '%s': %s", cluster.ID(), err)
	}

	for _, np := range nodepools {
		// If autoscaling, calculate min and max, use min and max in separate messages below
		npAutoscaling, ok := np.GetAutoscaling()
		if !ok || npAutoscaling == nil {
			npReplicas, _ := np.GetReplicas()
			sumOfReplicas += npReplicas
		} else {
			sumOfMaxReplicas += npAutoscaling.MaxReplica()
			sumOfMinReplicas += npAutoscaling.MinReplica()
		}
	}

	maxNodesTotal := 0
	if clusterAutoscaler != nil && clusterAutoscaler.ResourceLimits() != nil {
		maxNodesTotal = clusterAutoscaler.ResourceLimits().MaxNodesTotal()
	}

	if maxNodesTotal > 0 { // Do not perform if maxNodesTotal == 0
		// Informational message for cluster autoscaler + scaling out to max nodes
		if autoscaling {
			r.Reporter.Infof("Scaling max replicas to the maximum allowed value is subject to cluster autoscaler" +
				" configuration")
		}

		// Informational message for sum of replicas > MaxNodesTotal
		if sumOfReplicas+sumOfMaxReplicas > maxNodesTotal {
			r.Reporter.Infof("Actual maximum replicas can be lowered, since the replicas defined exceeds "+
				"%s", clusterAutoscalerLimitMessage)
		}

		// Informational message for min-replicas or replicas > MaxNodesTotal
		if sumOfReplicas+sumOfMinReplicas > maxNodesTotal {
			r.Reporter.Infof("Actual total nodes in the cluster will be more than the maximum nodes configured " +
				"in the cluster autoscaler")
		}
	}

	update, err := npBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create machine pool for hosted cluster '%s': %v", clusterKey, err)
	}

	if isKubeletConfigSet && !promptForNodePoolNodeRecreate(
		nodePool, update, kubeletconfig.PromptToAcceptNodePoolNodeRecreate, r) {
		return nil
	}

	r.Reporter.Debugf("Updating machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateNodePool(cluster.ID(), update)
	if err != nil {
		return fmt.Errorf("failed to update machine pool '%s' on hosted cluster '%s': %s",
			nodePool.ID(), clusterKey, err)
	}
	r.Reporter.Infof("Updated machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
	return nil
}

func validateNodePoolEdit(cmd *cobra.Command, autoscaling bool, replicas int, minReplicas int, maxReplicas int) error {
	if !autoscaling && replicas < 0 {
		return fmt.Errorf("the number of machine pool replicas needs to be a non-negative integer")
	}

	if autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 1 {
		return fmt.Errorf("min-replicas must be greater than zero")
	}

	if autoscaling && cmd.Flags().Changed("max-replicas") && maxReplicas < 1 {
		return fmt.Errorf("max-replicas must be greater than zero")
	}

	if autoscaling && cmd.Flags().Changed("max-replicas") && cmd.Flags().Changed(
		"min-replicas") && minReplicas > maxReplicas {
		return fmt.Errorf("the number of machine pool min-replicas needs to be less than the number of " +
			"machine pool max-replicas")
	}
	return nil
}

// promptForNodePoolNodeRecreate - prompts the user to accept that their changes will cause the nodes
// in their nodepool to be recreated. This primarily applies to KubeletConfig modifications.
func promptForNodePoolNodeRecreate(
	original *cmv1.NodePool,
	update *cmv1.NodePool,
	promptFunc func(r *rosa.Runtime) bool, r *rosa.Runtime) bool {
	if len(original.KubeletConfigs()) != len(update.KubeletConfigs()) {
		return promptFunc(r)
	}

	for _, s := range update.KubeletConfigs() {
		if !slices.Contains(original.KubeletConfigs(), s) {
			return promptFunc(r)
		}
	}

	return true
}

func getNodePoolReplicas(cmd *cobra.Command,
	nodePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.NodePoolAutoscaling,
	isAnyAdditionalParameterSet bool,
	clusterVersion string) (autoscaling bool,
	replicas, minReplicas, maxReplicas int, err error) {

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")

	replicas, err = cmd.Flags().GetInt("replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted replicas: %s", err)
		return
	}
	minReplicas, err = cmd.Flags().GetInt("min-replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted min replicas: %s", err)
		return
	}
	maxReplicas, err = cmd.Flags().GetInt("max-replicas")
	if err != nil {
		err = fmt.Errorf("failed to get inputted max replicas: %s", err)
		return
	}
	autoscaling, err = cmd.Flags().GetBool("enable-autoscaling")
	if err != nil {
		err = fmt.Errorf("failed to get inputted autoscaling: %s", err)
		return
	}
	replicasRequired := existingAutoscaling == nil

	// if the user set min/max replicas and hasn't enabled autoscaling, or existing is disabled
	if (isMinReplicasSet || isMaxReplicasSet) && !autoscaling && existingAutoscaling == nil {
		err = fmt.Errorf("autoscaling is not enabled on machine pool '%s'. can't set min or max replicas",
			nodePoolID)
		return
	}

	// if the user set replicas but enabled autoscaling or hasn't disabled existing autoscaling
	if isReplicasSet && existingAutoscaling != nil && (!isAutoscalingSet || autoscaling) {
		err = fmt.Errorf("autoscaling enabled on machine pool '%s'. can't set replicas",
			nodePoolID)
		return
	}

	if !isAutoscalingSet {
		autoscaling = existingAutoscaling != nil
		if interactive.Enabled() {
			autoscaling, err = interactive.GetBool(interactive.Input{
				Question: "Enable autoscaling",
				Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
				Default:  autoscaling,
				Required: false,
			})
			if err != nil {
				err = fmt.Errorf("expected a valid value for enable-autoscaling: %s", err)
				return
			}
		}
	}

	replicaSizeValidation := &ReplicaSizeValidation{
		MinReplicas:    minReplicas,
		ClusterVersion: clusterVersion,
		MultiAz:        false,
		Autoscaling:    autoscaling,
		IsHostedCp:     true,
	}

	if autoscaling {
		// Prompt for min replicas if neither min or max is set or interactive mode
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet && !isAnyAdditionalParameterSet) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  existingAutoscaling.MinReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					replicaSizeValidation.MinReplicaValidator(),
				},
			})
			if err != nil {
				err = fmt.Errorf("expected a valid number of min replicas: %s", err)
				return
			}
			replicaSizeValidation.MinReplicas = minReplicas
		}

		// Prompt for max replicas if neither min or max is set or interactive mode
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet && !isAnyAdditionalParameterSet) {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  existingAutoscaling.MaxReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					replicaSizeValidation.MaxReplicaValidator(),
				},
			})
			if err != nil {
				err = fmt.Errorf("expected a valid number of max replicas: %s", err)
				return
			}
		}
	} else if interactive.Enabled() || !isReplicasSet {
		if !isReplicasSet {
			replicas = existingReplicas
		}
		if !interactive.Enabled() && isAnyAdditionalParameterSet {
			// Not interactive mode and we have at least an additional parameter set, just keep the existing replicas
			return
		}
		replicas, err = interactive.GetInt(interactive.Input{
			Question: "Replicas",
			Help:     cmd.Flags().Lookup("replicas").Usage,
			Default:  replicas,
			Required: true,
			Validators: []interactive.Validator{
				replicaSizeValidation.MinReplicaValidator(),
			},
		})
		if err != nil {
			err = fmt.Errorf("expected a valid number of replicas: %s", err)
			return
		}
	}

	err = validateEditInput("node", autoscaling, minReplicas, maxReplicas, replicas, isReplicasSet,
		isAutoscalingSet, isMinReplicasSet, isMaxReplicasSet, nodePoolID)

	return
}

func editAutoscaling(nodePool *cmv1.NodePool, minReplicas int, maxReplicas int) *cmv1.NodePoolAutoscalingBuilder {
	existingMinReplica := nodePool.Autoscaling().MinReplica()
	existingMaxReplica := nodePool.Autoscaling().MaxReplica()

	min := existingMinReplica
	max := existingMaxReplica

	if minReplicas != 0 {
		min = minReplicas
	}
	if maxReplicas != 0 {
		max = maxReplicas
	}

	if min >= 1 && max >= 1 {
		return cmv1.NewNodePoolAutoscaling().MinReplica(min).MaxReplica(max)
	}

	return nil
}
func manageReplicas(cmd *cobra.Command, args *mpOpts.CreateMachinepoolUserOptions,
	replicaSizeValidation *ReplicaSizeValidation) (
	minReplicas, maxReplicas, replicas int, autoscaling bool, err error) {
	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isReplicasSet := cmd.Flags().Changed("replicas")

	minReplicas = args.MinReplicas
	maxReplicas = args.MaxReplicas
	autoscaling = args.AutoscalingEnabled
	replicas = args.Replicas

	// Autoscaling
	if !isReplicasSet && !autoscaling && !isAutoscalingSet && interactive.Enabled() {
		autoscaling, err = interactive.GetBool(interactive.Input{
			Question: "Enable autoscaling",
			Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
			Default:  autoscaling,
			Required: false,
		})
		if err != nil {
			return minReplicas, maxReplicas, replicas, autoscaling,
				fmt.Errorf("expected a valid value for enable-autoscaling: %s", err)
		}
	}

	if autoscaling {
		// if the user set replicas and enabled autoscaling
		if isReplicasSet {
			return minReplicas, maxReplicas, replicas, autoscaling,
				fmt.Errorf("replicas can't be set when autoscaling is enabled")
		}

		// Min replicas
		if interactive.Enabled() || !isMinReplicasSet {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: true,
				Validators: []interactive.Validator{
					replicaSizeValidation.MinReplicaValidator(),
				},
			})
			if err != nil {
				return minReplicas, maxReplicas, replicas, autoscaling,
					fmt.Errorf("expected a valid number of min replicas: %s", err)
			}
		}
		err = replicaSizeValidation.MinReplicaValidator()(minReplicas)
		if err != nil {
			return minReplicas, maxReplicas, replicas, autoscaling, err
		}
		replicaSizeValidation.MinReplicas = minReplicas

		// Max replicas
		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					replicaSizeValidation.MaxReplicaValidator(),
				},
			})
			if err != nil {
				return minReplicas, maxReplicas, replicas, autoscaling,
					fmt.Errorf("expected a valid number of max replicas: %s", err)
			}
		}
		err = replicaSizeValidation.MaxReplicaValidator()(maxReplicas)
		if err != nil {
			return minReplicas, maxReplicas, replicas, autoscaling, err
		}
	} else {
		// if the user set min/max replicas and hasn't enabled autoscaling
		if isMinReplicasSet || isMaxReplicasSet {
			return minReplicas, maxReplicas, replicas, autoscaling,
				fmt.Errorf("autoscaling must be enabled in order to set min and max replicas")
		}

		// Replicas
		if interactive.Enabled() || !isReplicasSet {
			replicas, err = interactive.GetInt(interactive.Input{
				Question: "Replicas",
				Help:     cmd.Flags().Lookup("replicas").Usage,
				Default:  replicas,
				Required: true,
				Validators: []interactive.Validator{
					replicaSizeValidation.MinReplicaValidator(),
				},
			})
			if err != nil {
				return minReplicas, maxReplicas, replicas, autoscaling, fmt.Errorf("expected a valid number of replicas: %s", err)
			}
		}
		err = replicaSizeValidation.MinReplicaValidator()(replicas)
		if err != nil {
			return minReplicas, maxReplicas, replicas, autoscaling, err
		}
	}
	return minReplicas, maxReplicas, replicas, autoscaling, nil
}
