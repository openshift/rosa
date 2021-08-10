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

func buildGitlabIdp(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	gitlabURL := args.gitlabURL

	if !cmd.Flags().Changed("host-url") {
		gitlabURL, err = interactive.GetString(interactive.Input{
			Question: "URL",
			Help:     cmd.Flags().Lookup("host-url").Usage,
			Default:  gitlabURL,
			Required: true,
			Validators: []interactive.Validator{
				interactive.IsURL,
				validateGitlabHostURL,
			},
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid GitLab provider URL: %s", err)
		}
	}
	err = validateGitlabHostURL(gitlabURL)
	if err != nil {
		return idpBuilder, err
	}

	if clientID == "" || clientSecret == "" {
		instructionsURL := fmt.Sprintf("%s/profile/applications", gitlabURL)
		consoleURL := cluster.Console().URL()
		oauthURL := strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
		err = interactive.PrintHelp(interactive.Help{
			Message: "To use GitLab as an identity provider, register the application by opening:",
			Steps:   []string{instructionsURL},
		})
		if err != nil {
			return idpBuilder, err
		}
		err = interactive.PrintHelp(interactive.Help{
			Message: "Then enter the following information:",
			Steps: []string{
				fmt.Sprintf("Name: %s", cluster.Name()),
				fmt.Sprintf("Redirect URI: %s/oauth2callback/%s", oauthURL, idpName),
				"Scopes: openid",
			},
		})
		if err != nil {
			return idpBuilder, err
		}

		clientID, err = interactive.GetString(interactive.Input{
			Question: "Application ID",
			Help:     "Paste the Application ID provided by GitLab when registering your application.",
			Default:  clientID,
			Required: true,
		})
		if err != nil {
			return idpBuilder, errors.New("Expected a GitLab application Application ID")
		}

		if clientSecret == "" {
			clientSecret, err = interactive.GetPassword(interactive.Input{
				Question: "Secret",
				Help:     "Paste the Secret provided by GitLab when registering your application.",
				Required: true,
			})
			if err != nil {
				return idpBuilder, errors.New("Expected a GitLab application Secret")
			}
		}
	}

	caPath := args.caPath
	if interactive.Enabled() && cmd.Flags().Changed("host-url") {
		caPath, err = interactive.GetCert(interactive.Input{
			Question: "CA file path",
			Help:     cmd.Flags().Lookup("ca").Usage,
			Default:  caPath,
			Validators: []interactive.Validator{
				func(val interface{}) error {
					_, err := ioutil.ReadFile(fmt.Sprintf("%v", val))
					return err
				},
			},
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

	// Create GitLab IDP
	gitlabIDP := cmv1.NewGitlabIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		URL(gitlabURL)

	// Set the CA file, if any
	if ca != "" {
		gitlabIDP = gitlabIDP.CA(ca)
	}

	// Create new IDP with GitLab provider
	idpBuilder.
		Type("GitlabIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(mappingMethod)).
		Gitlab(gitlabIDP)

	return
}

func validateGitlabHostURL(val interface{}) error {
	gitlabURL := fmt.Sprintf("%v", val)
	parsedIssuerURL, err := url.ParseRequestURI(gitlabURL)
	if err != nil {
		return fmt.Errorf("Expected a valid GitLab provider URL: %s", err)
	}
	if parsedIssuerURL.Scheme != "https" {
		return errors.New("Expected GitLab provider URL to use an https:// scheme")
	}
	if parsedIssuerURL.RawQuery != "" {
		return errors.New("GitLab provider URL must not have query parameters")
	}
	if parsedIssuerURL.Fragment != "" {
		return errors.New("GitLab provider URL must not have a fragment")
	}
	return nil
}
