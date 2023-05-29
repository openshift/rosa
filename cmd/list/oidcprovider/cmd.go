/*
Copyright (c) 2023 Red Hat, Inc.

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

package oidcprovider

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "oidc-providers",
	Aliases: []string{"oidcprovider", "oidc-provider", "oidcproviders"},
	Short:   "List OIDC providers",
	Long:    "List OIDC providers for the current AWS account.",
	Example: `  # List all oidc providers
  rosa list oidc-providers`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	output.AddFlag(Cmd)
	ocm.AddOptionalClusterFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS()
	defer r.Cleanup()

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching OIDC providers")
		spin.Start()
	}

	clusterId := ""
	if cmd.Flags().Changed("cluster") {
		clusterKey := r.GetClusterKey()
		clusterId = clusterKey
	}

	providers, err := r.AWSClient.ListOidcProviders(clusterId)
	if spin != nil {
		spin.Stop()
	}
	if err != nil {
		r.Reporter.Errorf("Failed to get OIDC providers: %v", err)
		os.Exit(1)
	}

	if len(providers) == 0 {
		r.Reporter.Infof("No OIDC providers available")
		os.Exit(0)
	}
	if output.HasFlag() {
		outList := []map[string]interface{}{}
		for _, provider := range providers {
			outList = append(outList, map[string]interface{}{"arn": provider.Arn, "cluster_id": provider.ClusterId})
		}
		err = output.Print(outList)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "OIDC PROVIDER ARN\tCluster ID\n")
	for _, oidcProvider := range providers {
		fmt.Fprintf(
			writer,
			"%s\t%s\n",
			oidcProvider.Arn,
			oidcProvider.ClusterId,
		)
	}
	writer.Flush()
}
