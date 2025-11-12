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

package addon

import (
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "addons",
	Aliases: []string{"addon", "add-ons", "add-on"},
	Short:   "List add-on installations",
	Long:    "List add-ons installed on a cluster.",
	Example: `  # List all add-on installations on a cluster named "mycluster"
  rosa list addons --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the add-ons of (required).",
	)

	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

// When no specific cluster id is provided by the user, this function lists all available AddOns
func listAllAddOns(r *rosa.Runtime) {
	r.Reporter.Debugf("Fetching all available add-ons")
	addOnResources, err := r.OCMClient.GetAvailableAddOns()
	if err != nil {
		r.Reporter.Errorf("Failed to fetch add-ons: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(addOnResources)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(addOnResources) == 0 {
		r.Reporter.Infof("There are no add-ons available")
		os.Exit(0)
	}

	headers := []string{"ID", "NAME", "AVAILABILITY"}
	var tableData [][]string
	for _, addOnResource := range addOnResources {
		availability := "unavailable"
		if addOnResource.Available {
			availability = "available"
		}
		row := []string{
			addOnResource.AddOn.ID(),
			addOnResource.AddOn.Name(),
			availability,
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t\t", tableData)

	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// When the user specifies a clusterKey, this function lists the AddOns for that cluster
func listClusterAddOns(clusterKey string, r *rosa.Runtime) {

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	ocm.SetClusterKey(clusterKey)
	clusterKey = r.GetClusterKey()

	if !ocm.IsValidClusterKey(clusterKey) {
		r.Reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady &&
		cluster.State() != cmv1.ClusterStateHibernating {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load any existing Add-Ons for this cluster
	r.Reporter.Debugf("Loading add-ons installations for cluster '%s'", clusterKey)
	clusterAddOns, err := r.OCMClient.GetClusterAddOns(cluster)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-ons for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(clusterAddOns)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(clusterAddOns) == 0 {
		r.Reporter.Infof("There are no add-ons installed on cluster '%s'", clusterKey)
		os.Exit(0)
	}

	headers := []string{"ID", "NAME", "STATE"}

	var tableData [][]string
	for _, clusterAddOn := range clusterAddOns {
		row := []string{
			clusterAddOn.ID,
			clusterAddOn.Name,
			clusterAddOn.State,
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t\t", tableData)

	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if args.clusterKey == "" {
		listAllAddOns(r)
	} else {
		listClusterAddOns(args.clusterKey, r)
	}
}
