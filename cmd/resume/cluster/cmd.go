/*
Copyright (c) 2021 Red Hat, Inc.

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

package cluster

import (
	"os"

	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Resume cluster",
	Long:  "Resume cluster.",
	Example: `  # Resume the cluster
  rosa resume cluster -c mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	// Get the cluster to check the state
	cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateHibernating {
		r.Reporter.Errorf("Resuming a cluster from hibernation is only supported for clusters in "+
			"'Hibernating' state. Cluster '%s' is in '%s' state",
			clusterKey, cluster.State())
		os.Exit(1)
	}
	if !confirm.Confirm("resume cluster %s", clusterKey) {
		os.Exit(1)
	}
	err = r.OCMClient.ResumeCluster(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}
	r.Reporter.Infof("Cluster '%s' is resuming.", clusterKey)
}
