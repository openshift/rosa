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

package service

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args ocm.DescribeManagedServiceArgs

var Cmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"appliance"},
	Short:   "Show details of a service",
	Long:    "Show details of a service",
	Example: `  # Describe a service with id aaabbbccc"
  rosa describe service --id=aaabbbccc`,
	Run:    run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.ID,
		"id",
		"",
		"The id of the service to describe",
	)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	if args.ID == "" {
		reporter.Errorf("id not specified.")
		cmd.Help()
		os.Exit(1)
	}

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

	// Try to find the cluster:
	reporter.Debugf("Loading service with id %q", args.ID)
	service, err := ocmClient.GetManagedService(args)
	if err != nil {
		reporter.Errorf("Failed to get service with id %q: %v", args.ID, err)
		os.Exit(1)
	}

	fmt.Printf(`%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
`,
		"Id:", service.ID(),
		"Href:", service.HREF(),
		"Service type:", service.Service(),
		"Service State:", service.ServiceState(),
		"Cluster Name:", service.Cluster().Name(),
		"Created At:", service.CreatedAt(),
		"Updated At:", service.UpdatedAt())

}
