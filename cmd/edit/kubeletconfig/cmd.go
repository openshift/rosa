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
	"context"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "kubeletconfig"
	short   = "Edit a kubeletconfig for a cluster"
	long    = short
	example = `  # Edit a KubeletConfig to have a pod-pids-limit of 10000
  rosa edit kubeletconfig --cluster=mycluster --pod-pids-limit=10000
  # Edit a KubeletConfig named 'bar' to have a pod-pids-limit of 10000
  rosa edit kubeletconfig --cluster=mycluster --name=bar --pod-pids-limit=10000
  `
	kubeletNotExistingMessage = "The specified KubeletConfig does not exist for cluster '%s'." +
		" You should first create it via 'rosa create kubeletconfig'"
)

var aliases = []string{"kubelet-config"}

func NewEditKubeletConfigCommand() *cobra.Command {

	options := NewKubeletConfigOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), EditKubeletConfigRunner(options)),
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(cmd)
	interactive.AddFlag(flags)
	options.AddFlagsToCommand(cmd)
	return cmd
}

func EditKubeletConfigRunner(options *KubeletConfigOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {
		cluster, err := r.OCMClient.GetCluster(r.GetClusterKey(), r.Creator)
		if err != nil {
			return err
		}

		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", r.GetClusterKey(), cluster.State())
		}

		var kubeletconfig *cmv1.KubeletConfig
		var exists bool

		if cluster.Hypershift().Enabled() {
			options.Name, err = PromptForName(options.Name)
			if err != nil {
				return err
			}

			options.ValidateForHypershift()
			kubeletconfig, exists, err = r.OCMClient.FindKubeletConfigByName(ctx, cluster.ID(), options.Name)
		} else {
			kubeletconfig, exists, err = r.OCMClient.GetClusterKubeletConfig(cluster.ID())
		}

		if err != nil {
			return fmt.Errorf("Failed to fetch KubeletConfig configuration for cluster '%s': %s",
				r.GetClusterKey(), err)
		}

		if !exists {
			return fmt.Errorf(kubeletNotExistingMessage, r.GetClusterKey())
		}

		requestedPids, err := ValidateOrPromptForRequestedPidsLimit(options.PodPidsLimit, r.GetClusterKey(), kubeletconfig, r)
		if err != nil {
			return err
		}

		if !cluster.Hypershift().Enabled() {
			// Classic clusters must prompt the user as edit will cause all worker nodes to reboot
			prompt := fmt.Sprintf("Updating the custom KubeletConfig for cluster '%s' will cause all non-Control Plane "+
				"nodes to reboot. This may cause outages to your applications. Do you wish to continue?", r.GetClusterKey())

			if !confirm.ConfirmRaw(prompt) {
				r.Reporter.Infof("Update of custom KubeletConfig for cluster '%s' aborted.", r.GetClusterKey())
				return nil
			}
		}

		r.Reporter.Debugf("Updating KubeletConfig '%s' for cluster '%s'", kubeletconfig.ID(), r.GetClusterKey())
		_, err = r.OCMClient.UpdateKubeletConfig(
			ctx, cluster.ID(), kubeletconfig.ID(), ocm.KubeletConfigArgs{PodPidsLimit: requestedPids, Name: options.Name})

		if err != nil {
			return fmt.Errorf("Failed to update KubeletConfig for cluster '%s': %s",
				r.GetClusterKey(), err)
		}

		r.Reporter.Infof("Successfully updated KubeletConfig for cluster '%s'", r.GetClusterKey())
		return nil
	}
}
