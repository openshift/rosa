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

package add

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"gitlab.cee.redhat.com/service/moactl/pkg/interactive"
)

func buildGithubIdp(cluster *cmv1.Cluster, idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	organizations := args.githubOrganizations
	teams := args.githubTeams
	teamsOrOrgs := ""

	if organizations != "" && teams != "" {
		return idpBuilder, errors.New("GitHub IDP only allows either organizations or teams, but not both.")
	}

	isInteractive := clientID == "" || clientSecret == "" || (organizations == "" && teams == "")

	if isInteractive {
		fmt.Println("To use GitHub as an identity provider, you must first register the application:")

		if organizations == "" && teams == "" {
			teamsOrOrgs, err = interactive.GetInput("Enter a list of GitHub organizations or teams that will have access to this cluster")
			if err != nil {
				return idpBuilder, errors.New("Expected a GitHub organization or team name")
			}
		}

		// Determine if the user entered teams or organizations
		if strings.Contains(teamsOrOrgs, "/") {
			teams = teamsOrOrgs
		} else {
			organizations = teamsOrOrgs
		}

		// Create the full URL to automatically generate the GitHub app info
		registerURLBase := "https://github.com/settings/applications/new"

		// If a single organization was listed, use that to register the application
		if organizations != "" && !strings.Contains(organizations, ",") {
			registerURLBase = fmt.Sprintf("https://github.com/organizations/%s/settings/applications/new", organizations)
		} else if teams != "" && !strings.Contains(teams, ",") {
			teamOrg := strings.Split(teams, "/")[0]
			registerURLBase = fmt.Sprintf("https://github.com/organizations/%s/settings/applications/new", teamOrg)
		}

		registerURL, err := url.Parse(registerURLBase)
		if err != nil {
			return idpBuilder, fmt.Errorf("Error parsing URL: %v", err)
		}

		// Populate fields in the GitHub registration form
		consoleURL := cluster.Console().URL()
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		urlParams := url.Values{}
		urlParams.Add("oauth_application[name]", cluster.Name())
		urlParams.Add("oauth_application[url]", consoleURL)
		urlParams.Add("oauth_application[callback_url]", oauthURL+"/oauth2callback/"+idpName)

		registerURL.RawQuery = urlParams.Encode()

		fmt.Println("* Open the following URL:", registerURL.String())
		fmt.Println("* Click on 'Register application'")

		if clientID == "" {
			clientID, err = interactive.GetInput("Copy the Client ID provided by GitHub")
			if err != nil {
				return idpBuilder, errors.New("Expected a GitHub application Client ID")
			}
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetInput("Copy the Client Secret provided by GitHub")
			if err != nil {
				return idpBuilder, errors.New("Expected a GitHub application Client Secret")
			}
		}
	}

	// Create GitHub IDP
	githubIDP := cmv1.NewGithubIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret)

	if args.githubHostname != "" {
		_, err = url.ParseRequestURI(args.githubHostname)
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid Hostname: %v", err)
		}
		// Set the hostname, if any
		githubIDP = githubIDP.Hostname(args.githubHostname)
	}

	// Set organizations or teams in the IDP object
	if organizations != "" {
		githubIDP = githubIDP.Organizations(strings.Split(organizations, ",")...)
	} else if teams != "" {
		githubIDP = githubIDP.Teams(strings.Split(teams, ",")...)
	}

	// Create new IDP with GitHub provider
	idpBuilder.
		Type("GithubIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		Github(githubIDP)

	return
}
