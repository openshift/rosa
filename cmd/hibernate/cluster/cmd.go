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
	Short: "Hibernate cluster",
	Long:  "Hibernate cluster.",
	Example: `  # Hibernate the cluster
  rosa hibernate cluster -c mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(Cmd.Flags())
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Hibernating a cluster is only supported for 'Ready' clusters."+
			" Cluster '%s' is in '%s' state",
			clusterKey, cluster.State())
		os.Exit(1)
	}

	if !confirm.Yes() && !confirm.Confirm("hibernate cluster %s", clusterKey) {
		os.Exit(1)
	}

	err := r.OCMClient.HibernateCluster(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}
	r.Reporter.Infof("Cluster '%s' is hibernating.", clusterKey)
}
