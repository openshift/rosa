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

package tuningconfigs

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "tuning-configs",
	Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
	Short:   "List tuning configs",
	Long:    "List tuning configuration resources for a cluster.",
	Example: `  # List all tuning configuration for a cluster named 'mycluster'"
  rosa list tuning-configs -c mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	input.CheckIfHypershiftClusterOrExit(r, cluster)

	// Load any existing tuning configs for this cluster
	r.Reporter.Debugf("Loading tuning configs for cluster '%s'", clusterKey)
	tuningConfigs, err := r.OCMClient.GetTuningConfigs(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get tuning configs for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(tuningConfigs)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(tuningConfigs) == 0 {
		r.Reporter.Infof("There are no tuning configs for this cluster.")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tNAME\n")
	for _, tuningConfig := range tuningConfigs {
		fmt.Fprintf(writer, "%s\t%s\n",
			tuningConfig.ID(),
			tuningConfig.Name(),
		)
	}
	writer.Flush()
}
