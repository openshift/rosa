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

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

const (
	idpName  = "htpasswd"
	username = "cluster-admin"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:   "admin",
	Short: "Creates an admin user to login to the cluster",
	Long:  "Creates a cluster-admin user with an auto-generated password to login to the cluster",
	Example: `  # Create an admin user to login to the cluster
  moactl create admin --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the IdP to (required).",
	)
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	reporter.Warnf("It is recommended to add an identity provider to login to this cluster. " +
		"See 'moactl create idp --help' for more information.")

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
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// TODO: Verify that the htpasswd IdP does not already exist

	// TODO: Verify that the user does not already exist

	password, err := generateRandomPassword(23)
	if err != nil {
		reporter.Errorf("Failed to generate a random password")
		os.Exit(1)
	}

	// Add admin user to the cluster-admins group:
	reporter.Debugf("Adding '%s' user to cluster '%s'", username, clusterKey)
	user, err := cmv1.NewUser().ID(username).Build()
	if err != nil {
		reporter.Errorf("Failed to create user '%s' for cluster '%s'", username, clusterKey)
		os.Exit(1)
	}
	_, err = clustersCollection.Cluster(cluster.ID()).
		Groups().Group("cluster-admins").
		Users().Add().Body(user).
		Send()
	if err != nil {
		reporter.Errorf("Failed to add user '%s' to cluster '%s': %v", username, clusterKey, err)
		os.Exit(1)
	}

	// Create HTPasswd IDP configuration:
	reporter.Debugf("Adding '%s' udp to cluster '%s'", idpName, clusterKey)
	htpasswdIDP := cmv1.NewHTPasswdIdentityProvider().
		Username(username).
		Password(password)

	// Create new IDP with HTPasswd provider:
	idp, err := cmv1.NewIdentityProvider().
		Type("HTPasswdIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod("claim")).
		Htpasswd(htpasswdIDP).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create '%s' identity provider for cluster '%s'", idpName, clusterKey)
		os.Exit(1)
	}

	// Add HTPasswd IDP to cluster:
	_, err = clustersCollection.Cluster(cluster.ID()).
		IdentityProviders().
		Add().
		Body(idp).
		Send()
	if err != nil {
		reporter.Errorf("Failed to add '%s' identity provider to cluster '%s': %s", idpName, clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Admin account has been added to cluster '%s'. "+
		"It may take up to a minute for the account to become active.", clusterKey)
	reporter.Infof("To login, run the following command:\n"+
		"   oc login %s \\\n   --username %s \\\n   --password %s", cluster.API().URL(), username, password)
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
