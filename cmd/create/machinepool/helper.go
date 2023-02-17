package machinepool

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func getSubnetFromUser(cmd *cobra.Command, r *rosa.Runtime, isSubnetSet bool,
	cluster *cmv1.Cluster) string {
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
		subnet = aws.ParseSubnet(subnetOption)
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
		subnetOptions = append(subnetOptions, aws.SetSubnetOption(*subnet.SubnetId, *subnet.AvailabilityZone))
	}

	return subnetOptions, nil
}

func getTaints(cmd *cobra.Command, r *rosa.Runtime) []*cmv1.TaintBuilder {
	taints := args.taints
	if interactive.Enabled() {
		var err error
		taints, err = interactive.GetString(interactive.Input{
			Question: "Taints",
			Help:     cmd.Flags().Lookup("taints").Usage,
			Default:  taints,
			Validators: []interactive.Validator{
				taintValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	taintBuilders, err := parseTaints(taints)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	return taintBuilders
}

func getLabelMap(cmd *cobra.Command, r *rosa.Runtime) map[string]string {
	labels := args.labels
	if interactive.Enabled() {
		var err error
		labels, err = interactive.GetString(interactive.Input{
			Question: "Labels",
			Help:     cmd.Flags().Lookup("labels").Usage,
			Default:  labels,
			Validators: []interactive.Validator{
				LabelValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	labelMap, err := ParseLabels(labels)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	return labelMap
}
