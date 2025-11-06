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

package upgrade

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	nodePool string
}

var Cmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"upgrades"},
	Short:   "Cancel cluster upgrade",
	Long:    "Cancel scheduled cluster upgrade",
	Run:     run,
	Args:    machinepool.NewMachinepoolArgsFunction(true),
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.nodePool,
		"machinepool",
		"",
		"Machine pool of the cluster to target",
	)

	confirm.AddFlag(flags)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime) error {
	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}

	if args.nodePool != "" && !ocm.IsHyperShiftCluster(cluster) {
		return fmt.Errorf("the '--machinepool' option is only supported for Hosted Control Planes")
	}

	if args.nodePool != "" {
		return deleteHypershiftNodePoolUpgrade(r, cluster.ID(), clusterKey, args.nodePool)
	}

	if ocm.IsHyperShiftCluster(cluster) {
		return deleteHypershiftUpgrade(r, cluster.ID(), clusterKey)
	} else {
		return deleteClassicUpgrade(r, cluster.ID(), clusterKey)
	}
}

func deleteClassicUpgrade(r *rosa.Runtime, clusterID, clusterKey string) error {
	scheduledUpgrade, _, err := r.OCMClient.GetScheduledUpgrade(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
	}
	if scheduledUpgrade == nil {
		r.Reporter.Infof("There are no scheduled upgrades on cluster '%s'", clusterKey)
		return nil
	}

	if confirm.Confirm("cancel scheduled upgrade on cluster %s", clusterKey) {
		r.Reporter.Debugf("Deleting scheduled upgrade for cluster '%s'", clusterKey)
		canceled, err := r.OCMClient.CancelUpgrade(clusterID)
		if err != nil {
			return fmt.Errorf("failed to cancel scheduled upgrade on cluster '%s': %v", clusterKey, err)
		}

		if !canceled {
			r.Reporter.Warnf("There were no scheduled upgrades on cluster '%s'", clusterKey)
			return nil
		}

		r.Reporter.Infof("Successfully canceled scheduled upgrade on cluster '%s'", clusterKey)
	}
	return nil
}

func deleteHypershiftUpgrade(r *rosa.Runtime, clusterID, clusterKey string) error {
	scheduledUpgrade, err := r.OCMClient.GetControlPlaneScheduledUpgrade(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
	}

	if scheduledUpgrade == nil {
		r.Reporter.Infof("There are no scheduled upgrades on cluster '%s'", clusterKey)
		return nil
	}

	if confirm.Confirm("cancel scheduled upgrade on cluster '%s'", clusterKey) {
		r.Reporter.Debugf("Deleting scheduled upgrade for cluster '%s'", clusterKey)
		canceled, err := r.OCMClient.CancelControlPlaneUpgrade(clusterID, scheduledUpgrade.ID())
		if err != nil {
			return fmt.Errorf("failed to cancel scheduled upgrade on cluster '%s': %v", clusterKey, err)
		}

		if !canceled {
			r.Reporter.Warnf("There were no scheduled upgrades on cluster '%s'", clusterKey)
			return nil
		}

		r.Reporter.Infof("Successfully canceled scheduled upgrade on cluster '%s'", clusterKey)
	}
	return nil
}

func deleteHypershiftNodePoolUpgrade(r *rosa.Runtime, clusterID, clusterKey, nodePoolID string) error {
	_, scheduledUpgrade, err := r.OCMClient.GetHypershiftNodePoolUpgrade(clusterID, clusterKey, nodePoolID)
	if err != nil {
		return err
	}

	if scheduledUpgrade == nil {
		r.Reporter.Infof("There are no scheduled upgrades for machine pool '%s' for cluster '%s'",
			nodePoolID, clusterKey)
		return nil
	}

	if confirm.Confirm("cancel scheduled upgrade on machine pool '%s'", nodePoolID) {
		r.Reporter.Debugf("Deleting scheduled upgrade for machine pool '%s'", nodePoolID)
		canceled, err := r.OCMClient.CancelNodePoolUpgrade(clusterID, nodePoolID, scheduledUpgrade.ID())
		if err != nil {
			return fmt.Errorf("failed to cancel scheduled upgrades for machine pool '%s': %v", nodePoolID, err)
		}

		if !canceled {
			r.Reporter.Warnf("There were no scheduled upgrades for machine pool '%s' for cluster '%s'",
				nodePoolID, clusterKey)
			return nil
		}

		r.Reporter.Infof("Successfully canceled scheduled upgrade for machine pool '%s' for cluster '%s'",
			nodePoolID, clusterKey)
	}
	return nil
}
