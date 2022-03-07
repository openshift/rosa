/*
Copyright (c) 2022 Red Hat, Inc.

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

package idp

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

const ClusterAdminUsername = "cluster-admin"

func createHTPasswdIDP(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	clusterKey string,
	idpName string,
	ocmClient *ocm.Client) {
	var err error
	username := args.htpasswdUsername
	password := args.htpasswdPassword

	if username == "" || password == "" {
		reporter.Infof("At least one user is required to create the IDP.")
		username, password = getUserDetails(cmd)
	}

	// Choose which way to create the IDP according to whether it already has an admin or not.
	htpasswdIDP := FindExistingHTPasswdIDP(cluster, ocmClient)
	if htpasswdIDP != nil {
		containsAdminOnly := !HasClusterAdmin(htpasswdIDP) || htpasswdIDP.Htpasswd().Users().Len() == 0
		if !containsAdminOnly {
			exitHTPasswdCreate("Cluster already has an HTPasswd IDP", nil)
		}
		// Existing IDP contains only admin. Add new user to it
		reporter.Infof("Cluster already has an HTPasswd IDP, new user will be added to it")
		err = ocmClient.AddHTPasswdUser(username, password, cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			reporter.Errorf(
				"Failed to add a user to the HTPasswd IDP of cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		reporter.Infof("User '%s' added", username)
	} else {
		// HTPasswd IDP does not exist - create it
		idpBuilder := cmv1.NewIdentityProvider().
			Type("HTPasswdIdentityProvider").
			Name(idpName).
			Htpasswd(
				cmv1.NewHTPasswdIdentityProvider().Users(
					cmv1.NewHTPasswdUserList().Items(CreateHTPasswdUser(username, password)),
				),
			)
		htpasswdIDP = doCreateIDP(idpName, *idpBuilder, cluster, clusterKey, ocmClient)
	}

	if interactive.Enabled() {
		for shouldAddAnotherUser() {
			username, password = getUserDetails(cmd)
			err = ocmClient.AddHTPasswdUser(username, password, cluster.ID(), htpasswdIDP.ID())
			if err != nil {
				reporter.Errorf(
					"Failed to add a user to the HTPasswd IDP of cluster '%s': %v", clusterKey, err)
				os.Exit(1)
			}
			reporter.Infof("User '%s' added", username)
		}
	}
}

func getUserDetails(cmd *cobra.Command) (string, string) {
	username, err := interactive.GetString(interactive.Input{
		Question: "Username",
		Help:     cmd.Flags().Lookup("username").Usage,
		Default:  "",
		Required: true,
		Validators: []interactive.Validator{
			usernameValidator,
		},
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid username: %s", err)
	}
	password, err := interactive.GetPassword(interactive.Input{
		Question: "Password",
		Help:     cmd.Flags().Lookup("password").Usage,
		Default:  "",
		Required: true,
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid password: %s", err)
	}
	return username, password
}

func shouldAddAnotherUser() bool {
	addAnother, err := interactive.GetBool(interactive.Input{
		Question: "Add another user",
		Help:     "HTPasswd: Add more users to the IDP, to log into the cluster with.\n",
		Default:  false,
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid reply: %s", err)
	}
	return addAnother
}

func CreateHTPasswdUser(username, password string) *cmv1.HTPasswdUserBuilder {
	builder := cmv1.NewHTPasswdUser()
	if username != "" {
		builder = builder.Username(username)
	}
	if password != "" {
		builder = builder.Password(password)
	}
	return builder
}

func exitHTPasswdCreate(format string, err error) {
	reporter.Errorf("Failed to create IDP for cluster '%s': %v",
		clusterKey,
		fmt.Errorf(format, err))
	os.Exit(1)
}

func usernameValidator(val interface{}) error {
	if username, ok := val.(string); ok {
		if username == ClusterAdminUsername {
			return fmt.Errorf("username '%s' is not allowed", username)
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func HasClusterAdmin(htpasswdIDP *cmv1.IdentityProvider) bool {
	hasAdmin := false
	if htpasswdIDP != nil {
		htpasswdIDP.Htpasswd().Users().Each(func(user *cmv1.HTPasswdUser) bool {
			if user.Username() == ClusterAdminUsername {
				hasAdmin = true
			}
			return true
		})
	}
	return hasAdmin
}

func FindExistingHTPasswdIDP(cluster *cmv1.Cluster, ocmClient *ocm.Client) (htpasswdIDP *cmv1.IdentityProvider) {
	reporter.Debugf("Loading cluster's identity providers")
	idps, err := ocmClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	for _, item := range idps {
		if ocm.IdentityProviderType(item) == "htpasswd" {
			htpasswdIDP = item
		}
	}
	return
}
