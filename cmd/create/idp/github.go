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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/interactive"
)

func buildGithubIdp(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	organizations := args.githubOrganizations
	teams := args.githubTeams

	if organizations != "" && teams != "" {
		return idpBuilder, errors.New("GitHub IDP only allows either organizations or teams, but not both")
	}

	var restrictType string
	if organizations != "" {
		restrictType = "organizations"
	}
	if teams != "" {
		restrictType = "teams"
	}

	if interactive.Enabled() || restrictType == "" {
		restrictType, err = interactive.GetOption(interactive.Input{
			Question: "Restrict to members of",
			Help:     "GitHub authentication lets you use either GitHub organizations or GitHub teams to restrict access.",
			Options:  []string{"organizations", "teams"},
			Default:  "organizations",
			Required: true,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid option: %s", err)
		}
	}

	if interactive.Enabled() || (organizations == "" && teams == "") {
		if restrictType == "organizations" {
			organizations, err = interactive.GetString(interactive.Input{
				Question: "GitHub organizations",
				Help:     cmd.Flags().Lookup("organizations").Usage,
				Default:  organizations,
				Required: true,
			})
			if err != nil {
				return idpBuilder, fmt.Errorf("Expected a valid GitHub organization: %s", err)
			}
		} else if restrictType == "teams" {
			teams, err = interactive.GetString(interactive.Input{
				Question: "GitHub teams",
				Help:     cmd.Flags().Lookup("teams").Usage,
				Default:  teams,
				Required: true,
			})
			if err != nil {
				return idpBuilder, fmt.Errorf("Expected a valid GitHub organization: %s", err)
			}
		}
	}

	if organizations == "" && teams == "" {
		return idpBuilder, errors.New("GitHub IdP requires either organizations or teams")
	}

	clientID := args.clientID
	clientSecret := args.clientSecret
	if interactive.Enabled() || clientID == "" || clientSecret == "" {
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

		err = interactive.PrintHelp(interactive.Help{
			Message: "To use GitHub as an identity provider, you must first register the application:",
			Steps: []string{
				fmt.Sprintf(`Open the following URL:
    %s`, registerURL.String()),
				"Click on 'Register application'",
			},
		})
		if err != nil {
			return idpBuilder, err
		}

		clientID, err = interactive.GetString(interactive.Input{
			Question: "Client ID",
			Help:     "Paste the Client ID provided by GitHub when registering your application.",
			Default:  clientID,
			Required: true,
		})
		if err != nil {
			return idpBuilder, errors.New("Expected a GitHub application Client ID")
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetPassword(interactive.Input{
				Question: "Client Secret",
				Help:     "Paste the Client Secret provided by GitHub when registering your application.",
				Required: true,
			})
			if err != nil {
				return idpBuilder, errors.New("Expected a GitHub application Client Secret")
			}
		}
	}

	// Create GitHub IDP
	githubIDP := cmv1.NewGithubIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret)

	githubHostname := args.githubHostname
	if interactive.Enabled() {
		githubHostname, err = interactive.GetString(interactive.Input{
			Question: "GitHub Enterprise Hostname",
			Help:     cmd.Flags().Lookup("hostname").Usage,
			Default:  githubHostname,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid Hostname: %s", err)
		}
	}
	if githubHostname != "" {
		_, err = url.ParseRequestURI(githubHostname)
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid Hostname: %s", err)
		}
		// Set the hostname, if any
		githubIDP = githubIDP.Hostname(githubHostname)

		caPath := args.caPath
		if interactive.Enabled() {
			caPath, err = interactive.GetCert(interactive.Input{
				Question: "CA file path",
				Help:     cmd.Flags().Lookup("ca").Usage,
				Default:  caPath,
			})
			if err != nil {
				return idpBuilder, fmt.Errorf("Expected a valid certificate bundle: %s", err)
			}
		}
		// Get certificate contents
		ca := ""
		if caPath != "" {
			cert, err := ioutil.ReadFile(caPath)
			if err != nil {
				return idpBuilder, fmt.Errorf("Expected a valid certificate bundle: %s", err)
			}
			ca = string(cert)
		}
		// Set the CA file, if any
		if ca != "" {
			githubIDP = githubIDP.CA(ca)
		}
	}

	mappingMethod, err := getMappingMethod(cmd, args.mappingMethod)
	if err != nil {
		return idpBuilder, fmt.Errorf("Expected a valid mapping method: %s", err)
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
		MappingMethod(cmv1.IdentityProviderMappingMethod(mappingMethod)).
		Github(githubIDP)

	return
}
