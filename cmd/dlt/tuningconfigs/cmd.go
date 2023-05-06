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

	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "tuning-configs",
	Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
	Short:   "Delete tuning config",
	Long:    "Delete a tuning config for a cluster.",
	Example: `  # Delete tuning config with name tuned1 from a cluster named 'mycluster'
  rosa delete tuning-config --cluster=mycluster tuned1`,
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

	if confirm.Confirm("delete tuning config %s on cluster %s", tuningConfigName, clusterKey) {
		r.Reporter.Debugf("Deleting tuning config '%s' on cluster '%s'", tuningConfigName, clusterKey)
		err = r.OCMClient.DeleteTuningConfig(cluster.ID(), tuningConfig.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete tuning config '%s' on cluster '%s': %v",
				tuningConfigName, clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted tuning config '%s' from cluster '%s'", tuningConfigName, clusterKey)
	}
}
