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

package kubeletconfig

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "kubeletconfig",
	Aliases: []string{"kubelet-config"},
	Short:   "Delete the custom kubeletconfig for a cluster",
	Long:    "Delete the custom kubeletconfig for a cluster",
	Example: `  # Delete the custom kubeletconfig for cluster 'foo'
  rosa delete kubeletconfig --cluster foo`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(Cmd.Flags())
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	r.Reporter.Debugf("Deleting KubeletConfig for cluster '%s'", clusterKey)

	prompt := fmt.Sprintf("Deleting the custom KubeletConfig for cluster '%s' will cause all non-Control Plane "+
		"nodes to reboot. This may cause outages to your applications. Do you wish to continue?", clusterKey)

	if confirm.ConfirmRaw(prompt) {

		err := r.OCMClient.DeleteKubeletConfig(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete custom KubeletConfig for cluster '%s': '%s'",
				clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted custom KubeletConfig for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	r.Reporter.Infof("Delete of custom KubeletConfig for cluster '%s' aborted.", clusterKey)
}
