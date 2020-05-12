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

	"github.com/openshift/moactl/pkg/interactive"
)

func buildOpenidIdp(cluster *cmv1.Cluster, idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	issuerURL := args.openidIssuerURL
	email := args.openidEmail
	name := args.openidName
	username := args.openidUsername

	isInteractive := clientID == "" || clientSecret == "" || issuerURL == "" || (email == "" && name == "" && username == "")

	if isInteractive {
		fmt.Println("To use OpenID as an identity provider, you must first register the application:")
		instructionsURL := "https://docs.openshift.com/dedicated/4/authentication/identity_providers/configuring-oidc-identity-provider.html"
		fmt.Println("* Open the following URL:", instructionsURL)
		fmt.Println("* Follow the instructions to register your application")

		consoleURL := cluster.Console().URL()
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		fmt.Println("* When creating the OpenID, use the following URL for the Authorized redirect URI:", oauthURL+"/oauth2callback/"+idpName)

		if clientID == "" {
			clientID, err = interactive.GetInput("Copy the Client ID provided by the OpenID Provider")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid application Client ID")
			}
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetInput("Copy the Client Secret provided by the OpenID Provider")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid application Client Secret")
			}
		}

		if issuerURL == "" {
			issuerURL, err = interactive.GetInput("URL that the OpenID Provider asserts as the Issuer Identifier")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid OpenID Issuer URL")
			}
		}

		if email == "" {
			email, err = interactive.GetInput("Claim mappings to use as the email address")
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the email address.")
			}
		}

		if name == "" {
			name, err = interactive.GetInput("Claim mappings to use as the display name")
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the display name.")
			}
		}

		if username == "" {
			username, err = interactive.GetInput("Claim mappings to use as the preferred username")
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the preferred username.")
			}
		}
	}

	if email == "" && name == "" && username == "" {
		return idpBuilder, errors.New("At least one claim is required: [email-claims name-claims username-claims]")
	}

	parsedIssuerURL, err := url.ParseRequestURI(issuerURL)
	if err != nil {
		return idpBuilder, fmt.Errorf("Expected a valid OpenID issuer URL: %v", err)
	}
	if parsedIssuerURL.Scheme != "https" {
		return idpBuilder, errors.New("Expected OpenID issuer URL to use an https:// scheme.")
	}
	if parsedIssuerURL.RawQuery != "" {
		return idpBuilder, errors.New("OpenID issuer URL must not have query parameters.")
	}
	if parsedIssuerURL.Fragment != "" {
		return idpBuilder, errors.New("OpenID issuer URL must not have a fragment.")
	}

	// Build OpenID Claims
	openIDClaims := cmv1.NewOpenIDClaims()
	if email != "" {
		openIDClaims = openIDClaims.Email(strings.Split(email, ",")...)
	}
	if name != "" {
		openIDClaims = openIDClaims.Name(strings.Split(name, ",")...)
	}
	if username != "" {
		openIDClaims = openIDClaims.PreferredUsername(strings.Split(username, ",")...)
	}

	// Create OpenID IDP
	openIDIDP := cmv1.NewOpenIDIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		Issuer(issuerURL).
		Claims(openIDClaims)

	// Create new IDP with OpenID provider
	idpBuilder.
		Type("OpenIDIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		OpenID(openIDIDP)

	return
}
