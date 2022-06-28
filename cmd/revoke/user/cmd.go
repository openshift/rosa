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
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	username string
}

var Cmd = &cobra.Command{
	Use:     "user ROLE",
	Aliases: []string{"role"},
	Short:   "Revoke role from users",
	Long:    "Revoke role from cluster user",
	Example: `  # Revoke cluster-admin role from a user
  rosa revoke user cluster-admins --user=myusername --cluster=mycluster

  # Revoke dedicated-admin role from a user
  rosa revoke user dedicated-admins --user=myusername --cluster=mycluster`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line argument containing the name " +
					"of the group or role to revoke from the user.",
			)
		}
		return nil
	},
}

var validRoles = []string{"cluster-admins", "dedicated-admins"}
var validRolesAliases = []string{"cluster-admin", "dedicated-admin"}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVarP(
		&args.username,
		"user",
		"u",
		"",
		"Username to revoke the role from (required).",
	)
	Cmd.MarkFlagRequired("user")
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	username := args.username
	if !ocm.IsValidUsername(username) {
		r.Reporter.Errorf(
			"Username '%s' isn't valid: it must contain only letters, digits, dashes and underscores",
			username,
		)
		os.Exit(1)
	}
	if username == "cluster-admin" {
		r.Reporter.Errorf("Username 'cluster-admin' is not allowed")
		os.Exit(1)
	}

	role := argv[0]
	// Allow role aliases
	for _, validAlias := range validRolesAliases {
		if role == validAlias {
			role = fmt.Sprintf("%ss", role)
		}
	}
	isRoleValid := false
	// Determine if role is valid
	for _, validRole := range validRoles {
		if role == validRole {
			isRoleValid = true
		}
	}
	if !isRoleValid {
		r.Reporter.Errorf("Expected at least one of %s", validRoles)
		os.Exit(1)
	}

	// Try to find the cluster:
	r.Reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Try to find the user:
	r.Reporter.Debugf("Loading '%s' users for cluster '%s'", role, clusterKey)
	user, err := r.OCMClient.GetUser(cluster.ID(), role, username)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}

	if user == nil {
		r.Reporter.Warnf("Cannot find user '%s' with role '%s' on cluster '%s'", username, role, clusterKey)
		os.Exit(0)
	}

	if !confirm.Confirm("revoke role %s from user %s in cluster %s", role, username, clusterKey) {
		os.Exit(0)
	}

	r.Reporter.Debugf("Removing user '%s' from group '%s' in cluster '%s'", username, role, clusterKey)
	err = r.OCMClient.DeleteUser(cluster.ID(), role, username)
	if err != nil {
		r.Reporter.Errorf("Failed to revoke '%s' from user '%s' in cluster '%s': %s",
			role, username, clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Revoked role '%s' from user '%s' on cluster '%s'", role, username, clusterKey)
}
