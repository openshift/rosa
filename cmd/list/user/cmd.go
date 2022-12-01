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

package user

import (
	"fmt"
	"math"
	"os"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "List cluster users",
	Long:    "List administrative cluster users.",
	Example: `  # List all users on a cluster named "mycluster"
  rosa list users --cluster=mycluster`,
	Run: run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	var clusterAdmins []*cmv1.User
	var err error
	r.Reporter.Debugf("Loading users for cluster '%s'", clusterKey)
	// Load cluster-admins for this cluster
	clusterAdmins, err = r.OCMClient.GetUsers(cluster.ID(), "cluster-admins")
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster-admins for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	// Remove cluster-admin user
	for i, user := range clusterAdmins {
		if user.ID() == idp.ClusterAdminUsername {
			clusterAdmins = append(clusterAdmins[:i], clusterAdmins[i+1:]...)
		}
	}

	// Load dedicated-admins for this cluster
	dedicatedAdmins, err := r.OCMClient.GetUsers(cluster.ID(), "dedicated-admins")
	if err != nil {
		r.Reporter.Errorf("Failed to get dedicated-admins for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if len(clusterAdmins) == 0 && len(dedicatedAdmins) == 0 {
		r.Reporter.Warnf("There are no users configured for cluster '%s'", clusterKey)
		os.Exit(1)
	}

	longestUserId := 0.0
	groups := make(map[string][]string)
	for _, user := range clusterAdmins {
		longestUserId = math.Max(longestUserId, float64(len(user.ID())))
		groups[user.ID()] = []string{"cluster-admins"}
	}
	for _, user := range dedicatedAdmins {
		longestUserId = math.Max(longestUserId, float64(len(user.ID())))
		if _, ok := groups[user.ID()]; ok {
			groups[user.ID()] = []string{"cluster-admins", "dedicated-admins"}
		} else {
			groups[user.ID()] = []string{"dedicated-admins"}
		}
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	idLabel := "ID"
	fmt.Fprintf(writer, "%s%-*sGROUPS\n", idLabel, int(longestUserId)-len(idLabel)+1, "")

	for u, r := range groups {
		fmt.Fprintf(writer, "%s%-*s%s\n", u, int(longestUserId)-len(u)+1, "", strings.Join(r, ", "))
		writer.Flush()
	}
}
