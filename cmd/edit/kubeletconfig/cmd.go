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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "kubeletconfig",
	Aliases: []string{"kubelet-config"},
	Short:   "Edit the custom kubeletconfig for a cluster",
	Long:    "Edit the custom kubeletconfig for a cluster.",
	Example: `  # Edit a custom kubeletconfig to have a pod-pids-limit of 10000
  rosa edit kubeletconfig --cluster=mycluster --pod-pids-limit=10000
  `,
	Run: run,
}

var args struct {
	podPidsLimit int
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(flags)

	flags.IntVar(
		&args.podPidsLimit,
		PodPidsLimitOption,
		PodPidsLimitOptionDefaultValue,
		PodPidsLimitOptionUsage)

}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if cluster.Hypershift().Enabled() {
		r.Reporter.Errorf("Hosted Control Plane clusters do not support KubeletConfig configuration")
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		os.Exit(1)
	}

	kubeletconfig, err := r.OCMClient.GetClusterKubeletConfig(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to fetch existing KubeletConfig configuration for cluster '%s': %s",
			clusterKey, err)
		os.Exit(1)
	}

	if kubeletconfig == nil {
		r.Reporter.Errorf("No KubeletConfig for cluster '%s' has been found. "+
			"You should first create it via 'rosa create kubeletconfig'", clusterKey)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating KubeletConfig for cluster '%s'", clusterKey)

	requestedPids, err := ValidateOrPromptForRequestedPidsLimit(args.podPidsLimit, clusterKey, kubeletconfig, r)
	if err != nil {
		os.Exit(1)
	}

	prompt := fmt.Sprintf("Updating the custom KubeletConfig for cluster '%s' will cause all non-Control Plane "+
		"nodes to reboot. This may cause outages to your applications. Do you wish to continue?", clusterKey)

	if confirm.ConfirmRaw(prompt) {
		r.Reporter.Debugf("Updating KubeletConfig for cluster '%s'", clusterKey)
		_, err = r.OCMClient.UpdateKubeletConfig(cluster.ID(), ocm.KubeletConfigArgs{PodPidsLimit: requestedPids})
		if err != nil {
			r.Reporter.Errorf("Failed creating custom KubeletConfig for cluster '%s': %s",
				cluster.ID(), err)
			os.Exit(1)
		}

		r.Reporter.Infof("Successfully updated custom KubeletConfig for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	r.Reporter.Infof("Update of custom KubeletConfig for cluster '%s' aborted.", clusterKey)
}
