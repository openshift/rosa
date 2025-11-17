package machinepool

import (
	"fmt"
	"strconv"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	clustervalidations "github.com/openshift-online/ocm-common/pkg/cluster/validations"
	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/interactive"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	hcpMaxNodesLimit = 500

	zoneTypeWavelength = "Wavelength"
	zoneTypeOutpost    = "Outpost"
	zoneTypeLocalZone  = "LocalZone"
	zoneTypeStandard   = "Standard"
	zoneTypeNA         = "N/A"

	displayValueYes     = "Yes"
	displayValueNo      = "No"
	displayValueUnknown = "Unknown"

	imageTypeWindows = "windows"

	labelImageType = "image_type"
)

type ReplicaSizeValidation struct {
	MinReplicas         int
	ClusterVersion      string
	PrivateSubnetsCount int
	Autoscaling         bool
	IsHostedCp          bool
	MultiAz             bool
}

// Parse labels if the 'labels' flag is set
func ValidateLabels(cmd *cobra.Command, args *mpOpts.CreateMachinepoolUserOptions) error {
	if cmd.Flags().Changed("labels") {
		if _, err := mpHelpers.ParseLabels(args.Labels); err != nil {
			return fmt.Errorf("%s", err)
		}
	}
	return nil
}

// Validate that the image type is a real one
func ValidateImageType(cmd *cobra.Command, args *mpOpts.CreateMachinepoolUserOptions, cluster *cmv1.Cluster) error {
	if cmd.Flags().Changed("type") {
		if !cluster.Hypershift().Enabled() {
			return fmt.Errorf("the '--type' flag can only be used with Hosted Control Plane clusters")
		}
		imageType := args.Type

		if mpHelpers.IsValidImageType(imageType) {
			return nil
		}
		return fmt.Errorf("invalid image type: '%s' - please use one of: %v", imageType, prettyPrintImageTypes())
	}
	return nil
}

// Validate the cluster's state is ready
func ValidateClusterState(cluster *cmv1.Cluster, clusterKey string) error {
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}
	return nil
}

func getSubnetFromUser(cmd *cobra.Command, r *rosa.Runtime, isSubnetSet bool,
	cluster *cmv1.Cluster, args *mpOpts.CreateMachinepoolUserOptions) (string, error) {
	var selectSubnet bool
	var subnet string
	var err error

	question := "Select subnet for a single AZ machine pool"
	questionError := "Expected a valid value for subnet for a single AZ machine pool"

	if cluster.Hypershift().Enabled() {
		question = "Select subnet for a hosted machine pool"
		questionError = "Expected a valid value for subnet for a hosted machine pool"
	}

	if !isSubnetSet && interactive.Enabled() {
		selectSubnet, err = interactive.GetBool(interactive.Input{
			Question: question,
			Help:     cmd.Flags().Lookup("subnet").Usage,
			Default:  false,
			Required: false,
		})
		if err != nil {
			return "", fmt.Errorf("%s", questionError)
		}
	} else {
		subnet = args.Subnet
	}

	if selectSubnet {
		subnetOptions, err := getSubnetOptions(r, cluster)
		if err != nil {
			return "", err
		}

		subnetOption, err := interactive.GetOption(interactive.Input{
			Question: "Subnet ID",
			Help:     cmd.Flags().Lookup("subnet").Usage,
			Options:  subnetOptions,
			Default:  subnetOptions[0],
			Required: true,
		})
		if err != nil {
			return "", fmt.Errorf("expected a valid AWS subnet: %s", err)
		}
		subnet = aws.ParseOption(subnetOption)
	}

	return subnet, nil
}

// getSubnetOptions gets one of the cluster subnets and returns a slice of formatted VPC's private subnets.
func getSubnetOptions(r *rosa.Runtime, cluster *cmv1.Cluster) ([]string, error) {
	// Fetch VPC's subnets
	privateSubnets, err := r.AWSClient.GetVPCPrivateSubnets(cluster.AWS().SubnetIDs()[0])
	if err != nil {
		return nil, err
	}

	// Format subnet options
	var subnetOptions []string
	for _, subnet := range privateSubnets {
		subnetOptions = append(subnetOptions, aws.SetSubnetOption(subnet))
	}

	return subnetOptions, nil
}

func getSecurityGroupsOption(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) ([]string, error) {
	if len(cluster.AWS().SubnetIDs()) == 0 {
		return []string{}, fmt.Errorf("expected cluster's subnets to contain subnets IDs, but got an empty list")
	}

	availableSubnets, err := r.AWSClient.GetVPCSubnets(cluster.AWS().SubnetIDs()[0])
	if err != nil {
		return []string{}, fmt.Errorf("failed to retrieve available subnets: %v", err)
	}
	firstSubnet := availableSubnets[0]
	vpcId, err := getVpcIdFromSubnet(firstSubnet)
	if err != nil {
		return []string{}, err
	}

	var id string
	if cluster.Hypershift().Enabled() {
		id = cluster.ID()
	} else {
		id = cluster.InfraID()
	}

	return interactiveSgs.GetSecurityGroupIds(r, cmd, vpcId, interactiveSgs.MachinePoolKind, id), nil
}

