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

func buildGoogleIdp(cluster *cmv1.Cluster, idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	hostedDomain := args.googleHostedDomain

	isInteractive := clientID == "" ||
		clientSecret == "" ||
		(args.mappingMethod != "lookup" && hostedDomain == "")

	if isInteractive {
		fmt.Println("To use Google as an identity provider, you must first register the application:")
		// instructionsURL := "https://docs.openshift.com/dedicated/4/authentication/identity_providers/configuring-google-identity-provider.html"
		instructionsURL := "https://console.developers.google.com/projectcreate"
		fmt.Println("* Open the following URL:", instructionsURL)
		fmt.Println("* Follow the instructions to register your application")

		consoleURL := cluster.Console().URL()
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		fmt.Println("* When creating the OAuth client ID, use the following URL for the Authorized redirect URI:", oauthURL+"/oauth2callback/"+idpName)

		if clientID == "" {
			clientID, err = interactive.GetInput("Copy the Client ID provided by Google")
			if err != nil {
				return idpBuilder, errors.New("Expected a Google application Client ID")
			}
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetInput("Copy the Client Secret provided by Google")
			if err != nil {
				return idpBuilder, errors.New("Expected a Google application Client Secret")
			}
		}

		if args.mappingMethod != "lookup" && hostedDomain == "" {
			hostedDomain, err = interactive.GetInput("Enter the Hosted Domain to restrict users")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid Hosted Domain")
			}
		}
	}

	// Create Google IDP
	googleIDP := cmv1.NewGoogleIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret)

	if hostedDomain != "" {
		_, err = url.ParseRequestURI(hostedDomain)
		if err != nil {
			return idpBuilder, errors.New(fmt.Sprintf("Expected a valid Hosted Domain: %v", err))
		}
		// Set the hosted domain, if any
		googleIDP = googleIDP.HostedDomain(hostedDomain)
	}

	// Create new IDP with Google provider
	idpBuilder.
		Type("GoogleIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		Google(googleIDP)

	return
}
