/*
Copyright (c) 2022 Red Hat, Inc.

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

package service

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	serviceName string
	clusterName string
}

var Cmd = &cobra.Command{
	Use:   "service",
	Short: "Creates a managed service.",
	Long: `  Managed Services are Openshift clusters that provide a specific function.
  Use this command to create managed services.`,
	Example: `  # Create a Managed Service using service1.
  rosa create service --service=service1 --clusterName=clusterName`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	// Basic options
	flags.StringVar(
		&args.serviceName,
		"service",
		"",
		"Name of the service.",
	)

	flags.StringVar(
		&args.clusterName,
		"clusterName",
		"",
		"Name of the cluster.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	_, err = ocmClient.CreateService()
	if err != nil {
		reporter.Errorf("Failed to create managed service: %s", err)
		os.Exit(1)
	}
}
