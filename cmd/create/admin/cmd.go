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

	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

const (
	idpName = "htpasswd-1"
)

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
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Try to find an existing htpasswd identity provider and
	// check if cluster-admin user already exists
	existingHTPasswdIDP, existingUserList := idp.FindExistingHTPasswdIDP(cluster, ocmClient)
	if idp.HasClusterAdmin(existingUserList) {
		reporter.Errorf("Cluster '%s' already has an admin", clusterKey)
		os.Exit(1)
	}

	// No cluster admin yet: proceed to create it.
	var password string
	passwordArg := args.passwordArg
	if len(passwordArg) == 0 {
		reporter.Debugf("Generating random password")
		password, err = generateRandomPassword(23)
		if err != nil {
			reporter.Errorf("Failed to generate a random password")
			os.Exit(1)
		}
	} else {
		password = passwordArg
		reporter.Debugf("Using user provided password")
	}

	// Add admin user to the cluster-admins group:
	reporter.Debugf("Adding '%s' user to cluster '%s'", idp.ClusterAdminUsername, clusterKey)
	user, err := cmv1.NewUser().ID(idp.ClusterAdminUsername).Build()
	if err != nil {
		reporter.Errorf("Failed to create user '%s' for cluster '%s'", idp.ClusterAdminUsername, clusterKey)
		os.Exit(1)
	}

	_, err = ocmClient.CreateUser(cluster.ID(), "cluster-admins", user)
	if err != nil {
		reporter.Errorf("Failed to add user '%s' to cluster '%s': %s",
			idp.ClusterAdminUsername, clusterKey, err)
		os.Exit(1)
	}

	// No HTPasswd IDP - create it with cluster-admin user.
	if existingHTPasswdIDP == nil {
		reporter.Debugf("Adding '%s' idp to cluster '%s'", idpName, clusterKey)
		htpasswdIDP := cmv1.NewHTPasswdIdentityProvider().Users(cmv1.NewHTPasswdUserList().Items(
			idp.CreateHTPasswdUser(idp.ClusterAdminUsername, password),
		))
		newIDP, err := cmv1.NewIdentityProvider().
			Type("HTPasswdIdentityProvider").
			Name(idpName).
			Htpasswd(htpasswdIDP).
			Build()
		if err != nil {
			reporter.Errorf("Failed to create '%s' identity provider for cluster '%s'", idpName, clusterKey)
			os.Exit(1)
		}

		// Add HTPasswd IDP to cluster:
		_, err = ocmClient.CreateIdentityProvider(cluster.ID(), newIDP)
		if err != nil {
			reporter.Errorf("Failed to add '%s' identity provider to cluster '%s': %s",
				idpName, clusterKey, err)
			os.Exit(1)
		}
	} else {
		// HTPasswd IDP exists - add new cluster-admin user to it.
		reporter.Debugf("Cluster has an HTPasswd IDP, will add cluster-admin to it")
		err = ocmClient.AddHTPasswdUser(idp.ClusterAdminUsername, password, cluster.ID(), existingHTPasswdIDP.ID())
		if err != nil {
			reporter.Errorf("Failed to add user '%s' to the HTPasswd IDP of cluster '%s': %s",
				idp.ClusterAdminUsername, clusterKey, err)
			os.Exit(1)
		}
	}

	reporter.Infof("Admin account has been added to cluster '%s'.", clusterKey)
	reporter.Infof("Please securely store this generated password. " +
		"If you lose this password you can delete and recreate the cluster admin user.")
	reporter.Infof("To login, run the following command:\n\n"+
		"   oc login %s --username %s --password %s\n", cluster.API().URL(), idp.ClusterAdminUsername, password)
	reporter.Infof("It may take up to a minute for the account to become active.")
}

func generateRandomPassword(length int) (string, error) {
	const (
		lowerLetters = "abcdefghijkmnopqrstuvwxyz"
		upperLetters = "ABCDEFGHIJKLMNPQRSTUVWXYZ"
		digits       = "23456789"
		all          = lowerLetters + upperLetters + digits
	)
	var password string
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			return "", err
		}
		newchar := string(all[n.Int64()])
		if password == "" {
			password = newchar
		}
		if i < length-1 {
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
