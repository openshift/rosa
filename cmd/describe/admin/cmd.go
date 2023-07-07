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

package admin

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	cadmin "github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Show details of the cluster-admin user",
	Long:  "Show details of the cluster-admin user and a command to login to the cluster",
	Example: `  # Describe cluster-admin user of a cluster named mycluster
  rosa describe admin -c mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Try to find an existing htpasswd identity provider and
	// check if cluster-admin user already exists
	existingClusterAdminIdp, _ := cadmin.FindExistingClusterAdminIDP(cluster, r)

	if existingClusterAdminIdp != nil {
		r.Reporter.Infof("There is an admin on cluster '%s'. To login, run the following command:\n"+
			"   oc login %s --username %s", clusterKey, cluster.API().URL(), cadmin.ClusterAdminUsername)
	} else {
		r.Reporter.Warnf("There is no admin on cluster '%s'. To create it run the following command:\n"+
			"   rosa create admin -c %s", clusterKey, clusterKey)
		os.Exit(0)
	}
}
