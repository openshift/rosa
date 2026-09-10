package network

import (
	"fmt"
	"os"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	helper "github.com/openshift/rosa/pkg/network"
	opts "github.com/openshift/rosa/pkg/options/network"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfCreateNetwork are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled       = hyperfleet.Enabled
	hfExitFn        = func(code int) { os.Exit(code) }
	hfCreateNetwork = func(userOptions *opts.NetworkUserOptions, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		runHyperfleetCreateNetwork(r, userOptions, argv)
	}
)

// runHyperfleetCreateNetwork creates a VPC and private hosted zone for HCP clusters via CloudFormation.
// The hosted zone is named {cluster-name}.hypershift.local and is associated with the VPC.
func runHyperfleetCreateNetwork(r *rosa.Runtime, userOptions *opts.NetworkUserOptions, argv []string) {
	userOptions.CleanTemplateDir()

	// Parse parameters and tags
	parsedParams, parsedTags, err := helper.ParseParams(userOptions.Params)
	if err != nil {
		r.Reporter.Errorf("Failed to parse parameters: %v", err)
		hfExitFn(1)
		return
	}

	// Require ClusterName parameter for hosted zone creation
	clusterName := parsedParams["ClusterName"]
	if clusterName == "" {
		r.Reporter.Errorf("--param ClusterName=<name> is required for hyperfleet network creation")
		r.Reporter.Infof("The cluster name is used to create the private hosted zone: <cluster-name>.hypershift.local")
		hfExitFn(1)
		return
	}

	// Set default stack name if not provided
	if parsedParams["Name"] == "" {
		parsedParams["Name"] = fmt.Sprintf("rosa-hcp-network-%s", clusterName)
		r.Reporter.Infof("Stack name not provided, using default: %s", parsedParams["Name"])
	}

	// Set default region if not provided
	if parsedParams["Region"] == "" {
		parsedParams["Region"] = r.AWSClient.GetRegion()
		r.Reporter.Infof("Region not provided, using: %s", parsedParams["Region"])
	}

	// Add cluster tag to all resources
	if parsedTags == nil {
		parsedTags = make(map[string]string)
	}
	parsedTags[fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)] = "owned"

	// Get mode
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
		return
	}

	service := helper.NewNetworkService()

	switch mode {
	case interactive.ModeManual:
		r.Reporter.Infof(helper.ManualModeHelperMessage(parsedParams, parsedTags))
		r.Reporter.Infof("\nTo create the network stack manually, save the template below and use:")
		r.Reporter.Infof("  aws cloudformation create-stack --stack-name %s --template-body file://template.yaml --parameters ...", parsedParams["Name"])
		fmt.Println("\nCloudFormation Template:")
		fmt.Println("---")
		fmt.Println(CloudFormationHCPTemplateFile)
		return

	default:
		r.Reporter.Infof("Creating network stack for HCP cluster '%s'", clusterName)
		r.Reporter.Infof("  Stack Name: %s", parsedParams["Name"])
		r.Reporter.Infof("  Region: %s", parsedParams["Region"])
		r.Reporter.Infof("  Hosted Zone: %s.hypershift.local", clusterName)

		// Create stack using CloudFormation template
		templateFile := CloudFormationHCPTemplateFile
		templateBody := []byte{}
		err = service.CreateStack(&templateFile, &templateBody, parsedParams, parsedTags)
		if err != nil {
			r.Reporter.Errorf("Failed to create network stack: %v", err)
			hfExitFn(1)
			return
		}

		r.Reporter.Infof("Network stack created successfully")
		r.Reporter.Infof("Use the following to get stack outputs:")
		r.Reporter.Infof("  aws cloudformation describe-stacks --stack-name %s --query 'Stacks[0].Outputs'", parsedParams["Name"])
	}
}
