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
	"errors"
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

var errNotLoggedIn = errors.New("user is not logged in to OCM")

func init() {
	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime()
	// Load config before initializing OCM so we can detect hyperfleet-only mode.
	// When only a Platform API URL is configured (no OCM credentials), skip the
	// OCM dependency and the region validation that WithAWS() requires.
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg != nil && cfg.HyperfleetURL != "" && config.IsNotValid(cfg) {
		r = r.WithAWSOnly()
	} else {
		r = r.WithAWS()
	}
	err := runWithRuntime(r)
	r.Cleanup()
	if err != nil {
		if errors.Is(err, errNotLoggedIn) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime) error {
	awsRegion, err := aws.GetRegion("")
	if err != nil {
		r.Reporter.Errorf("Error getting AWS region: %v", err)
		return fmt.Errorf("error getting AWS region: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		r.Reporter.Errorf("Failed to load config file: %v", err)
		return fmt.Errorf("loading config file: %w", err)
	}

	hfURL := ""
	if cfg != nil {
		hfURL = cfg.HyperfleetURL
	}

	ocmArmed := cfg != nil && !config.IsNotValid(cfg)
	if ocmArmed {
		var armed bool
		armed, err = cfg.Armed()
		if err != nil {
			r.Reporter.Errorf("Failed to verify configuration: %v", err)
			return fmt.Errorf("verifying configuration: %w", err)
		}
		ocmArmed = armed
	}

	if !ocmArmed && hfURL == "" {
		r.Reporter.Errorf("User is not logged in")
		return errNotLoggedIn
	}

	outputObject := object.Object{
		"AWS Account ID":     r.Creator.AccountID,
		"AWS Default Region": awsRegion,
		"AWS ARN":            r.Creator.ARN,
	}

	if hfURL != "" {
		outputObject["Platform API"] = hfURL
	}

	if ocmArmed {
		if r.OCMClient != nil {
			err = r.OCMClient.Close()
			if err != nil {
				r.Reporter.Errorf("Failed to close existing OCM connection: %v", err)
				return fmt.Errorf("closing existing OCM connection: %w", err)
			}
		}

		r.OCMClient, err = ocm.NewClient().
			Config(cfg).
			Logger(r.Logger).
			Build()
		if err != nil {
			if hfURL != "" {
				// Hyperfleet-only login: stale OCM credentials may be present
				// but are not required. Degrade gracefully without OCM data.
				r.Reporter.Warnf("Skipping OCM info (not logged in to OCM): %v", err)
			} else {
				r.Reporter.Errorf("Failed to create OCM connection: %v", err)
				return fmt.Errorf("creating OCM connection: %w", err)
			}
		} else {
			account, err := r.OCMClient.GetCurrentAccount()
			if err != nil {
				r.Reporter.Errorf("Failed to get current account: %s", err)
				return fmt.Errorf("getting current account: %w", err)
			}

			if account == nil {
				account, err = getAccountDataFromToken(cfg)
				if err != nil {
					r.Reporter.Errorf("Failed to get account data from token: %v", err)
					return fmt.Errorf("getting account data from token: %w", err)
				}
			}

			outputObject["OCM API"] = cfg.URL
			outputObject["OCM Account ID"] = account.ID()
			outputObject["OCM Account Name"] = fmt.Sprintf("%s %s", account.FirstName(), account.LastName())
			outputObject["OCM Account Username"] = account.Username()
			outputObject["OCM Account Email"] = account.Email()
			outputObject["OCM Organization ID"] = account.Organization().ID()
			outputObject["OCM Organization Name"] = account.Organization().Name()
			if account.Organization().ExternalID() != "" {
				outputObject["OCM Organization External ID"] = account.Organization().ExternalID()
			}
		}
	}

	if output.HasFlag() {
		err = output.Print(outputObject)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			return fmt.Errorf("printing whoami output: %w", err)
		}
		return nil
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
	return nil
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
