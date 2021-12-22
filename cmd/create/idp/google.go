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

	"github.com/openshift/rosa/pkg/interactive"
)

func buildGoogleIdp(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret

	if clientID == "" || clientSecret == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
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

		clientID, err = interactive.GetString(interactive.Input{
			Question: "Client ID",
			Help:     "Paste the Client ID provided by Google when registering your application.",
			Default:  clientID,
			Required: true,
		})
		if err != nil {
			return idpBuilder, errors.New("Expected a Google application Client ID")
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

	// Create Google IDP
	googleIDP := cmv1.NewGoogleIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret)

	mappingMethod := args.mappingMethod
	hostedDomain := args.googleHostedDomain

	mappingMethod, err = getMappingMethod(cmd, mappingMethod)
	if err != nil {
		return idpBuilder, err
	}

	if mappingMethod != "lookup" && hostedDomain == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		hostedDomain, err = interactive.GetString(interactive.Input{
			Question: "Hosted domain",
			Help:     cmd.Flags().Lookup("hosted-domain").Usage,
			Default:  hostedDomain,
			Required: mappingMethod != "lookup",
			Validators: []interactive.Validator{
				interactive.IsURL,
				validateGoogleHostedDomain,
			},
		})
		if err != nil {
			return idpBuilder, errors.New("Expected a valid Hosted Domain")
		}
	}

	if hostedDomain != "" {
		err = validateGoogleHostedDomain(hostedDomain)
		if err != nil {
			return idpBuilder, err
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

func validateGoogleHostedDomain(val interface{}) error {
	hostedDomain := fmt.Sprintf("%v", val)
	parsedHostedDomain, err := url.Parse(hostedDomain)
	if err != nil {
		return fmt.Errorf("Expected a valid Hosted Domain: %v", err)
	}
	if parsedHostedDomain.RawQuery != "" {
		return errors.New("Hosted Domain URL must not have query parameters")
	}
	if parsedHostedDomain.Fragment != "" {
		return errors.New("Hosted Domain URL must not have a fragment")
	}
	return nil
}
