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

package ingress

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "ingresses",
	Aliases: []string{"route", "routes", "ingress"},
	Short:   "List cluster Ingresses",
	Long:    "List API and ingress endpoints for a cluster.",
	Example: `  # List all routes on a cluster named "mycluster"
  rosa list ingresses --cluster=mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
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

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load any existing ingresses for this cluster
	reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := ocmClient.GetIngresses(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(ingresses)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(ingresses) == 0 {
		reporter.Infof("There are no ingresses configured for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tAPPLICATION ROUTER\t\t\tPRIVATE\t\tDEFAULT\t\tROUTE SELECTORS\n")
	for _, ingress := range ingresses {
		fmt.Fprintf(writer, "%s\thttps://%s\t\t\t%s\t\t%s\t\t%s\n",
			ingress.ID(),
			ingress.DNSName(),
			isPrivate(ingress.Listening()),
			isDefault(ingress),
			printRouteSelectors(ingress),
		)
	}
	writer.Flush()
}

func isPrivate(listeningMethod cmv1.ListeningMethod) string {
	if listeningMethod == cmv1.ListeningMethodInternal {
		return "yes"
	}
	return "no"
}

func isDefault(ingress *cmv1.Ingress) string {
	if ingress.Default() {
		return "yes"
	}
	return "no"
}

func printRouteSelectors(ingress *cmv1.Ingress) string {
	routeSelectors := ingress.RouteSelectors()
	if len(routeSelectors) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range routeSelectors {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}
