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
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func buildGithubIdp(idpBuilder *cmv1.IdentityProviderBuilder, cluster *cmv1.Cluster) (err error) {
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
				return errors.New("Expected a GitHub organization name")
			}
		}

		// Create the full URL to automatically generate the GitHub app info
		registerURLBase := fmt.Sprintf("https://github.com/organizations/%s/settings/applications/new", organization)
		registerURL, err := url.Parse(registerURLBase)
		if err != nil {
			return errors.New(fmt.Sprintf("Error parsing URL: %v", err))
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
				return errors.New("Expected a GitHub application Client ID")
			}
		}

		if clientSecret == "" {
			clientSecret, err = getInput(reader, "\t* Copy the Client Secret provided by GitHub")
			if err != nil {
				return errors.New("Expected a GitHub application Client Secret")
			}
		}
	}

	// Create GitHub IDP
	githubIDP := cmv1.NewGithubIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		Organizations(organization)

	// Create new IDP with GitHub provider
	idpBuilder.
		Name("GitHub").
		Type("GithubIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Github(githubIDP)

	return
}
