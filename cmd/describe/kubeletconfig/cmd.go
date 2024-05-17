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

	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "kubeletconfig"
	short   = "Show details of a kubeletconfig for a cluster"
	long    = short
	example = `  # Describe the custom kubeletconfig for ROSA Classic cluster 'foo'
  rosa describe kubeletconfig --cluster foo
  # Describe the custom kubeletconfig named 'bar' for cluster 'foo'
  rosa describe kubeletconfig --cluster foo --name bar
`
)

func NewDescribeKubeletConfigCommand() *cobra.Command {
	options := NewKubeletConfigOptions()

	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"kubelet-config"},
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeKubeletConfigRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}

	ocm.AddClusterFlag(cmd)
	output.AddFlag(cmd)
	options.AddNameFlag(cmd)

	return cmd
}

func DescribeKubeletConfigRunner(options *KubeletConfigOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {

		options.BindFromArgs(args)
		cluster, err := r.OCMClient.GetCluster(r.GetClusterKey(), r.Creator)
		if err != nil {
			return err
		}

		r.Reporter.Debugf("Loading KubeletConfig for cluster '%s'", r.GetClusterKey())
		var kubeletconfig *cmv1.KubeletConfig
		var exists bool

		if cluster.Hypershift().Enabled() {
			options.Name, err = PromptForName(options.Name)
			if err != nil {
				return err
			}

			err = options.ValidateForHypershift()
			if err != nil {
				return err
			}
			kubeletconfig, exists, err = r.OCMClient.FindKubeletConfigByName(ctx, cluster.ID(), options.Name)
		} else {
			// Name isn't required for Classic clusters, but for correctness, if the user has set it, lets check things
			if options.Name != "" {
				kubeletconfig, exists, err = r.OCMClient.FindKubeletConfigByName(ctx, cluster.ID(), options.Name)
			} else {
				kubeletconfig, exists, err = r.OCMClient.GetClusterKubeletConfig(cluster.ID())
			}
		}

		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("The KubeletConfig specified does not exist for cluster '%s'", r.GetClusterKey())
		}

		if output.HasFlag() {
			return output.Print(kubeletconfig)
		}

		if cluster.Hypershift().Enabled() {
			nodePools, err := r.OCMClient.FindNodePoolsUsingKubeletConfig(cluster.ID(), options.Name)
			if err != nil {
				return err
			}
			fmt.Print(PrintKubeletConfigForHcp(kubeletconfig, nodePools))

		} else {
			fmt.Print(PrintKubeletConfigForClassic(kubeletconfig))
		}
		return nil
	}
}
