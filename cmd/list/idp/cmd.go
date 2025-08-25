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

package idp

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "idps",
	Aliases: []string{"idp"},
	Short:   "List cluster IDPs",
	Long:    "List identity providers for a cluster.",
	Example: `  # List all identity providers on a cluster named "mycluster"
  rosa list idps --cluster=mycluster`,
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

	if cluster.ExternalAuthConfig().Enabled() {
		r.Reporter.Errorf("Listing identity providers is not supported for clusters with external authentication configured.")
		os.Exit(1)
	}

	// Load any existing IDPs for this cluster
	r.Reporter.Debugf("Loading identity providers for cluster '%s'", clusterKey)
	idps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(idps)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(idps) == 0 {
		r.Reporter.Infof("There are no identity providers configured for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	includeAuthURL := !(len(idps) == 1 && !ocm.HasAuthURLSupport(idps[0]))

	headers := []string{"NAME", "TYPE"}
	if includeAuthURL {
		headers = append(headers, "AUTH URL")
	}

	var tableData [][]string
	for _, idp := range idps {
		row := []string{
			idp.Name(),
			ocm.IdentityProviderType(idp),
		}

		if includeAuthURL {
			oauthURL, err := ocm.GetOAuthURL(cluster, idp)
			if err != nil {
				r.Reporter.Warnf("Error building OAuth URL for %s: %v", idp.Name(), err)
			}
			row = append(row, oauthURL)
		}
		tableData = append(tableData, row)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if output.ShouldHideEmptyColumns() {
		newHeaders, newData := output.RemoveEmptyColumns(headers, tableData)

		config := output.TableConfig{
			Separator:            "\t\t",
			HasTrailingSeparator: false,
			UseFprintln:          false,
		}
		output.PrintTable(writer, newHeaders, newData, config)
	} else {
		if includeAuthURL {
			fmt.Fprintf(writer, "NAME\t\tTYPE\t\tAUTH URL\n")
			for _, row := range tableData {
				fmt.Fprintf(writer, "%s\n", strings.Join(row, "\t"))
			}
		} else {
			fmt.Fprintf(writer, "NAME\t\tTYPE\n")
			for _, row := range tableData {
				fmt.Fprintf(writer, "%s\n", strings.Join(row, "\t"))
			}
		}
	}
	writer.Flush()
}
