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
	"crypto/rand"
	"math/big"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/object"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const ClusterAdminUsername = "cluster-admin"
const ClusterAdminIDPname = "cluster-admin"
const GeneratingRandomPasswordString = "Generating random password"
const MaxPasswordLength = 23

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Creates an admin user to login to the cluster",
	Long:  "Creates a cluster-admin user with an auto-generated password to login to the cluster",
	Example: `  # Create an admin user to login to the cluster
  rosa create admin -c mycluster -p MasterKey123`,
	Run: run,
}

var args struct {
	passwordArg string
}

func init() {
	ocm.AddClusterFlag(Cmd)
	flags := Cmd.Flags()
	flags.StringVarP(
		&args.passwordArg,
		"password",
		"p",
		"",
		"Choice of password for admin user.",
	)
	output.AddFlag(Cmd)
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

	// Try to find an existing cluster admin IDP
	existingClusterAdminIDP, _ := FindExistingClusterAdminIDP(cluster, r)
	if existingClusterAdminIDP != nil {
		r.Reporter.Errorf("Cluster '%s' already has an admin", clusterKey)
		os.Exit(1)
	}

	// No cluster admin yet: proceed to create it.
	var password string
	var err error
	passwordArg := args.passwordArg
	if len(passwordArg) == 0 {
		r.Reporter.Debugf(GeneratingRandomPasswordString)
		password, err = GenerateRandomPassword()
		if err != nil {
			r.Reporter.Errorf("Failed to generate a random password")
			os.Exit(1)
		}
	} else {
		password = passwordArg
		r.Reporter.Debugf("Using user provided password")
	}

	// Add admin user to the cluster-admins group:
	r.Reporter.Debugf("Adding '%s' user to cluster '%s'", ClusterAdminUsername, clusterKey)
	user, err := cmv1.NewUser().ID(ClusterAdminUsername).Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create user '%s' for cluster '%s'", ClusterAdminUsername, clusterKey)
		os.Exit(1)
	}

	_, err = r.OCMClient.CreateUser(cluster.ID(), "cluster-admins", user)
	if err != nil {
		r.Reporter.Errorf("Failed to add user '%s' to cluster '%s': %s",
			ClusterAdminUsername, clusterKey, err)
		os.Exit(1)
	}

	// No ClusterAdmin IDP exists, create an Htpasswd IDP
	// named 'ClusterAdmin' specifically for cluster-admin user
	r.Reporter.Debugf("Adding '%s' idp to cluster '%s'", ClusterAdminIDPname, clusterKey)
	htpasswdIDP := cmv1.NewHTPasswdIdentityProvider().Users(cmv1.NewHTPasswdUserList().Items(
		cmv1.NewHTPasswdUser().Username(ClusterAdminUsername).Password(password),
	))
	clusterAdminIDP, err := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeHtpasswd).
		Name(ClusterAdminIDPname).
		Htpasswd(htpasswdIDP).
		Build()
	if err != nil {
		r.Reporter.Errorf(
			"Failed to create '%s' identity provider for cluster '%s'",
			ClusterAdminIDPname,
			clusterKey,
		)
		os.Exit(1)
	}

	// Add HTPasswd IDP to cluster:
	_, err = r.OCMClient.CreateIdentityProvider(cluster.ID(), clusterAdminIDP)
	if err != nil {
		//since we could not add the HTPasswd IDP to the cluster, roll back and remove the cluster admin
		r.Reporter.Errorf("Failed to add '%s' identity provider to cluster '%s' as part of admin flow. "+
			"Please try again: %s", ClusterAdminIDPname, clusterKey, err)

		err = r.OCMClient.DeleteUser(cluster.ID(), "cluster-admins", user.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to revert the admin user for cluster '%s'. Please try again: %s",
				clusterKey, err)
		}
		os.Exit(1)
	}

	outputObject := object.Object{
		"api_url":  cluster.API().URL(),
		"username": ClusterAdminUsername,
		"password": password,
	}

	if output.HasFlag() {
		if len(passwordArg) != 0 {
			delete(outputObject, "password")
		}
		err = output.Print(outputObject)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	r.Reporter.Infof("Admin account has been added to cluster '%s'.", clusterKey)
	r.Reporter.Infof("Please securely store this generated password. " +
		"If you lose this password you can delete and recreate the cluster admin user.")
	r.Reporter.Infof("To login, run the following command:\n\n"+
		"   oc login %s --username %s --password %s\n",
		outputObject["api_url"], outputObject["username"], outputObject["password"])
	r.Reporter.Infof("It may take several minutes for this access to become active.")
}

func GenerateRandomPassword() (string, error) {
	const (
		lowerLetters = "abcdefghijkmnopqrstuvwxyz"
		upperLetters = "ABCDEFGHIJKLMNPQRSTUVWXYZ"
		digits       = "23456789"
		all          = lowerLetters + upperLetters + digits
	)
	var password string
	for i := 0; i < MaxPasswordLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			return "", err
		}
		newchar := string(all[n.Int64()])
		if password == "" {
			password = newchar
		}
		if i < MaxPasswordLength-1 {
			n, err = rand.Int(rand.Reader, big.NewInt(int64(len(password)+1)))
			if err != nil {
				return "", err
			}
			j := n.Int64()
			password = password[0:j] + newchar + password[j:]
		}
	}

	pw := []rune(password)
	for _, replace := range []int{5, 11, 17} {
		pw[replace] = '-'
	}

	return string(pw), nil
}

func FindExistingClusterAdminIDP(cluster *cmv1.Cluster, r *rosa.Runtime) (
	htpasswdIDP *cmv1.IdentityProvider, userList *cmv1.HTPasswdUserList) {

	// admin user will now be created in a htpasswd IDP named 'cluster-admin'
	// It should suffice to look for this idp but for back-compatibility ,
	// i.e for cases where the cluster was created  with previous versions of rosacli
	// we are going to search  all htpasswd idps

	r.Reporter.Debugf("Loading cluster's identity providers")
	idps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get identity providers for cluster '%s': %v", r.ClusterKey, err)
		os.Exit(1)
	}

	for _, item := range idps {
		if ocm.IdentityProviderType(item) == ocm.HTPasswdIDPType {

			itemUserList, err := r.OCMClient.GetHTPasswdUserList(cluster.ID(), item.ID())
			r.Reporter.Debugf("user list %s: %v", item.Name(), itemUserList)
			if err != nil {
				r.Reporter.Errorf("Failed to get user list of the HTPasswd IDP of '%s: %s': %v", item.Name(), r.ClusterKey, err)
				os.Exit(1)
			}
			if HasClusterAdmin(itemUserList) {
				htpasswdIDP = item
				userList = itemUserList
				return
			}
		}
	}
	return
}

func HasClusterAdmin(userList *cmv1.HTPasswdUserList) bool {
	hasAdmin := false
	userList.Each(func(user *cmv1.HTPasswdUser) bool {
		if user.Username() == ClusterAdminUsername {
			hasAdmin = true
			return false
		}
		return true
	})
	return hasAdmin
}
