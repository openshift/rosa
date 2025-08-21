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

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "ingresses",
	Aliases: []string{"route", "routes", "ingress"},
	Short:   "List cluster Ingresses",
	Long:    "List API and ingress endpoints for a cluster.",
	Example: `  # List all routes on a cluster named "mycluster"
  rosa list ingresses --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady &&
		cluster.State() != cmv1.ClusterStateHibernating {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load any existing ingresses for this cluster
	r.Reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := r.OCMClient.GetIngresses(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(ingresses)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(ingresses) == 0 {
		r.Reporter.Infof("There are no ingresses configured for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	headers := []string{"ID", "APPLICATION ROUTER", "PRIVATE", "DEFAULT", "ROUTE SELECTORS",
		"LB-TYPE", "EXCLUDED NAMESPACE", "WILDCARD POLICY", "NAMESPACE OWNERSHIP"}
	var tableData [][]string
	for _, ingress := range ingresses {
		row := []string{
			ingress.ID(),
			fmt.Sprintf("https://%s", ingress.DNSName()),
			isPrivate(ingress.Listening()),
			isDefault(ingress),
			printRouteSelectors(ingress),
			string(ingress.LoadBalancerType()),
			helper.SliceToSortedString(ingress.ExcludedNamespaces()),
			string(ingress.RouteWildcardPolicy()),
			string(ingress.RouteNamespaceOwnershipPolicy()),
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	output.BuildTable(writer, "\t", tableData)

	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}
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
