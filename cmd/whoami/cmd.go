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
	"net/http"
	"os"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/ocm/config"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Displays user account information",
	Long:  "Displays information about your AWS and Red Hat accounts",
	Example: `  # Displays user information
  moactl whoami`,
	Run: run,
}

func run(_ *cobra.Command, _ []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Failed to create logger: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("failed to create AWS client: %v", err)
		os.Exit(1)
	}

	// Get current AWS account information:
	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Get default AWS region:
	awsRegion, err := aws.GetRegion("")
	if err != nil {
		reporter.Errorf("Error getting AWS region: %v", err)
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		reporter.Errorf("Failed to load config file: %v", err)
		os.Exit(1)
	}
	if cfg == nil {
		reporter.Errorf("User is not logged in to OCM")
		os.Exit(0)
	}

	// Verify configuration file:
	loggedIn, err := cfg.Armed()
	if err != nil {
		reporter.Errorf("Failed to verify configuration: %v", err)
		os.Exit(1)
	}
	if !loggedIn {
		reporter.Errorf("User is not logged in to OCM")
		os.Exit(0)
	}

	// Create a connection to OCM:
	connection, err := ocm.NewConnection().
		Config(cfg).
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = connection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get current OCM account:
	useTokenData := false
	response, err := connection.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			useTokenData = true
		} else {
			reporter.Errorf("Failed to get current account: %v", err)
			os.Exit(1)
		}
	}

	var account *amsv1.Account
	if useTokenData {
		account, err = getAccountDataFromToken(cfg)
		if err != nil {
			reporter.Errorf("Failed to get account data from token: %v", err)
			os.Exit(1)
		}
	} else {
		account = response.Body()
	}
	fmt.Printf(""+
		"AWS Account ID:           %s\n"+
		"AWS Default Region:       %s\n"+
		"AWS ARN:                  %s\n"+
		"OpenShift API:            %s\n"+
		"Account ID:               %s\n"+
		"Account Name:             %s %s\n"+
		"Account Username:         %s\n"+
		"Account Email:            %s\n"+
		"Organization ID:          %s\n"+
		"Organization Name:        %s\n"+
		"Organization External ID: %s\n",
		awsCreator.AccountID,
		awsRegion,
		awsCreator.ARN,
		cfg.URL,
		account.ID(),
		account.FirstName(), account.LastName(),
		account.Username(),
		account.Email(),
		account.Organization().ID(),
		account.Organization().Name(),
		account.Organization().ExternalID(),
	)
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
