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

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/cmd/login"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Managed OpenShift on AWS clusters",
	Long:  "Applies templates to support Managed OpenShift on AWS clusters",
	Run:   run,
}

func init() {
	// Force-load all flags from `login` into `init`
	Cmd.Flags().AddFlagSet(login.Cmd.Flags())
}

func run(cmd *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Can't create logger: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	// Call `login` as part of `init`. We do this before other validations
	// to get the prompt out of the way before performing longer checks.
	login.Cmd.Run(cmd, argv)

	// Validate AWS credentials for current user
	reporter.Infof("Validating AWS credentials...")
	ok, err := client.ValidateCredentials()
	if err != nil {
		reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	reporter.Infof("AWS credentials are valid!")

	// Validate SCP policies for current user's account
	reporter.Infof("Validating SCP policies...")
	ok, err = client.ValidateSCP()
	if err != nil {
		reporter.Errorf("Error validating SCP policies: %v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Infof("Failed to validate SCP policies. Will try to continue anyway...")
	}

	// Ensure that there is an AWS user to create all the resources needed by the cluster:
	reporter.Infof("Ensuring cluster administrator user '%s'...", aws.AdminUserName)
	created, err := client.EnsureOsdCcsAdminUser()
	if err != nil {
		reporter.Errorf("Failed to create user '%s': %v", aws.AdminUserName, err)
		os.Exit(1)
	}
	if created {
		reporter.Infof("Admin user '%s' created successfuly!", aws.AdminUserName)
	} else {
		reporter.Infof("Admin user '%s' already exists!", aws.AdminUserName)
	}
}
