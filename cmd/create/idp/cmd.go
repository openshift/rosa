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

package idp

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/interactive"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey string

	idpType string

	clientID      string
	clientSecret  string
	mappingMethod string

	// GitHub
	githubHostname      string
	githubOrganizations string
	githubTeams         string

	// Google
	googleHostedDomain string

	// LDAP
	ldapURL          string
	ldapBindDN       string
	ldapBindPassword string
	ldapIDs          string
	ldapUsernames    string
	ldapDisplayNames string
	ldapEmails       string

	// OpenID
	openidIssuerURL string
	openidEmail     string
	openidName      string
	openidUsername  string
}

var validIdps []string = []string{"github", "google", "ldap", "openid"}

var Cmd = &cobra.Command{
	Use:   "idp",
	Short: "Add IDP for cluster",
	Long:  "Add an Identity providers to determine how users log into the cluster.",
	Example: `  # Add a GitHub identity provider to a cluster named "mycluster"
  moactl create idp --type=github --cluster=mycluster

  # Add an identity provider following interactive prompts
  moactl create idp --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the IdP to (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVarP(
		&args.idpType,
		"type",
		"t",
		"",
		fmt.Sprintf("Type of identity provider. Options are %s\n", validIdps),
	)

	flags.StringVar(
		&args.mappingMethod,
		"mapping-method",
		"claim",
		"Specifies how new identities are mapped to users when they log in",
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		"Client ID from the registered application.",
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"Client Secret from the registered application.\n",
	)

	// GitHub
	flags.StringVar(
		&args.githubHostname,
		"hostname",
		"",
		"GitHub: Optional domain to use with a hosted instance of GitHub Enterprise.",
	)
	flags.StringVar(
		&args.githubOrganizations,
		"organizations",
		"",
		"GitHub: Only users that are members of at least one of the listed organizations will be allowed to log in.",
	)
	flags.StringVar(
		&args.githubTeams,
		"teams",
		"",
		"GitHub: Only users that are members of at least one of the listed teams will be allowed to log in. "+
			"The format is <org>/<team>.\n",
	)

	// Google
	flags.StringVar(
		&args.googleHostedDomain,
		"hosted-domain",
		"",
		"Google: Restrict users to a Google Apps domain.\n",
	)

	// LDAP
	flags.StringVar(
		&args.ldapURL,
		"url",
		"",
		"LDAP: An RFC 2255 URL which specifies the LDAP search parameters to use.",
	)
	flags.StringVar(
		&args.ldapBindDN,
		"bind-dn",
		"",
		"LDAP: DN to bind with during the search phase.",
	)
	flags.StringVar(
		&args.ldapBindPassword,
		"bind-password",
		"",
		"LDAP: Password to bind with during the search phase.",
	)
	flags.StringVar(
		&args.ldapIDs,
		"id-attributes",
		"dn",
		"LDAP: The list of attributes whose values should be used as the user ID.",
	)
	flags.StringVar(
		&args.ldapUsernames,
		"username-attributes",
		"uid",
		"LDAP: The list of attributes whose values should be used as the preferred username.",
	)
	flags.StringVar(
		&args.ldapDisplayNames,
		"name-attributes",
		"cn",
		"LDAP: The list of attributes whose values should be used as the display name.",
	)
	flags.StringVar(
		&args.ldapEmails,
		"email-attributes",
		"",
		"LDAP: The list of attributes whose values should be used as the email address.\n",
	)

	// OpenID
	flags.StringVar(
		&args.openidIssuerURL,
		"issuer-url",
		"",
		"OpenID: The URL that the OpenID Provider asserts as the Issuer Identifier. "+
			"It must use the https scheme with no URL query parameters or fragment.",
	)
	flags.StringVar(
		&args.openidEmail,
		"email-claims",
		"",
		"OpenID: List of claims to use as the email address.",
	)
	flags.StringVar(
		&args.openidName,
		"name-claims",
		"",
		"OpenID: List of claims to use as the display name.",
	)
	flags.StringVar(
		&args.openidUsername,
		"username-claims",
		"",
		"OpenID: List of claims to use as the preferred username when provisioning a user.\n",
	)
}

func run(_ *cobra.Command, _ []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Failed to create logger: %v", err)
		os.Exit(1)
	}

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
	reporter.Infof("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load any existing IDPs for this cluster
	reporter.Infof("Loading identity providers for cluster '%s'", clusterKey)
	idps, err := ocm.GetIdentityProviders(clustersCollection, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Grab all the IDP information interactively if necessary
	idpType := args.idpType

	if idpType == "" {
		idpType, err = interactive.GetInput(fmt.Sprintf("Type of identity provider. Options are %s", validIdps))
		if err != nil {
			reporter.Errorf("Expected a valid IDP type. Options are %s", validIdps)
			os.Exit(1)
		}
	}

	if idpType != "" {
		isValidIdp := false
		for _, idp := range validIdps {
			if idp == idpType {
				isValidIdp = true
			}
		}
		if !isValidIdp {
			reporter.Errorf("Expected a valid IDP type. Options are %s", validIdps)
			os.Exit(1)
		}
	}

	var idpBuilder cmv1.IdentityProviderBuilder
	idpName := getNextName(idpType, idps)
	switch idpType {
	case "github":
		idpBuilder, err = buildGithubIdp(cluster, idpName)
	case "google":
		idpBuilder, err = buildGoogleIdp(cluster, idpName)
	case "ldap":
		idpBuilder, err = buildLdapIdp(cluster, idpName)
	case "openid":
		idpBuilder, err = buildOpenidIdp(cluster, idpName)
	}
	if err != nil {
		reporter.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Configuring IDP for cluster '%s'", clusterKey)

	idp, err := idpBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	_, err = clustersCollection.Cluster(cluster.ID()).
		IdentityProviders().
		Add().
		Body(idp).
		Send()
	if err != nil {
		reporter.Errorf("Failed to add IDP to cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof(
		"Identity Provider '%s' has been created. You need to ensure that there is a list "+
			"of cluster administrators defined. See 'moactl create user --help' for more "+
			"information. To login into the console, open %s and click on %s",
		idpName, cluster.Console().URL(), idpName,
	)
}

func getNextName(idpType string, idps []*cmv1.IdentityProvider) string {
	nextSuffix := 0
	for _, idp := range idps {
		if strings.Contains(idp.Name(), idpType) {
			lastSuffix, err := strconv.Atoi(strings.Split(idp.Name(), "-")[1])
			if err != nil {
				continue
			}
			if lastSuffix >= nextSuffix {
				nextSuffix = lastSuffix
			}
		}
	}
	return fmt.Sprintf("%s-%d", idpType, nextSuffix+1)
}
