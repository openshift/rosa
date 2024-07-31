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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/machinepool"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
)

type CreateMachinePoolSpec struct {
	Service machinepool.MachinePoolService
}

type CreateMachinePool struct {
	service machinepool.MachinePoolService
}

func NewCreateMachinePool(spec CreateMachinePoolSpec) CreateMachinePool {
	return CreateMachinePool{
		service: spec.Service,
	}
}

func NewCreateMachinePoolCommand() *cobra.Command {
	cmd, options := mpOpts.BuildMachinePoolCreateCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateMachinepoolRunner(options))
	return cmd
}

// Original function refactored to use the new helper functions
func CreateMachinepoolRunner(userOptions *mpOpts.CreateMachinepoolUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		var err error
		options := NewCreateMachinepoolOptions()
		clusterKey := r.GetClusterKey()
		options.args = userOptions

		cluster := r.FetchCluster()
		if err := validateClusterState(cluster, clusterKey); err != nil {
			return err
		}

		val, ok := cluster.Properties()[properties.UseLocalCredentials]
		useLocalCredentials := ok && val == "true"

		if err := validateLabels(cmd, options.args); err != nil {
			return err
		}

		r.AWSClient, err = aws.NewClient().
			Region(cluster.Region().ID()).
			Logger(r.Logger).
			UseLocalCredentials(useLocalCredentials).
			Build()
		if err != nil {
			return fmt.Errorf("Failed to create awsClient: %s", err)
		}
		newService := NewCreateMachinePool(CreateMachinePoolSpec{
			Service: machinepool.NewMachinePoolService(),
		})

		return newService.service.CreateMachinePoolBasedOnClusterType(r,
			cmd, clusterKey, cluster, options.Machinepool())
	}
}

// Validate the cluster's state is ready
func validateClusterState(cluster *cmv1.Cluster, clusterKey string) error {
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}
	return nil
}

// Parse labels if the 'labels' flag is set
func validateLabels(cmd *cobra.Command, args *mpOpts.CreateMachinepoolUserOptions) error {
	if cmd.Flags().Changed("labels") {
		if _, err := mpHelpers.ParseLabels(args.Labels); err != nil {
			return fmt.Errorf("%s", err)
		}
	}
	return nil
}
