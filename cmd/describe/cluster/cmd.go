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
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/moactl/pkg/aws"
	clusterprovider "github.com/openshift/moactl/pkg/cluster"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/ocm/properties"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:   "cluster [ID|NAME]",
	Short: "Show details of a cluster",
	Long:  "Show details of a cluster",
	Example: `  # Describe a cluster named "mycluster"
  moactl describe cluster mycluster

  # Describe a cluster using the --cluster flag
  moactl describe cluster --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to describe.",
	)
}

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check command line arguments:
	clusterKey := args.clusterKey
	if clusterKey == "" {
		if len(argv) != 1 {
			reporter.Errorf(
				"Expected exactly one command line argument or flag containing the name " +
					"or identifier of the cluster",
			)
			os.Exit(1)
		}
		clusterKey = argv[0]
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if !clusterprovider.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

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

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := clusterprovider.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("Failed to get cluster '%s': %v", clusterKey, err))
		os.Exit(1)
	}

	creatorARN, err := arn.Parse(cluster.Properties()[properties.CreatorARN])
	if err != nil {
		reporter.Errorf("Failed to parse creator ARN for cluster '%s'", clusterKey)
		os.Exit(1)
	}
	phase := ""

	if cluster.State() == cmv1.ClusterStatePending {
		phase = "(Preparing account)"
	}

	if cluster.State() == cmv1.ClusterStateInstalling {
		if !cluster.Status().DNSReady() {
			phase = "(DNS setup in progress)"
		}
		if cluster.Status().ProvisionErrorMessage() != "" {
			errorCode := ""
			if cluster.Status().ProvisionErrorCode() != "" {
				errorCode = cluster.Status().ProvisionErrorCode() + " - "
			}
			phase = "(" + errorCode + "Install is taking longer than expected)"
		}
	}

	// Print short cluster description:
	str := fmt.Sprintf(""+
		"Name:                      %s\n"+
		"ID:                        %s\n"+
		"External ID:               %s\n"+
		"AWS Account:               %s\n"+
		"API URL:                   %s\n"+
		"Console URL:               %s\n"+
		"Nodes:                     Master: %d, Infra: %d, Compute: %d\n"+
		"Region:                    %s\n"+
		"State:                     %s %s\n"+
		"Channel Group:             %s\n"+
		"Created:                   %s\n",
		cluster.Name(),
		cluster.ID(),
		cluster.ExternalID(),
		creatorARN.AccountID,
		cluster.API().URL(),
		cluster.Console().URL(),
		cluster.Nodes().Master(), cluster.Nodes().Infra(), cluster.Nodes().Compute(),
		cluster.Region().ID(),
		cluster.State(), phase,
		cluster.Version().ChannelGroup(),
		cluster.CreationTimestamp().Format("Jan _2 2006 15:04:05 MST"),
	)

	if cluster.Status().State() == cmv1.ClusterStateError {
		str = fmt.Sprintf("%s"+
			"Provisioning Error Type:   %s\n"+
			"Provisioning Error Reason: %s\n",
			str,
			cluster.Status().ProvisionErrorType(),
			cluster.Status().ProvisionErrorReason(),
		)
	}
	// Print short cluster description:
	fmt.Print(str)
	fmt.Println()
}
