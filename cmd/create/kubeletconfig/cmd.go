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
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "kubeletconfig"
	short   = "Create a custom kubeletconfig for a cluster"
	long    = short
	example = `  # Create a custom kubeletconfig with a pod-pids-limit of 5000
  rosa create kubeletconfig --cluster=mycluster --pod-pids-limit=5000
  `
)

func NewCreateKubeletConfigCommand() *cobra.Command {

	options := NewKubeletConfigOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"kubelet-config"},
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateKubeletConfigRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}

	options.AddAllFlags(cmd)
	ocm.AddClusterFlag(cmd)
	interactive.AddFlag(cmd.Flags())
	return cmd
}

func CreateKubeletConfigRunner(options *KubeletConfigOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {

		options.BindFromArgs(args)
		clusterKey := r.GetClusterKey()
		cluster, err := r.OCMClient.GetCluster(r.GetClusterKey(), r.Creator)
		if err != nil {
			return err
		}

		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("Cluster '%s' is not yet ready. Current state is '%s'", clusterKey, cluster.State())
		}

		if !cluster.Hypershift().Enabled() {
			// Classic clusters can only have a single KubeletConfig
			kubeletConfig, _, err := r.OCMClient.GetClusterKubeletConfig(cluster.ID())
			if err != nil {
				return fmt.Errorf("Failed getting KubeletConfig for cluster '%s': %s",
					r.ClusterKey, err)
			}

			if kubeletConfig != nil {
				return fmt.Errorf("A KubeletConfig for cluster '%s' already exists. "+
					"You should edit it via 'rosa edit kubeletconfig'", clusterKey)
			}
		} else {
			options.Name, err = PromptForName(options.Name)
			if err != nil {
				return err
			}

			err = options.ValidateForHypershift()
			if err != nil {
				return err
			}
		}

		options.PodPidsLimit, err = ValidateOrPromptForRequestedPidsLimit(options.PodPidsLimit, clusterKey, nil, r)
		if err != nil {
			return err
		}

		if !cluster.Hypershift().Enabled() {
			// Creating a KubeletConfig for a classic cluster must prompt the user, as the changes apply
			// immediately and cause reboots of the worker nodes in their cluster
			if !PromptUserToAcceptWorkerNodeReboot(OperationCreate, r) {
				return nil
			}
		}

		r.Reporter.Debugf("Creating KubeletConfig for cluster '%s'", clusterKey)
		kubeletConfigArgs := ocm.KubeletConfigArgs{PodPidsLimit: options.PodPidsLimit, Name: options.Name}

		_, err = r.OCMClient.CreateKubeletConfig(cluster.ID(), kubeletConfigArgs)
		if err != nil {
			return fmt.Errorf("Failed creating KubeletConfig for cluster '%s': '%s'",
				clusterKey, err)
		}
		r.Reporter.Infof("Successfully created KubeletConfig for cluster '%s'", clusterKey)
		return nil
	}
}
