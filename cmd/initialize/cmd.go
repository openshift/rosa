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

package initialize

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/cmd/login"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/ocm/config"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	deleteStack bool
}

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Managed OpenShift on AWS clusters",
	Long: "Applies templates to support Managed OpenShift on AWS clusters. If you are not\n" +
		"yet logged in to OCM, it will prompt you for credentials.",
	Example: `  # Configure your AWS account to allow MOA clusters
  moactl init

  # Configure a new AWS account using pre-existing OCM credentials
  moactl init --token=$OFFLINE_ACCESS_TOKEN`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.BoolVar(
		&args.deleteStack,
		"delete-stack",
		false,
		"Deletes stack template applied to your AWS account during the 'init' command.\n",
	)

	// Force-load all flags from `login` into `init`
	flags.AddFlagSet(login.Cmd.Flags())
}

func run(cmd *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create reporter: %v\n", err)
		os.Exit(1)
	}

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	if cmd.Flags().NFlag() == 0 || (args.deleteStack && cmd.Flags().NFlag() == 1) {
		// Verify if user is already logged in:
		isLoggedIn := false
		cfg, err := config.Load()
		if err != nil {
			reporter.Errorf("Failed to load config file: %v", err)
			os.Exit(1)
		}
		if cfg != nil {
			// Check that credentials in the config file are valid
			isLoggedIn, err = cfg.Armed()
		}

		if isLoggedIn {
			username, err := cfg.UserName()
			if err != nil {
				reporter.Errorf("Failed to get username: %v", err)
				os.Exit(1)
			}

			reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
		} else {
			login.Cmd.Run(cmd, argv)
		}
	} else {
		// Always force login if user sets login flags
		login.Cmd.Run(cmd, argv)
	}

	// Validate AWS credentials for current user
	reporter.Infof("Validating AWS credentials...")
	ok, err := aws.ValidateCredentials()
	if err != nil {
		reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	reporter.Infof("AWS credentials are valid!")

	// Delete CloudFormation stack and exit
	if args.deleteStack {
		// Create the logger:
		logger, err := logging.NewLogger().Build()
		if err != nil {
			reporter.Errorf("Failed to create logger: %v", err)
			os.Exit(1)
		}

		ocmConnection, err := ocm.NewConnection().
			Logger(logger).
			Build()
		if err != nil {
			reporter.Errorf("Failed to create OCM connection: %v", err)
		}
		defer func() {
			err = ocmConnection.Close()
			if err != nil {
				reporter.Errorf("Failed to close OCM connection: %v", err)
			}
		}()

		if err = aws.DeleteStack(ocmConnection.ClustersMgmt().V1().Clusters()); err != nil {
			reporter.Errorf("Error while deleting stack: %v", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	if err = aws.CreateStack(); err != nil {
		reporter.Errorf("Error while creating stack: %v", err)
		os.Exit(1)
	}

	// Verify whether `oc` is installed
	reporter.Infof("Verifying whether OpenShift command-line tool is available...")
	ocDownloadURL := "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/"

	output, err := exec.Command("oc", "version", "--client").Output()
	if err != nil {
		reporter.Errorf("OpenShift command-line tool is not installed.\n"+
			"Go to %s to download the OpenShift client and add it to your PATH.", ocDownloadURL)
		os.Exit(1)
	}

	// Parse the version for the OpenShift Client
	version := strings.Replace(string(output), "\n", "", 1)
	isCorrectVersion, err := regexp.Match(`\W4.\d*`, output)
	if err != nil {
		reporter.Errorf("Failed to parse OpenShift Client version: %v", err)
		os.Exit(1)
	}

	if !isCorrectVersion {
		reporter.Warnf("Current OpenShift %s", version)
		reporter.Warnf("Your version of the OpenShift command-line tool is not supported.")
		fmt.Printf("Go to %s to download the latest version.\n", ocDownloadURL)
		os.Exit(1)
	}

	reporter.Infof("Current OpenShift %s", version)
}
