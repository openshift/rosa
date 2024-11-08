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
	"sort"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/object"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Displays user account information",
	Long:  "Displays information about your AWS and Red Hat accounts",
	Example: `  # Displays user information
  rosa whoami`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	output.AddFlag(Cmd)
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
	if cfg == nil || config.IsNotValid(cfg) {
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
	outputObject := object.Object{
		"AWS Account ID":        r.Creator.AccountID,
		"AWS Default Region":    awsRegion,
		"AWS ARN":               r.Creator.ARN,
		"OCM API":               cfg.URL,
		"OCM Account ID":        account.ID(),
		"OCM Account Name":      fmt.Sprintf("%s %s", account.FirstName(), account.LastName()),
		"OCM Account Username":  account.Username(),
		"OCM Account Email":     account.Email(),
		"OCM Organization ID":   account.Organization().ID(),
		"OCM Organization Name": account.Organization().Name(),
	}
	if account.Organization().ExternalID() != "" {
		outputObject["OCM Organization External ID"] = account.Organization().ExternalID()
	}

	if output.HasFlag() {
		err = output.Print(outputObject)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}
	keys := make([]string, 0, len(outputObject))
	for key := range outputObject {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("%-30s%v\n", key+":", outputObject[key])
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
