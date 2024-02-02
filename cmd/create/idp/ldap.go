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
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
)

func buildLdapIdp(cmd *cobra.Command,
	_ *cmv1.Cluster,
	idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	ldapURL := args.ldapURL
	ldapIDs := args.ldapIDs

	if ldapURL == "" || ldapIDs == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		instructionsURL := instructionsURLBase + "config-ldap-idp_config-identity-providers"
		err = interactive.PrintHelp(interactive.Help{
			Message: "To use LDAP as an identity provider, you must first register the application:",
			Steps: []string{
				fmt.Sprintf(`Open the following URL:
    %s`, instructionsURL),
				"Follow the instructions to register your application",
			},
		})
		if err != nil {
			return idpBuilder, err
		}
	}

	if interactive.Enabled() {
		ldapURL, err = interactive.GetString(interactive.Input{
			Question: "LDAP URL",
			Help:     cmd.Flags().Lookup("url").Usage,
			Default:  ldapURL,
			Required: true,
			Validators: []interactive.Validator{
				interactive.IsURL,
				validateLdapURL,
			},
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid LDAP URL: %s", err)
		}
	}
	err = validateLdapURL(ldapURL)
	if err != nil {
		return idpBuilder, err
	}

	needsSecure := strings.HasPrefix(ldapURL, "ldaps")

	ldapInsecure := args.ldapInsecure
	if interactive.Enabled() && !needsSecure {
		ldapInsecure, err = interactive.GetBool(interactive.Input{
			Question: "Insecure",
			Help:     cmd.Flags().Lookup("insecure").Usage,
			Default:  !needsSecure,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid insecure value: %s", err)
		}
	}
	if needsSecure && ldapInsecure {
		return idpBuilder, fmt.Errorf("Cannot use insecure connection on ldaps URLs")
	}

	caPath := args.caPath
	if interactive.Enabled() && !ldapInsecure {
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
		if ldapInsecure {
			return idpBuilder, fmt.Errorf("Cannot use certificate bundle with an insecure connection")
		}
		cert, err := os.ReadFile(caPath)
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid certificate bundle: %s", err)
		}
		ca = string(cert)
	}

	mappingMethod, err := getMappingMethod(cmd, args.mappingMethod)
	if err != nil {
		return idpBuilder, err
	}

	ldapBindDN := args.ldapBindDN
	ldapBindPassword := args.ldapBindPassword
	if interactive.Enabled() {
		ldapBindDN, err = interactive.GetString(interactive.Input{
			Question: "Bind DN",
			Help:     cmd.Flags().Lookup("bind-dn").Usage,
			Default:  ldapBindDN,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid DN to bind with: %s", err)
		}

		if ldapBindDN != "" {
			ldapBindPassword, err = interactive.GetPassword(interactive.Input{
				Question: "Bind password",
				Help:     cmd.Flags().Lookup("bind-password").Usage,
				Required: true,
			})
			if err != nil {
				return idpBuilder, fmt.Errorf("Expected a valid password to bind with: %s", err)
			}
		}
	}

	if interactive.Enabled() {
		err = interactive.PrintHelp(interactive.Help{
			Message: "The following options map LDAP attributes to identities. Enter multiple values separated by commas.",
		})
		if err != nil {
			return idpBuilder, err
		}
	}

	if interactive.Enabled() {
		ldapIDs, err = interactive.GetString(interactive.Input{
			Question: "ID",
			Help:     cmd.Flags().Lookup("id-attributes").Usage,
			Default:  ldapIDs,
			Required: true,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}
	}
	if ldapIDs == "" {
		return idpBuilder, fmt.Errorf("LDAP ID is required")
	}

	ldapUsernames := args.ldapUsernames
	ldapDisplayNames := args.ldapDisplayNames
	ldapEmails := args.ldapEmails
	if interactive.Enabled() {
		ldapUsernames, err = interactive.GetString(interactive.Input{
			Question: "Preferred username",
			Help:     cmd.Flags().Lookup("username-attributes").Usage,
			Default:  ldapUsernames,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}

		ldapDisplayNames, err = interactive.GetString(interactive.Input{
			Question: "Name",
			Help:     cmd.Flags().Lookup("name-attributes").Usage,
			Default:  ldapDisplayNames,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}

		ldapEmails, err = interactive.GetString(interactive.Input{
			Question: "Email",
			Help:     cmd.Flags().Lookup("email-attributes").Usage,
			Default:  ldapEmails,
		})
		if err != nil {
			return idpBuilder, fmt.Errorf("Expected a valid comma-separated list of attributes: %s", err)
		}
	}

	// Create LDAP attributes
	ldapAttributes := cmv1.NewLDAPAttributes().
		ID(strings.Split(ldapIDs, ",")...)

	if ldapUsernames != "" {
		ldapAttributes = ldapAttributes.PreferredUsername(strings.Split(ldapUsernames, ",")...)
	}
	if ldapDisplayNames != "" {
		ldapAttributes = ldapAttributes.Name(strings.Split(ldapDisplayNames, ",")...)
	}
	if ldapEmails != "" {
		ldapAttributes = ldapAttributes.Email(strings.Split(ldapEmails, ",")...)
	}

	// Create LDAP IDP
	ldapIDP := cmv1.NewLDAPIdentityProvider().
		URL(ldapURL).
		Insecure(ldapInsecure).
		Attributes(ldapAttributes)

	if ldapBindDN != "" {
		ldapIDP = ldapIDP.BindDN(ldapBindDN)
		if ldapBindPassword != "" {
			ldapIDP = ldapIDP.BindPassword(ldapBindPassword)
		}
	}

	// Set the CA file, if any
	if ca != "" {
		ldapIDP = ldapIDP.CA(ca)
	}

	// Create new IDP with LDAP provider
	idpBuilder.
		Type(cmv1.IdentityProviderTypeLDAP).
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(mappingMethod)).
		LDAP(ldapIDP)

	return
}

func validateLdapURL(val interface{}) error {
	ldapURL := fmt.Sprintf("%v", val)
	parsedLdapURL, err := url.ParseRequestURI(ldapURL)
	if err != nil {
		return fmt.Errorf("Expected a valid LDAP URL: %v", err)
	}
	if parsedLdapURL.Scheme != "ldap" && parsedLdapURL.Scheme != "ldaps" {
		return errors.New("Expected LDAP URL to have an ldap:// or ldaps:// scheme")
	}
	return nil
}
