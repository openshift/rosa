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

package whoami

import (
	"fmt"
	"os"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Displays user account information",
	Long:  "Displays information about your AWS and Red Hat accounts",
	Example: `  # Displays user information
  rosa whoami`,
	Run: run,
}

func init() {
	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS()

	// Get default AWS region:
	awsRegion, err := aws.GetRegion("")
	if err != nil {
		r.Reporter.Errorf("Error getting AWS region: %v", err)
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		r.Reporter.Errorf("Failed to load config file: %v", err)
		os.Exit(1)
	}
	if cfg == nil {
		r.Reporter.Errorf("User is not logged in to OCM")
		os.Exit(0)
	}

	// Verify configuration file:
	loggedIn, err := cfg.Armed()
	if err != nil {
		r.Reporter.Errorf("Failed to verify configuration: %v", err)
		os.Exit(1)
	}
	if !loggedIn {
		r.Reporter.Errorf("User is not logged in to OCM")
		os.Exit(0)
	}

	// Create a connection to OCM:
	r.OCMClient, err = ocm.NewClient().
		Config(cfg).
		Logger(r.Logger).
		Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer r.Cleanup()

	// Get current OCM account:
	account, err := r.OCMClient.GetCurrentAccount()
	if err != nil {
		r.Reporter.Errorf("Failed to get current account: %s", err)
		os.Exit(1)
	}

	if account == nil {
		account, err = getAccountDataFromToken(cfg)
		if err != nil {
			r.Reporter.Errorf("Failed to get account data from token: %v", err)
			os.Exit(1)
		}
	}
	fmt.Printf(""+
		"AWS Account ID:               %s\n"+
		"AWS Default Region:           %s\n"+
		"AWS ARN:                      %s\n"+
		"OCM API:                      %s\n"+
		"OCM Account ID:               %s\n"+
		"OCM Account Name:             %s %s\n"+
		"OCM Account Username:         %s\n"+
		"OCM Account Email:            %s\n"+
		"OCM Organization ID:          %s\n"+
		"OCM Organization Name:        %s\n",
		r.Creator.AccountID,
		awsRegion,
		r.Creator.ARN,
		cfg.URL,
		account.ID(),
		account.FirstName(), account.LastName(),
		account.Username(),
		account.Email(),
		account.Organization().ID(),
		account.Organization().Name(),
	)
	if account.Organization().ExternalID() != "" {
		fmt.Printf(""+
			"OCM Organization External ID: %s\n",
			account.Organization().ExternalID(),
		)
	}
	fmt.Println()
}

func getAccountDataFromToken(cfg *config.Config) (*amsv1.Account, error) {
	firstName, err := cfg.GetData("first_name")
	if err != nil {
		return nil, err
	}
	lastName, err := cfg.GetData("last_name")
	if err != nil {
		return nil, err
	}
	username, err := cfg.GetData("username")
	if err != nil {
		return nil, err
	}
	email, err := cfg.GetData("email")
	if err != nil {
		return nil, err
	}
	orgID, err := cfg.GetData("org_id")
	if err != nil {
		return nil, err
	}
	return amsv1.NewAccount().
		FirstName(firstName).
		LastName(lastName).
		Username(username).
		Email(email).
		Organization(amsv1.NewOrganization().
			ExternalID(orgID),
		).
		Build()
}
