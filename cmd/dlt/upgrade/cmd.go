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
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"upgrades"},
	Short:   "Cancel cluster upgrade",
	Long:    "Cancel scheduled cluster upgrade",
	Run:     run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	// Try to find the cluster:
	r.Reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	scheduledUpgrade, _, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	if scheduledUpgrade == nil {
		r.Reporter.Warnf("There are no scheduled upgrades on cluster '%s'", clusterKey)
		os.Exit(0)
	}

	if confirm.Confirm("cancel scheduled upgrade on cluster %s", clusterKey) {
		r.Reporter.Debugf("Deleting scheduled upgrade for cluster '%s'", clusterKey)
		canceled, err := r.OCMClient.CancelUpgrade(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to cancel scheduled upgrade on cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		if !canceled {
			r.Reporter.Warnf("There were no scheduled upgrades on cluster '%s'", clusterKey)
			os.Exit(0)
		}

		r.Reporter.Infof("Successfully canceled scheduled upgrade on cluster '%s'", clusterKey)
	}
}
