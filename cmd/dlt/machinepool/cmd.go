/*
Copyright (c) 2020 Red Hat, Inc.

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

package machinepool

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "machinepool ID"
	short   = "Delete machine pool"
	long    = "Delete the additional machine pool from a cluster."
	example = `  # Delete machine pool with ID mp-1 from a cluster named 'mycluster'
  rosa delete machinepool --cluster=mycluster mp-1`
)

var (
	aliases = []string{"machinepools", "machine-pool", "machine-pools"}
)

func NewDeleteMachinePoolCommand() *cobra.Command {
	options := NewDeleteMachinepoolUserOptions()
	var cmd = &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteMachinePoolRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(
		&options.machinepool,
		"machinepool",
		"",
		"Machine pool of the cluster to target",
	)

	ocm.AddClusterFlag(cmd)
	confirm.AddFlag(cmd.Flags())
	return cmd
}

func DeleteMachinePoolRunner(userOptions *DeleteMachinepoolUserOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		options := NewDeleteMachinepoolOptions()

		err := options.Bind(userOptions, argv)
		if err != nil {
			return err
		}

		clusterKey := runtime.GetClusterKey()
		cluster := runtime.FetchCluster()

		service := machinepool.NewMachinePoolService()
		err = service.DeleteMachinePool(runtime, options.Machinepool(), clusterKey, cluster)
		if err != nil {
			return fmt.Errorf("Error deleting machinepool: %v", err)
		}
		return nil
	}
}
