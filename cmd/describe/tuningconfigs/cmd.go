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
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "tuning-configs",
	Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
	Short:   "Show details of tuning config",
	Long:    "Show details of a tuning config for a cluster.",
	Example: `  # Describe the 'tuned1' tuned config on cluster 'foo'
  rosa describe tuning-config --cluster foo tuned1`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the name of the tuned config",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	tuningConfigName := argv[0]

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	input.CheckIfHypershiftClusterOrExit(r, cluster)

	// Try to find the tuning config:
	r.Reporter.Debugf("Loading tuning configs for cluster '%s'", clusterKey)
	tuningConfig, err := r.OCMClient.FindTuningConfigByName(cluster.ID(), tuningConfigName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(tuningConfig)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Pretty print the spec
	tuningConfigSpec, err := json.MarshalIndent(tuningConfig.Spec(), "                            ", "  ")
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Describing tuning config '%s' on cluster '%s'", tuningConfig.Name(), clusterKey)
	// Prepare string
	tuningConfigOutput := fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"Spec:                       %s\n",
		tuningConfig.Name(), tuningConfig.ID(), tuningConfigSpec,
	)
	fmt.Print(tuningConfigOutput)
}
