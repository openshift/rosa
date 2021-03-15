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
	"regexp"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

type IdentityProvider interface {
	Name() string
}

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
	ldapInsecure     bool
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
	openidScopes    string
}

var validIdps []string = []string{"github", "gitlab", "google", "ldap", "openid"}
var validMappingMethods []string = []string{"add", "claim", "generate", "lookup"}

var idRE = regexp.MustCompile(`(?i)^[0-9a-z]+([-_][0-9a-z]+)*$`)

var Cmd = &cobra.Command{
	Use:   "idp",
	Short: "Add IDP for cluster",
	Long:  "Add an Identity providers to determine how users log into the cluster.",
	Example: `  # Add a GitHub identity provider to a cluster named "mycluster"
  rosa create idp --type=github --cluster=mycluster

  # Add an identity provider following interactive prompts
  rosa create idp --cluster=mycluster --interactive`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	arguments.AddRegionFlag(flags)

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
		fmt.Sprintf("Specifies how new identities are mapped to users when they log in. Options are %s", validMappingMethods),
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
	flags.BoolVar(
		&args.ldapInsecure,
		"insecure",
		false,
		"LDAP: Do not make TLS connections to the server.",
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
		"OpenID: List of claims to use as the preferred username when provisioning a user.",
	)
	flags.StringVar(
		&args.openidScopes,
		"extra-scopes",
		"",
		"OpenID: List of scopes to request, in addition to the 'openid' scope, during the authorization token request.\n",
	)

	interactive.AddFlag(flags)
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

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Region(region).
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

	if interactive.Enabled() {
		reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	// Grab all the IDP information interactively if necessary
	idpType := args.idpType
	if idpType == "" {
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

	idpName := strings.Trim(args.idpName, " \t")

	// Auto-generate a name if none provided
	if !cmd.Flags().Changed("name") {
		idps := getIdps(reporter, clustersCollection, cluster)
		idpName = GenerateIdpName(idpType, idps)
	} else {
		isValidIdpName := idRE.MatchString(idpName)
		if !isValidIdpName {
			reporter.Errorf("Invalid identifier '%s' for 'name'", idpName)
			os.Exit(1)
		}
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
	idpName = strings.Trim(idpName, " \t")

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

	res, err := clustersCollection.Cluster(cluster.ID()).
		IdentityProviders().
		Add().
		Body(idp).
		Send()
	if err != nil {
		reporter.Debugf(err.Error())
		reporter.Errorf("Failed to add IDP to cluster '%s': %s", clusterKey, res.Error().Reason())
		os.Exit(1)
	}

	reporter.Infof(
		"Identity Provider '%s' has been created.\n"+
			"   It will take up to 1 minute for this configuration to be enabled.\n"+
			"   To add cluster administrators, see 'rosa create user --help'.\n"+
			"   To login into the console, open %s and click on %s.",
		idpName, cluster.Console().URL(), idpName,
	)
}

func GenerateIdpName(idpType string, idps []IdentityProvider) string {
	nextSuffix := 0
	for _, idp := range idps {
		if strings.Contains(idp.Name(), idpType) {
			idpNameComponents := strings.Split(idp.Name(), "-")
			if len(idpNameComponents) < 2 {
				continue
			}
			lastSuffix, err := strconv.Atoi(idpNameComponents[1])
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
			Options:  validMappingMethods,
			Default:  mappingMethod,
			Required: true,
		})
	}
	isValidMappingMethod := false
	for _, validMappingMethod := range validMappingMethods {
		if mappingMethod == validMappingMethod {
			isValidMappingMethod = true
		}
	}
	if !isValidMappingMethod {
		err = fmt.Errorf("Expected a valid mapping method. Options are %s", validMappingMethods)
	}
	return mappingMethod, err
}

func getIdps(reporter *reporter.Object, clusters *cmv1.ClustersClient, cluster *cmv1.Cluster) []IdentityProvider {
	// Load any existing IDPs for this cluster
	reporter.Debugf("Loading identity providers for cluster '%s'", cluster.ID())

	ocmIdps, err := ocm.GetIdentityProviders(clusters, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get identity providers for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	idps := []IdentityProvider{}
	for _, idp := range ocmIdps {
		idps = append(idps, idp)
	}
	return idps
}
