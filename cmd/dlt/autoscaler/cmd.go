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

package autoscaler

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "autoscaler",
	Short: "Delete autoscaler for cluster",
	Long:  "Delete autoscaler configuration for a given cluster.",
	Example: `  # Delete the autoscaler config for cluster named "mycluster"
  rosa delete autoscaler --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(Cmd.Flags())
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if cluster.Hypershift().Enabled() {
		r.Reporter.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
		os.Exit(1)
	}

	if !confirm.Confirm("delete cluster autoscaler?") {
		os.Exit(0)
	}

	r.Reporter.Debugf("Deleting autoscaler for cluster '%s''", clusterKey)

	err := r.OCMClient.DeleteClusterAutoscaler(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to delete autoscaler configuration for cluster '%s': %s",
			cluster.ID(), err)
		os.Exit(1)
	}
	r.Reporter.Infof("Successfully deleted autoscaler configuration for cluster '%s'", cluster.ID())
}
