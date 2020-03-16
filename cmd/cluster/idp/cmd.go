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
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var args struct {
	clientID        string
	clientSecret    string
	organization    string
	dedicatedAdmins string
}

var env string

var Cmd = &cobra.Command{
	Use:   "idp [ID|NAME]",
	Short: "Configure IDP for cluster",
	Long:  "Identity providers determine how users log into the cluster.",
	PreRun: func(cmd *cobra.Command, argv []string) {
		env = cmd.Flags().Lookup("env").Value.String()
	},
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		"Client ID from GitHub application.",
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"Client Secret from GitHub application.",
	)
	flags.StringVar(
		&args.organization,
		"organization",
		"",
		"Only users that are members of this GitHub organization will be allowed to log in.",
	)
	flags.StringVar(
		&args.dedicatedAdmins,
		"dedicated-admins",
		"",
		"Grant permission to manage this cluster to these GitHub users.",
	)
}

func run(_ *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Can't create logger: %v", err)
		os.Exit(1)
	}

	// Check command line arguments:
	if len(argv) < 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the name " +
				"or identifier of the cluster",
		)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
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
		reporter.Errorf("Can't create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Can't get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		SetEnv(env).
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Can't close OCM connection: %v", err)
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

	// Grab all the IDP information interactively if necessary
	reader := bufio.NewReader(os.Stdin)
	organization := args.organization
	clientID := args.clientID
	clientSecret := args.clientSecret
	consoleURL := cluster.Console().URL()

	if organization == "" || clientID == "" || clientSecret == "" {
		fmt.Println("To use GitHub as an identity provider, you must first register the application:")

		if organization == "" {
			organization, err = getInput(reader, "\t* Enter the name of your GitHub organization")
			if err != nil {
				reporter.Errorf("Expected a GitHub organization name")
				os.Exit(1)
			}
		}

		// Create the full URL to automatically generate the GitHub app info
		registerURLBase := fmt.Sprintf("https://github.com/organizations/%s/settings/applications/new", organization)
		registerURL, err := url.Parse(registerURLBase)
		if err != nil {
			reporter.Errorf("Error parsing URL: %v", err)
			os.Exit(1)
		}

		urlParams := url.Values{}
		urlParams.Add("oauth_application[name]", cluster.Name())
		urlParams.Add("oauth_application[url]", consoleURL)
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		urlParams.Add("oauth_application[callback_url]", oauthURL+"/oauth2callback/GitHub")

		registerURL.RawQuery = urlParams.Encode()

		fmt.Println("\t* Open the following URL:", registerURL.String())
		fmt.Println("\t* Click on 'Register application'")

		if clientID == "" {
			clientID, err = getInput(reader, "\t* Copy the Client ID provided by GitHub")
			if err != nil {
				reporter.Errorf("Expected a GitHub application Client ID")
				os.Exit(1)
			}
		}

		if clientSecret == "" {
			clientSecret, err = getInput(reader, "\t* Copy the Client Secret provided by GitHub")
			if err != nil {
				reporter.Errorf("Expected a GitHub application Client Secret")
				os.Exit(1)
			}
		}
	}

	dedicatedAdmins := args.dedicatedAdmins
	if dedicatedAdmins == "" {
		dedicatedAdmins, err = getInput(reader, "\t* Enter a comma-separated list of GitHub usernames to grant dedicated-admin rights to your cluster")
		if err != nil {
			reporter.Errorf("Expected a commad-separated list of GitHub usernames")
			os.Exit(1)
		}
	}

	reporter.Infof("Configuring IDP for cluster '%s'", clusterKey)

	// Create GitHub IDP
	githubIDP := cmv1.NewGithubIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		Organizations(organization)

	// Create new IDP with GitHub provider
	idp, err := cmv1.NewIdentityProvider().
		Name("GitHub").
		Type("GithubIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Github(githubIDP).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create IDP for cluster '%s'", clusterKey)
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

	reporter.Infof("Adding dedicated-admin users to cluster '%s'", clusterKey)
	for _, username := range strings.Split(dedicatedAdmins, ",") {
		user, err := cmv1.NewUser().ID(username).Build()
		if err != nil {
			reporter.Errorf("Failed to create dedicated-admin user '%s' for cluster '%s'", username, clusterKey)
			continue
		}
		_, err = clustersCollection.Cluster(cluster.ID()).
			Groups().
			Group("dedicated-admins").
			Users().
			Add().
			Body(user).
			Send()
		if err != nil {
			reporter.Errorf("Failed to add dedicated-admin user '%s' to cluster '%s': %v", username, clusterKey, err)
			continue
		}
	}

	reporter.Infof("Successfully created IDP. To login into the console, click on %s", consoleURL)
}

// Gets user input from the command line
func getInput(r *bufio.Reader, q string) (a string, err error) {
	fmt.Print(q+": ")
	text, err := r.ReadString('\n')
	if err != nil {
		return
	}
	a = strings.Trim(text, "\n")
	return
}
