package machinepool

import (
	"fmt"
	"os"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

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
			return "", fmt.Errorf(questionError)
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
			return "", fmt.Errorf("Expected a valid AWS subnet: %s", err)
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
		return []string{}, fmt.Errorf("Expected cluster's subnets to contain subnets IDs, but got an empty list")
	}

	availableSubnets, err := r.AWSClient.GetVPCSubnets(cluster.AWS().SubnetIDs()[0])
	if err != nil {
		return []string{}, fmt.Errorf("Failed to retrieve available subnets: %v", err)
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
) *cmv1.AWSNodePoolBuilder {
	awsNpBuilder := cmv1.NewAWSNodePool().InstanceType(instanceType)

	if len(securityGroupIds) > 0 {
		awsNpBuilder.AdditionalSecurityGroupIds(securityGroupIds...)
	}

	if len(awsTags) > 0 {
		awsNpBuilder.Tags(awsTags)
	}

	awsNpBuilder.Ec2MetadataHttpTokens(cmv1.Ec2MetadataHttpTokens(httpTokens))

	return awsNpBuilder
}

func getVpcIdFromSubnet(subnet ec2types.Subnet) (string, error) {
	vpcId := awssdk.ToString(subnet.VpcId)
	if vpcId == "" {
		return "", fmt.Errorf("Unexpected situation a VPC ID should have been selected based on chosen subnets")
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

func minReplicaValidator(multiAZMachinePool bool) interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas < 0 {
			return fmt.Errorf("min-replicas must be a non-negative integer")
		}
		if multiAZMachinePool && minReplicas%3 != 0 {
			return fmt.Errorf("Multi AZ clusters require that the replicas be a multiple of 3")
		}
		return nil
	}
}

func maxReplicaValidator(minReplicas int, multiAZMachinePool bool) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas > maxReplicas {
			return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
		}
		if multiAZMachinePool && maxReplicas%3 != 0 {
			return fmt.Errorf("Multi AZ clusters require that the replicas be a multiple of 3")
		}
		return nil
	}
}

func spotMaxPriceValidator(val interface{}) error {
	spotMaxPrice := fmt.Sprintf("%v", val)
	if spotMaxPrice == "on-demand" {
		return nil
	}
	price, err := strconv.ParseFloat(spotMaxPrice, commonUtils.MaxByteSize)
	if err != nil {
		return fmt.Errorf("Expected a numeric value for spot max price")
	}

	if price <= 0 {
		return fmt.Errorf("Spot max price must be positive")
	}
	return nil
}

func getSubnetFromAvailabilityZone(cmd *cobra.Command, r *rosa.Runtime, isAvailabilityZoneSet bool,
	cluster *cmv1.Cluster, args MachinePoolArgs) (string, error) {

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
			return "", fmt.Errorf("Expected a valid AWS availability zone: %s", err)
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
		subnet := getSubnetFromUser(cmd, r, false, cluster, args)
		return subnet, nil
	}

	return "", fmt.Errorf("Failed to find a private subnet for '%s' availability zone", availabilityZone)
}
