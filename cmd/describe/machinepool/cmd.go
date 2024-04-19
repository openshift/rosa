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

package machinepool

import (
	"context"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "machinepool"
	alias   = "machine-pool"
	short   = "Show details of a machine pool on a cluster"
	long    = "Show details of a machine pool on a cluster."
	example = `  # Show details of a machine pool named "mymachinepool"" on a cluster named "mycluster"
  rosa describe machinepool --cluster=mycluster --machinepool=mymachinepool`
)

func NewDescribeMachinePoolCommand() *cobra.Command {
	options := NewDescribeMachinepoolUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: []string{alias},
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeMachinePoolRunner(options)),
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.machinepool,
		"machinepool",
		"",
		"Machine pool of the cluster to target",
	)

	output.AddFlag(cmd)
	ocm.AddClusterFlag(cmd)
	return cmd
}

func DescribeMachinePoolRunner(userOptions DescribeMachinepoolUserOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		options := NewDescribeMachinepoolOptions()
		// Allow the use also directly the machine pool id as positional parameter
		if len(argv) == 1 && !cmd.Flag("machinepool").Changed {
			userOptions.machinepool = argv[0]
		} else {
			err := cmd.ParseFlags(argv)
			if err != nil {
				return fmt.Errorf("unable to parse flags: %s", err)
			}
		}
		err := options.Bind(userOptions)
		if err != nil {
			return err
		}
		clusterKey := runtime.GetClusterKey()
		cluster := runtime.FetchCluster()
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}
		isHypershift := cluster.Hypershift().Enabled()

		service := machinepool.NewMachinePoolService()

		return service.DescribeMachinePool(runtime, cluster, clusterKey, isHypershift, options.Machinepool())
	}
}
