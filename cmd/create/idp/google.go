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
	"net/url"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/interactive"
)

func buildGoogleIdp(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret

	if interactive.Enabled() || clientID == "" || clientSecret == "" {
		instructionsURL := "https://console.developers.google.com/projectcreate"
		consoleURL := cluster.Console().URL()
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		err = interactive.PrintHelp(interactive.Help{
			Message: "To use Google as an identity provider, you must first register the application:",
			Steps: []string{
				fmt.Sprintf(`Open the following URL:
    %s`, instructionsURL),
				"Follow the instructions to register your application",
				fmt.Sprintf(`When creating the OAuth client ID, use the following URL for the Authorized redirect URI:
    %s/oauth2callback/%s`, oauthURL, idpName),
			},
		})
		if err != nil {
			return idpBuilder, err
		}

		if clientID == "" {
			clientID, err = interactive.GetPassword(interactive.Input{
				Question: "Client ID",
				Help:     "Paste the Client ID provided by Google when registering your application.",
				Required: true,
			})
			if err != nil {
				return idpBuilder, errors.New("Expected a Google application Client ID")
			}
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetPassword(interactive.Input{
				Question: "Client Secret",
				Help:     "Paste the Client Secret provided by Google when registering your application.",
				Required: true,
			})
			if err != nil {
				return idpBuilder, errors.New("Expected a Google application Client Secret")
			}
		}
	}

	mappingMethod := args.mappingMethod
	if interactive.Enabled() {
		mappingMethod, err = interactive.GetOption(interactive.Input{
			Question: "Mapping method",
			Help:     cmd.Flags().Lookup("mapping-method").Usage,
			Options:  []string{"add", "claim", "generate", "lookup"},
			Default:  mappingMethod,
			Required: true,
		})
	}

	// Create Google IDP
	googleIDP := cmv1.NewGoogleIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret)

	hostedDomain := args.googleHostedDomain
	if interactive.Enabled() {
		hostedDomain, err = interactive.GetString(interactive.Input{
			Question: "Hosted domain",
			Help:     cmd.Flags().Lookup("hosted-domain").Usage,
			Default:  hostedDomain,
			Required: mappingMethod != "lookup",
		})
		if err != nil {
			return idpBuilder, errors.New("Expected a valid Hosted Domain")
		}
	}

	if hostedDomain != "" {
		_, err = url.ParseRequestURI(hostedDomain)
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid Hosted Domain: %v", err)
		}
		// Set the hosted domain, if any
		googleIDP = googleIDP.HostedDomain(hostedDomain)
	}

	// Create new IDP with Google provider
	idpBuilder.
		Type("GoogleIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(mappingMethod)).
		Google(googleIDP)

	return
}
