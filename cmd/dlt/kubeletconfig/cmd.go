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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "kubeletconfig"
	short   = "Delete a kubeletconfig from a cluster"
	long    = short
	example = `  # Delete the KubeletConfig for ROSA Classic cluster 'foo'
  rosa delete kubeletconfig --cluster foo
  # Delete the KubeletConfig named 'bar' from cluster 'foo'
  rosa delete kubeletconfig --cluster foo --name bar
`
)

var aliases = []string{"kubelet-config"}

func NewDeleteKubeletConfigCommand() *cobra.Command {

	options := NewKubeletConfigOptions()

	var cmd = &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteKubeletConfigRunner(options)),
		Args:    cobra.NoArgs,
	}
	ocm.AddClusterFlag(cmd)
	confirm.AddFlag(cmd.Flags())
	options.AddNameFlag(cmd)
	return cmd
}

func DeleteKubeletConfigRunner(options *KubeletConfigOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {

		cluster, err := r.OCMClient.GetCluster(r.GetClusterKey(), r.Creator)
		if err != nil {
			return err
		}

		if !cluster.Hypershift().Enabled() {
			if !PromptUserToAcceptWorkerNodeReboot(OperationDelete, r) {
				return nil
			}
		}

		if cluster.Hypershift().Enabled() {
			options.Name, err = PromptForName(options.Name)
			if err != nil {
				return err
			}
			err = options.ValidateForHypershift()
			if err != nil {
				return err
			}
			err = r.OCMClient.DeleteKubeletConfigByName(ctx, cluster.ID(), options.Name)
		} else {
			err = r.OCMClient.DeleteKubeletConfig(ctx, cluster.ID())
		}

		if err != nil {
			return fmt.Errorf("Failed to delete KubeletConfig for cluster '%s': '%s'",
				r.GetClusterKey(), err)
		}

		r.Reporter.Infof("Successfully deleted KubeletConfig for cluster '%s'", r.GetClusterKey())
		return nil
	}
}