func createAwsNodePoolBuilder(
	instanceType string,
	securityGroupIds []string,
	httpTokens string,
	awsTags map[string]string,
	rootDiskSize *int,
) *cmv1.AWSNodePoolBuilder {
	awsNpBuilder := cmv1.NewAWSNodePool().InstanceType(instanceType)

	if len(securityGroupIds) > 0 {
		awsNpBuilder.AdditionalSecurityGroupIds(securityGroupIds...)
	}

	if len(awsTags) > 0 {
		awsNpBuilder.Tags(awsTags)
	}

	awsNpBuilder.Ec2MetadataHttpTokens(cmv1.Ec2MetadataHttpTokens(httpTokens))

	if rootDiskSize != nil {
		awsNpBuilder.RootVolume(cmv1.NewAWSVolume().Size(*rootDiskSize))
	}

	return awsNpBuilder
}

func getVpcIdFromSubnet(subnet ec2types.Subnet) (string, error) {
	vpcId := awssdk.ToString(subnet.VpcId)
	if vpcId == "" {
		return "", fmt.Errorf("unexpected situation a VPC ID should have been selected based on chosen subnets")
	}

	return vpcId, nil
}

func Split(r rune) bool {
	return r == '=' || r == ':'
}

// getMachinePoolAvailabilityZones derives the availability zone from the user input or the cluster spec
func getMachinePoolAvailabilityZones(r *rosa.Runtime, cluster *cmv1.Cluster, multiAZMachinePool bool,
	availabilityZoneUserInput string, subnetUserInput string) ([]string, error) {
	// Single AZ machine pool for a multi-AZ cluster
	if cluster.MultiAZ() && !multiAZMachinePool && availabilityZoneUserInput != "" {
		return []string{availabilityZoneUserInput}, nil
	}

	// Single AZ machine pool for a BYOVPC cluster
	if subnetUserInput != "" {
		availabilityZone, err := r.AWSClient.GetSubnetAvailabilityZone(subnetUserInput)
		if err != nil {
			return []string{}, err
		}

		return []string{availabilityZone}, nil
	}

	// Default option of cluster's nodes availability zones
	return cluster.Nodes().AvailabilityZones(), nil
}

func (r *ReplicaSizeValidation) MaxReplicaValidator() interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if r.MinReplicas > maxReplicas {
			return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
		}
		if r.MultiAz && maxReplicas%3 != 0 {
			return fmt.Errorf("multi AZ clusters require that the replicas be a multiple of 3")
		}
		return validateClusterVersionWithMaxNodesLimit(
			r.ClusterVersion, maxReplicas, r.IsHostedCp)
	}
}

func (r *ReplicaSizeValidation) MinReplicaValidator() interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if r.Autoscaling && minReplicas < 1 && r.IsHostedCp {
			return fmt.Errorf("min-replicas must be greater than zero")
		}
		if r.Autoscaling && minReplicas < 0 && !r.IsHostedCp {
			return fmt.Errorf("min-replicas must be a number that is 0 or greater when autoscaling is" +
				" enabled")
		}
		if !r.Autoscaling && minReplicas < 0 {
			return fmt.Errorf("replicas must be a non-negative integer")
		}
		if r.MultiAz && minReplicas%3 != 0 {
			return fmt.Errorf("multi AZ clusters require that the replicas be a multiple of 3")
		}
		return validateClusterVersionWithMaxNodesLimit(
			r.ClusterVersion, minReplicas, r.IsHostedCp)
	}
}

func spotMaxPriceValidator(val interface{}) error {
	spotMaxPrice := fmt.Sprintf("%v", val)
	if spotMaxPrice == "on-demand" {
		return nil
	}
	price, err := strconv.ParseFloat(spotMaxPrice, commonUtils.MaxByteSize)
	if err != nil {
		return fmt.Errorf("expected a numeric value for spot max price")
	}

	if price <= 0 {
		return fmt.Errorf("spot max price must be positive")
	}
	return nil
}

