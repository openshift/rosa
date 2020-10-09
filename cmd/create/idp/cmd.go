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
	idpName string

	clientID      string
	clientSecret  string
	mappingMethod string
	caPath        string

	// GitHub
	githubHostname      string
	githubOrganizations string
	githubTeams         string

	// GitLab
	gitlabURL string

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

// TODO: Add gitlab
var validIdps []string = []string{"github", "gitlab", "google", "ldap", "openid"}

var Cmd = &cobra.Command{
	Use:   "idp",
	Short: "Add IDP for cluster",
	Long:  "Add an Identity providers to determine how users log into the cluster.",
	Example: `  # Add a GitHub identity provider to a cluster named "mycluster"
  moactl create idp --type=github --cluster=mycluster

  # Add an identity provider following interactive prompts
  moactl create idp --cluster=mycluster --interactive`,
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
		fmt.Sprintf("Type of identity provider. Options are %s.", validIdps),
	)
	flags.StringVar(
		&args.idpName,
		"name",
		"",
		"Name for the identity provider.\n",
	)

	flags.StringVar(
		&args.mappingMethod,
		"mapping-method",
		"claim",
		"Specifies how new identities are mapped to users when they log in.",
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
		"Client Secret from the registered application.",
	)
	flags.StringVar(
		&args.caPath,
		"ca",
		"",
		"Path to PEM-encoded certificate file to use when making requests to the server.\n",
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

	// GitLab
	flags.StringVar(
		&args.gitlabURL,
		"host-url",
		"https://gitlab.com",
		"GitLab: The host URL of a GitLab provider.",
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

	// Load any existing IDPs for this cluster
	reporter.Debugf("Loading identity providers for cluster '%s'", clusterKey)
	idps, err := ocm.GetIdentityProviders(clustersCollection, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if interactive.Enabled() {
		reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Grab all the IDP information interactively if necessary
	idpType := args.idpType
	if interactive.Enabled() || idpType == "" {
		if idpType == "" {
			idpType = validIdps[0]
		}
		idpType, err = interactive.GetOption(interactive.Input{
			Question: "Type of identity provider",
			Options:  validIdps,
			Required: true,
			Default:  idpType,
		})
		if err != nil {
			reporter.Errorf("Expected a valid IdP type: %s", err)
			os.Exit(1)
		}
	}
	if idpType == "" {
		reporter.Errorf("Expected a valid IDP type. Options are: %s", strings.Join(validIdps, ","))
		os.Exit(1)
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

	idpName := args.idpName
	// Auto-generate a name if none provided
	if !cmd.Flags().Changed("name") {
		idpName = getNextName(idpType, idps)
	}
	if interactive.Enabled() {
		idpName, err = interactive.GetString(interactive.Input{
			Question: "Identity provider name",
			Help:     cmd.Flags().Lookup("name").Usage,
			Default:  idpName,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid name for the identity provider: %s", err)
			os.Exit(1)
		}
	}

	var idpBuilder cmv1.IdentityProviderBuilder
	switch idpType {
	case "github":
		idpBuilder, err = buildGithubIdp(cmd, cluster, idpName)
	case "gitlab":
		idpBuilder, err = buildGitlabIdp(cmd, cluster, idpName)
	case "google":
		idpBuilder, err = buildGoogleIdp(cmd, cluster, idpName)
	case "ldap":
		idpBuilder, err = buildLdapIdp(cmd, cluster, idpName)
	case "openid":
		idpBuilder, err = buildOpenidIdp(cmd, cluster, idpName)
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
		"Identity Provider '%s' has been created.\n"+
			"   It will take up to 1 minute for this configuration to be enabled.\n"+
			"   To add cluster administrators, see 'moactl create user --help'.\n"+
			"   To login into the console, open %s and click on %s.",
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

func getMappingMethod(cmd *cobra.Command, mappingMethod string) (string, error) {
	var err error
	if interactive.Enabled() {
		usage := fmt.Sprintf("%s\n  For more information see the documentation:\n  %s",
			cmd.Flags().Lookup("mapping-method").Usage,
			"https://docs.openshift.com/dedicated/4/authentication/dedicated-understanding-authentication.html")
		mappingMethod, err = interactive.GetOption(interactive.Input{
			Question: "Mapping method",
			Help:     usage,
			Options:  []string{"add", "claim", "generate", "lookup"},
			Default:  mappingMethod,
			Required: true,
		})
	}
	return mappingMethod, err
}
