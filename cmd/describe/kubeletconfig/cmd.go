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

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "kubeletconfig",
	Aliases: []string{"kubelet-config"},
	Short:   "Show details of the custom kubeletconfig for a cluster",
	Long:    "Show details of the custom kubeletconfig for a cluster.",
	Example: `  # Describe the custom kubeletconfig for cluster 'foo'
  rosa describe kubeletconfig --cluster foo`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	r.Reporter.Debugf("Loading KubeletConfig for cluster '%s'", clusterKey)
	kubeletConfig, err := r.OCMClient.GetClusterKubeletConfig(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	if kubeletConfig == nil {
		r.Reporter.Infof("No custom KubeletConfig exists for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	if output.HasFlag() {
		err = output.Print(kubeletConfig)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	r.Reporter.Debugf("Printing KubeletConfig for cluster '%s'", clusterKey)
	// Prepare string
	kubeletConfigOutput := fmt.Sprintf("\n"+
		"Pod Pids Limit:                       %d\n",
		kubeletConfig.PodPidsLimit(),
	)
	fmt.Print(kubeletConfigOutput)
}
