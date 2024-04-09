package machinepool

import (
	"fmt"
	"os"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/rosa"
)

func getSubnetFromUser(cmd *cobra.Command, r *rosa.Runtime, isSubnetSet bool, cluster *cmv1.Cluster) string {
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
			r.Reporter.Errorf(questionError)
			os.Exit(1)
		}
	} else {
		subnet = args.subnet
	}

	if selectSubnet {
		subnetOptions, err := getSubnetOptions(r, cluster)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		subnetOption, err := interactive.GetOption(interactive.Input{
			Question: "Subnet ID",
			Help:     cmd.Flags().Lookup("subnet").Usage,
			Options:  subnetOptions,
			Default:  subnetOptions[0],
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid AWS subnet: %s", err)
			os.Exit(1)
		}
		subnet = aws.ParseOption(subnetOption)
	}

	return subnet
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

func getVpcIdFromSubnet(subnet ec2types.Subnet) (string, error) {
	vpcId := awssdk.ToString(subnet.VpcId)
	if vpcId == "" {
		return "", fmt.Errorf("Unexpected situation a VPC ID should have been selected based on chosen subnets")
	}

	return vpcId, nil
}
