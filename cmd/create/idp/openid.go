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

	"github.com/openshift/rosa/pkg/interactive"
)

func buildOpenidIdp(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	issuerURL := args.openidIssuerURL
	email := args.openidEmail
	name := args.openidName
	username := args.openidUsername

	if clientID == "" || clientSecret == "" || issuerURL == "" || (email == "" && name == "" && username == "") {
		interactive.Enable()
	}

	if interactive.Enabled() {
		instructionsURL := "https://docs.openshift.com/dedicated/identity_providers/" +
			"config-identity-providers.html#config-openid-idp_config-identity-providers"
		oauthURL := strings.Replace(cluster.Console().URL(), "console-openshift-console", "oauth-openshift", 1)
		err = interactive.PrintHelp(interactive.Help{
			Message: "To use OpenID as an identity provider, you must first register the application:",
			Steps: []string{
				fmt.Sprintf(`Open the following URL:
    %s`, instructionsURL),
				"Follow the instructions to register your application",
				fmt.Sprintf(`When creating the OpenID, use the following URL for the Authorized redirect URI:
    %s/oauth2callback/%s`, oauthURL, idpName),
			},
		})
		if err != nil {
			return idpBuilder, err
		}
	}

	if interactive.Enabled() {
		clientID, err = interactive.GetString(interactive.Input{
			Question: "Client ID",
			Help:     "Paste the Client ID provided by the OpenID provider when registering your application.",
			Default:  clientID,
			Required: true,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid application Client ID: %s", err)
		}
	}

	if interactive.Enabled() && clientSecret == "" {
		clientSecret, err = interactive.GetPassword(interactive.Input{
			Question: "Client Secret",
			Help:     "Paste the Client Secret provided by the OpenID provider when registering your application.",
			Required: true,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid application Client Secret: %s", err)
		}
	}

	if interactive.Enabled() {
		issuerURL, err = interactive.GetString(interactive.Input{
			Question: "Issuer URL",
			Help:     cmd.Flags().Lookup("issuer-url").Usage,
			Default:  issuerURL,
			Required: true,
			Validators: []interactive.Validator{
				interactive.IsURL,
				validateOpenidIssuerURL,
			},
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid OpenID Issuer URL: %s", err)
		}
	}

	err = validateOpenidIssuerURL(issuerURL)
	if err != nil {
		return idpBuilder, err
	}

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

	mappingMethod, err := getMappingMethod(cmd, args.mappingMethod)
	if err != nil {
		return idpBuilder, err
	}

	if interactive.Enabled() {
		err = interactive.PrintHelp(interactive.Help{
			Message: `You can indicate which claims to use as the user’s preferred user name, display name, and email address.
  At least one claim must be configured to use as the user’s identity. Enter multiple values separated by commas.`,
		})
		if err != nil {
			return idpBuilder, err
		}

		email, err = interactive.GetString(interactive.Input{
			Question: "Email",
			Help:     cmd.Flags().Lookup("email-claims").Usage,
			Default:  email,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}
		name, err = interactive.GetString(interactive.Input{
			Question: "Name",
			Help:     cmd.Flags().Lookup("name-claims").Usage,
			Default:  name,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}
		username, err = interactive.GetString(interactive.Input{
			Question: "Preferred username",
			Help:     cmd.Flags().Lookup("username-claims").Usage,
			Default:  username,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}
	}
	if email == "" && name == "" && username == "" {
		return idpBuilder, errors.New("At least one claim is required: [email-claims name-claims username-claims]")
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

	// Build extra OpenID scopes
	scopes := args.openidScopes
	if interactive.Enabled() {
		scopes, err = interactive.GetString(interactive.Input{
			Question: "Extra scopes",
			Help:     cmd.Flags().Lookup("extra-scopes").Usage,
			Default:  scopes,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of scopes: %s", err)
		}
	}

	// Create OpenID IDP
	openIDIDP := cmv1.NewOpenIDIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		Issuer(issuerURL).
		Claims(openIDClaims)

	if scopes != "" {
		openIDIDP = openIDIDP.ExtraScopes(strings.Split(scopes, ",")...)
	}

	// Set the CA file, if any
	if ca != "" {
		openIDIDP = openIDIDP.CA(ca)
	}

	// Create new IDP with OpenID provider
	idpBuilder.
		Type("OpenIDIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(mappingMethod)).
		OpenID(openIDIDP)

	return
}

func validateOpenidIssuerURL(val interface{}) error {
	issuerURL := fmt.Sprintf("%v", val)
	parsedIssuerURL, err := url.ParseRequestURI(issuerURL)
	if err != nil {
		return fmt.Errorf("Expected a valid OpenID issuer URL: %v", err)
	}
	if parsedIssuerURL.Scheme != "https" {
		return errors.New("Expected OpenID issuer URL to use an https:// scheme")
	}
	if parsedIssuerURL.RawQuery != "" {
		return errors.New("OpenID issuer URL must not have query parameters")
	}
	if parsedIssuerURL.Fragment != "" {
		return errors.New("OpenID issuer URL must not have a fragment")
	}
	return nil
}
