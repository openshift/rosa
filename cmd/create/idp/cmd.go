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

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const HTPasswdIDPName = "htpasswd"

type IdentityProvider interface {
	Name() string
}

var args struct {
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
	openidGroups    string
	openidScopes    string

	// HTPasswd
	htpasswdUsername string
	htpasswdPassword string
	htpasswdUsers    []string
	htpasswdFile     string
}

var validIdps = []string{"github", "gitlab", "google", "htpasswd", "ldap", "openid"}
var validMappingMethods = []string{"add", "claim", "generate", "lookup"}

var idRE = regexp.MustCompile(`(?i)^[0-9a-z]+([-_][0-9a-z]+)*$`)

var Cmd = &cobra.Command{
	Use:   "idp",
	Short: "Add IDP for cluster",
	Long:  "Add an Identity providers to determine how users log into the cluster.",
	Example: `  # Add a GitHub identity provider to a cluster named "mycluster"
  rosa create idp --type=github --cluster=mycluster

  # Add an identity provider following interactive prompts
  rosa create idp --cluster=mycluster --interactive`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.StringVarP(
		&args.idpType,
		"type",
		"t",
		"",
		fmt.Sprintf("Type of identity provider. Options are %s.", validIdps),
	)
	Cmd.RegisterFlagCompletionFunc("type", typeCompletion)

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
		fmt.Sprintf(
			"Specifies how new identities are mapped to users when they log in. Options are %s",
			validMappingMethods,
		),
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
		&args.openidGroups,
		"groups-claims",
		"",
		"OpenID: List of claims to use as the groups names.",
	)
	flags.StringVar(
		&args.openidScopes,
		"extra-scopes",
		"",
		"OpenID: List of scopes to request, in addition to the 'openid' scope, during the authorization token request.\n",
	)

	// HTPasswd
	flags.StringVar(
		&args.htpasswdUsername,
		"username",
		"",
		"HTPasswd: Username to log into the cluster's console with.\n"+
			"Username must not contain /, :, or %%",
	)
	flags.StringVar(
		&args.htpasswdPassword,
		"password",
		"",
		"HTPasswd: Password for provided username, to log into the cluster's console with.\n"+
			"The password must\n"+
			"- Be at least 14 characters (ASCII-standard) without whitespaces\n"+
			"- Include uppercase letters, lowercase letters, and numbers or symbols (ASCII-standard characters only)",
	)

	//makring hidden as this is now only for backwards compatibility
	flags.MarkHidden("username")
	flags.MarkHidden("password")

	// HTPasswd
	flags.StringSliceVarP(
		&args.htpasswdUsers,
		"users",
		"u",
		[]string{},
		"HTPasswd: List of users to add to the IDP. \n"+
			"It must be a comma separated list of  username:password, i.e user1:password,user2:password \n",
	)

	flags.StringVar(
		&args.htpasswdFile,
		"from-file",
		"",
		"HTPasswd: Path to a well formed htpasswd file.\n",
	)

	interactive.AddFlag(flags)
}

func typeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return validIdps, cobra.ShellCompDirectiveDefault
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

	if cluster.ExternalAuthConfig().Enabled() {
		r.Reporter.Errorf("Adding IDP is not supported for clusters with external authentication configured.")
		os.Exit(1)
	}

	// Grab all the IDP information interactively if necessary
	idpType := args.idpType
	if idpType == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be left empty and a default will be selected.")
	}

	var err error
	if interactive.Enabled() {
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
			r.Reporter.Errorf("Expected a valid IdP type: %s", err)
			os.Exit(1)
		}
	}
	if idpType == "" {
		r.Reporter.Errorf("Expected a valid IDP type. Options are: %s", strings.Join(validIdps, ","))
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
			r.Reporter.Errorf("Expected a valid IDP type. Options are %s", validIdps)
			os.Exit(1)
		}
	}

	idpName := strings.Trim(args.idpName, " \t")

	// Auto-generate a name if none provided
	if !cmd.Flags().Changed("name") {
		idps := getIdps(r, cluster)
		idpName = GenerateIdpName(idpType, idps)
	}

	if interactive.Enabled() {
		idpName = getIDPName(cmd, idpName, r)
	}
	idpName = strings.Trim(idpName, " \t")

	err = ValidateIdpName(idpName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	var idpBuilder cmv1.IdentityProviderBuilder
	switch idpType {
	case "github":
		idpBuilder, err = buildGithubIdp(cmd, cluster, idpName)
	case "gitlab":
		idpBuilder, err = buildGitlabIdp(cmd, cluster, idpName)
	case "google":
		idpBuilder, err = buildGoogleIdp(cmd, cluster, idpName)
	case "htpasswd":
		createHTPasswdIDP(cmd, cluster, clusterKey, idpName, r)
		os.Exit(0)
	case "ldap":
		idpBuilder, err = buildLdapIdp(cmd, cluster, idpName)
	case "openid":
		idpBuilder, err = buildOpenidIdp(cmd, cluster, idpName)
	}
	if err != nil {
		r.Reporter.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	doCreateIDP(idpName, idpBuilder, cluster, clusterKey, r)
}

func getIDPName(cmd *cobra.Command, idpName string, r *rosa.Runtime) string {
	idpName, err := interactive.GetString(interactive.Input{
		Question: "Identity provider name",
		Help:     cmd.Flags().Lookup("name").Usage,
		Default:  idpName,
		Required: true,
		Validators: []interactive.Validator{
			ValidateIdpName,
		},
	})
	if err != nil {
		r.Reporter.Errorf("Expected a valid name for the identity provider: %s", err)
		os.Exit(1)
	}
	return strings.Trim(idpName, " \t")
}

func ValidateIdpName(idpName interface{}) error {

	name, ok := idpName.(string)

	if !ok {
		return fmt.Errorf("Invalid type for identity provider name. Expected a string,  got %T", idpName)
	}

	if !idRE.MatchString(name) {
		return fmt.Errorf("Invalid identifier '%s' for 'name'", idpName)
	}

	if strings.EqualFold(name, "cluster-admin") {
		return fmt.Errorf("The name \"cluster-admin\" is reserved for admin user IDP")
	}
	return nil
}

func doCreateIDP(
	idpName string,
	idpBuilder cmv1.IdentityProviderBuilder,
	cluster *cmv1.Cluster, clusterKey string,
	r *rosa.Runtime) *cmv1.IdentityProvider {
	r.Reporter.Infof("Configuring IDP for cluster '%s'", clusterKey)

	idp, err := idpBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	createdIdp, err := r.OCMClient.CreateIdentityProvider(cluster.ID(), idp)
	if err != nil {
		r.Reporter.Errorf("Failed to add IDP to cluster '%s': %s", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof(
		"Identity Provider '%s' has been created.\n"+
			"   It may take several minutes for this access to become active.\n"+
			"   To add cluster administrators, see 'rosa grant user --help'.\n", idpName)
	if !interactive.Enabled() && ocm.HasAuthURLSupport(createdIdp) {
		callbackURL, err := ocm.GetOAuthURL(cluster, createdIdp)
		if err == nil {
			r.Reporter.Infof("Callback URI: %s", callbackURL)
		}
	}
	// Console may not be available yet
	if ocm.IsConsoleAvailable(cluster) {
		clusterConsole := cluster.Console().URL()
		r.Reporter.Infof(
			"To log in to the console, open %s and click on '%s'.", clusterConsole, idpName)
	} else {
		// This warning is because IDPs depends on HostedCluster network which may not be available yet.
		// HTPasswd has no external dependencies so no warning needed.
		if createdIdp.Type() != cmv1.IdentityProviderTypeHtpasswd {
			r.Reporter.Warnf(
				"Authentication traffic for '%s' will be ready when workload\n"+
					"   nodes are provisioned and ready in your AWS account.", idpName)
		}
	}
	return createdIdp
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
			instructionsURLBase+"understanding-idp_config-identity-providers")
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

func getIdps(r *rosa.Runtime, cluster *cmv1.Cluster) []IdentityProvider {
	// Load any existing IDPs for this cluster
	r.Reporter.Debugf("Loading identity providers for cluster '%s'", cluster.ID())

	ocmIdps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get identity providers for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	idps := []IdentityProvider{}
	for _, idp := range ocmIdps {
		idps = append(idps, idp)
	}
	return idps
}
