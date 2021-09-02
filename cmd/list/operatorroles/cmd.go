/*
Copyright (c) 2021 Red Hat, Inc.
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

package operatorroles

import (
	"fmt"
	"github.com/briandowns/spinner"
	"os"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"text/tabwriter"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operator-roles", "operatorroles"},
	Short:   "List operator roles and policies",
	Long:    "List operator roles and policies for STS cluster(s)",
	Example: `  # List operator roles and policies for all clusters
  rosa list operator-roles
  # List operator roles with a specific cluster
  rosa list operator-roles -c mycluster`,

	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the operator roles for.",
	)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	var cluster *cmv1.Cluster
	if clusterKey != "" {
		if !ocm.IsValidClusterKey(clusterKey) {
			reporter.Errorf(
				"Cluster name, identifier or external identifier '%s' isn't valid: it "+
					"must contain only letters, digits, dashes and underscores",
				clusterKey,
			)
			os.Exit(1)
		}

		// Try to find the cluster:
		reporter.Debugf("Loading cluster '%s'", clusterKey)
		cluster, err = ocmClient.GetClusterOrArchived(clusterKey, awsCreator)
		if err != nil {
			fmt.Println(err.Error())
			reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		if cluster.AWS().STS().RoleARN() == "" {
			reporter.Errorf("Cluster '%s' is not an STS cluster, no associated operator roles",
				clusterKey)
			os.Exit(1)
		}
	}

	// get all active clusters
	activeClusters, err := ocmClient.GetClusters(awsCreator, 1000)
	if err != nil {
		reporter.Errorf("Failed to list clusters : '%v'", err)
		os.Exit(1)
	}

	// get all archived clusters
	archivedClusters, err := ocmClient.ListAccountArchivedClusters(awsCreator.AccountID)
	if err != nil {
		reporter.Errorf("Failed to list archived clusters : '%v'", err)
		os.Exit(1)
	}

	// print message to indicate there may be some wait time
	totalClusters := len(activeClusters) + len(archivedClusters)
	if clusterKey != "" {
		reporter.Infof("Fetching roles for cluster '%s'", clusterKey)
	} else if totalClusters == 0 {
		reporter.Infof("No active or archived clusters in current environment")
		os.Exit(0)
	} else {
		reporter.Infof("Fetching roles for %d active clusters and %d archived clusters",
			len(activeClusters), len(archivedClusters))
	}

	// start spinner
	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		spin.Start()
	}

	// fetch operator roles
	operatorRoles, err := awsClient.ListOperatorRoles(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get operator roles : %v", err)
		os.Exit(1)
	}

	// ensure roles found are for clusters in the current environment
	var validOperatorRoles []aws.OperatorRole
	if clusterKey != "" {
		validOperatorRoles = operatorRoles
	} else {
		for _, role := range operatorRoles {
			for _, cluster := range activeClusters {
				if cluster.ID() == role.ClusterID {
					validOperatorRoles = append(validOperatorRoles, role)
				}
			}
			for _, cluster := range archivedClusters {
				if cluster.ID() == role.ClusterID {
					validOperatorRoles = append(validOperatorRoles, role)
				}
			}
		}
	}

	if len(validOperatorRoles) == 0 {
		reporter.Infof("No operator roles available")
		os.Exit(1)
	}

	// stop the spinner before printing output
	spin.Stop()

	if output.HasFlag() {
		err = output.Print(validOperatorRoles)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "CLUSTER\tOPERATOR\tROLE\tACTIVE\n")
	for _, operatorRole := range validOperatorRoles {
		// see if operator role belongs to an active cluster
		for _, cluster := range activeClusters {
			if operatorRole.ClusterID == cluster.ID() {
				operatorRole.Active = "yes"
			} else {
				operatorRole.Active = "no"
			}
		}
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			operatorRole.ClusterID,
			operatorRole.OperatorNamespace,
			operatorRole.Role.RoleName,
			operatorRole.Active,
		)
	}
	writer.Flush()
}
