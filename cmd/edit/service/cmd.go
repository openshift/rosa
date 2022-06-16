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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args ocm.UpdateManagedServiceArgs

var Cmd = &cobra.Command{
	Use:     "managed-service",
	Aliases: []string{"appliance", "service"},
	Short:   "Edit parameters of service",
	Long:    "Edit the parameters of a Red Hat managed service",
	Example: `  # Edit the parameters of the Red Hat OpenShift logging operator add-on installation
  rosa edit managed-service --id=<service id> --parameter-key parameter-value`,
	Run:                run,
	Hidden:             true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		err := arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return err
		}

		if len(cmd.Flags().Args()) > 0 {
			return fmt.Errorf("Unrecognized command line parameter")
		}
		return nil
	},
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
	logger := logging.NewLogger()

	if args.ID == "" {
		reporter.Errorf("Service id not specified.")
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
	reporter.Debugf("Loading service %q", args.ID)
	service, err := ocmClient.GetManagedService(ocm.DescribeManagedServiceArgs{ID: args.ID})
	if err != nil {
		reporter.Errorf("Failed to get service %q: %v", args.ID, err)
		os.Exit(1)
	}

	parameters := service.Parameters()

	if len(parameters) == 0 {
		reporter.Errorf("Service %q has no parameters to edit", args.ID)
		os.Exit(1)
	}

	args.Parameters = map[string]string{}
	for _, param := range parameters {
		flag := cmd.Flags().Lookup(param.ID())
		if flag != nil {
			args.Parameters[param.ID()] = flag.Value.String()
		}
	}

	reporter.Debugf("Updating parameters for service %q", args.ID)
	err = ocmClient.UpdateManagedService(args)
	if err != nil {
		reporter.Errorf("Failed to update service %q: %v", args.ID, err)
		os.Exit(1)
	}
	reporter.Infof("Service %q is now updating. To check the status run 'rosa describe service --id %s'", args.ID, args.ID)
}
