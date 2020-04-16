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

func buildLdapIdp(cluster *cmv1.Cluster, idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	ldapURL := args.ldapURL
	ldapIDs := args.ldapIDs

	isInteractive := ldapURL == "" || ldapIDs == ""

	if isInteractive {
		fmt.Println("To use LDAP as an identity provider, you must first register the application:")
		instructionsURL := "https://docs.openshift.com/dedicated/4/authentication/identity_providers/configuring-ldap-identity-provider.html"
		fmt.Println("* Open the following URL:", instructionsURL)
		fmt.Println("* Follow the instructions to register your application")

		if ldapURL == "" {
			ldapURL, err = interactive.GetInput("Enter the URL which specifies the LDAP search parameters to use")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid LDAP URL")
			}
		}

		if ldapIDs == "" {
			ldapIDs, err = interactive.GetInput("Enter the list of attributes whose values should be used as the user ID")
			if err != nil {
				return idpBuilder, errors.New("Expected a valid comma-separated list of attributes")
			}
		}
	}

	parsedLdapURL, err := url.ParseRequestURI(ldapURL)
	if err != nil {
		return idpBuilder, fmt.Errorf("Expected a valid LDAP URL: %v", err)
	}
	if parsedLdapURL.Scheme != "ldap" && parsedLdapURL.Scheme != "ldaps" {
		return idpBuilder, errors.New("Expected LDAP URL to have an ldap:// or ldaps:// scheme.")
	}

	// Create LDAP attributes
	ldapAttributes := cmv1.NewLDAPAttributes().
		ID(strings.Split(ldapIDs, ",")...)

	if args.ldapUsernames != "" {
		ldapAttributes = ldapAttributes.PreferredUsername(strings.Split(args.ldapUsernames, ",")...)
	}
	if args.ldapDisplayNames != "" {
		ldapAttributes = ldapAttributes.Name(strings.Split(args.ldapDisplayNames, ",")...)
	}
	if args.ldapEmails != "" {
		ldapAttributes = ldapAttributes.Email(strings.Split(args.ldapEmails, ",")...)
	}

	// Create LDAP IDP
	ldapIDP := cmv1.NewLDAPIdentityProvider().
		URL(ldapURL).
		Attributes(ldapAttributes)

	if args.ldapBindDN != "" {
		ldapIDP = ldapIDP.BindDN(args.ldapBindDN)
		if args.ldapBindPassword != "" {
			ldapIDP = ldapIDP.BindPassword(args.ldapBindPassword)
		}
	}

	// Create new IDP with LDAP provider
	idpBuilder.
		Type("LDAPIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		LDAP(ldapIDP)

	return
}
