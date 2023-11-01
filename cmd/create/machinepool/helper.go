package machinepool

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
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
		s := subnet
		subnetOptions = append(subnetOptions, aws.SetSubnetOption(&s))
	}

	return subnetOptions, nil
}