func getSubnetFromAvailabilityZone(cmd *cobra.Command, r *rosa.Runtime, isAvailabilityZoneSet bool,
	cluster *cmv1.Cluster, args *mpOpts.CreateMachinepoolUserOptions) (string, error) {

	privateSubnets, err := r.AWSClient.GetVPCPrivateSubnets(cluster.AWS().SubnetIDs()[0])
	if err != nil {
		return "", err
	}

	// Fetching the availability zones from the VPC private subnets
	subnetsMap := make(map[string][]string)
	for _, privateSubnet := range privateSubnets {
		subnetsPerAZ, exist := subnetsMap[*privateSubnet.AvailabilityZone]
		if !exist {
			subnetsPerAZ = []string{*privateSubnet.SubnetId}
		} else {
			subnetsPerAZ = append(subnetsPerAZ, *privateSubnet.SubnetId)
		}
		subnetsMap[*privateSubnet.AvailabilityZone] = subnetsPerAZ
	}
	availabilityZones := make([]string, 0)
	for availabilizyZone := range subnetsMap {
		availabilityZones = append(availabilityZones, availabilizyZone)
	}

	availabilityZone := cluster.Nodes().AvailabilityZones()[0]
	if !isAvailabilityZoneSet && interactive.Enabled() {
		availabilityZone, err = interactive.GetOption(interactive.Input{
			Question: "AWS availability zone",
			Help:     cmd.Flags().Lookup("availability-zone").Usage,
			Options:  availabilityZones,
			Default:  availabilityZone,
			Required: true,
		})
		if err != nil {
			return "", fmt.Errorf("expected a valid AWS availability zone: %s", err)
		}
	} else if isAvailabilityZoneSet {
		availabilityZone = args.AvailabilityZone
	}

	if subnets, ok := subnetsMap[availabilityZone]; ok {
		if len(subnets) == 1 {
			return subnets[0], nil
		}
		r.Reporter.Infof("There are several subnets for availability zone '%s'", availabilityZone)
		interactive.Enable()
		subnet, err := getSubnetFromUser(cmd, r, false, cluster, args)
		if err != nil {
			return "", err
		}
		return subnet, nil
	}

	return "", fmt.Errorf("failed to find a private subnet for '%s' availability zone", availabilityZone)
}

// temporary fn until calculated default values can be retrieved from single source of truth
func validateClusterVersionWithMaxNodesLimit(clusterVersion string, replicas int, isHostedCp bool) error {
	if isHostedCp {
		if replicas > hcpMaxNodesLimit {
			return fmt.Errorf("should provide an integer number less than or equal to '%v'", hcpMaxNodesLimit)
		}
		return nil
	}

	maxNodesLimit := 180
	classicMaxNodeSize249SupportedVersion, _ := versions.IsGreaterThanOrEqual(
		clusterVersion, ocm.ClassicMaxNodeSize249Support)
	// on error do nothing
	if classicMaxNodeSize249SupportedVersion {
		maxNodesLimit = 249
	}
	if replicas > maxNodesLimit {
		return fmt.Errorf("should provide an integer number less than or equal to '%v'", maxNodesLimit)
	}

	return nil
}

func (r *ReplicaSizeValidation) MinReplicaValidatorOnClusterCreate() interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}

		if r.IsHostedCp && minReplicas < 2 {
			return fmt.Errorf("hosted Control Plane clusters require a minimum of 2 nodes, "+
				"but %d was requested", minReplicas)
		}

		err = validateClusterVersionWithMaxNodesLimit(
			r.ClusterVersion, minReplicas, r.IsHostedCp)
		if err != nil {
			return err
		}

		return clustervalidations.MinReplicasValidator(
			minReplicas,
			r.MultiAz,
			r.IsHostedCp,
			r.PrivateSubnetsCount,
		)
	}
}

func (r *ReplicaSizeValidation) MaxReplicaValidatorOnClusterCreate() interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}

		err = validateClusterVersionWithMaxNodesLimit(
			r.ClusterVersion, maxReplicas, r.IsHostedCp)
		if err != nil {
			return err
		}

		return clustervalidations.MaxReplicasValidator(
			r.MinReplicas,
			maxReplicas,
			r.MultiAz,
			r.IsHostedCp,
			r.PrivateSubnetsCount,
		)
	}
}

func getZoneType(machinePool *cmv1.MachinePool) string {
	zones := machinePool.AvailabilityZones()
	if len(zones) == 0 {
		return zoneTypeNA
	}

	for _, zone := range zones {
		zoneLower := strings.ToLower(zone)

		if strings.Contains(zoneLower, "wl") {
			return zoneTypeWavelength
		}
		if strings.Contains(zoneLower, "outpost") {
			return zoneTypeOutpost
		}
		if strings.Contains(zoneLower, "-lz") || strings.Count(zoneLower, "-") > 3 {
			return zoneTypeLocalZone
		}
	}

	return zoneTypeStandard
}

func isWinLIEnabled(labels map[string]string) string {
	if val, ok := labels[labelImageType]; ok && strings.ToLower(val) == imageTypeWindows {
		return displayValueYes
	}
	return displayValueNo
}

func isDedicatedHost(machinePool *cmv1.MachinePool, runtime *rosa.Runtime) string {
	if machinePool == nil {
		return displayValueNo
	}

	awsConfig := machinePool.AWS()
	if awsConfig == nil {
		return displayValueNo
	}

	awsMachinePoolID := awsConfig.ID()
	if awsMachinePoolID == "" || runtime == nil || runtime.AWSClient == nil {
		return displayValueNo
	}

	hasDedicatedHost, err := runtime.AWSClient.CheckIfMachinePoolHasDedicatedHost([]string{awsMachinePoolID})
	if err != nil {
		_ = runtime.Reporter.Errorf("Failed to check dedicated host status: %v", err)
		return displayValueUnknown
	}

	if hasDedicatedHost {
		return displayValueYes
	}
	return displayValueNo
}

func prettyPrintImageTypes() string {
	s := "'"
	for i, t := range mpHelpers.ImageTypes {
		if i == 0 {
			s += t
		} else {
			s += "', '" + t
		}
	}
	return s + "'"
}
