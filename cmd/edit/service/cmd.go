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
	"github.com/spf13/pflag"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
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
	r := rosa.NewRuntime().WithOCM().WithFlagChecker()
	defer r.Cleanup()

	// Adding known flags to flag checker before parsing the unknown flags
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		r.FlagChecker.AddValidFlag(flag)
	})

	err := arguments.ParseUnknownFlags(cmd, argv)
	if err != nil {
		r.Reporter.Errorf("Failed to parse flags: %v", err)
		os.Exit(1)
	}

	if len(cmd.Flags().Args()) > 0 {
		r.Reporter.Errorf("Unrecognized command line parameter")
		os.Exit(1)
	}

	if args.ID == "" {
		r.Reporter.Errorf("Service id not specified.")
		cmd.Help()
		os.Exit(1)
	}

	// Try to find the service:
	r.Reporter.Debugf("Loading service %q", args.ID)
	service, err := r.OCMClient.GetManagedService(ocm.DescribeManagedServiceArgs{ID: args.ID})
	if err != nil {
		r.Reporter.Errorf("Failed to get service %q: %v", args.ID, err)
		os.Exit(1)
	}

	// Setting parameter flags as valid
	addOn, err := r.OCMClient.GetAddOn(service.Service())
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on %q: %s", service.Service(), err)
		os.Exit(1)
	}

	addonParameters := addOn.Parameters()
	addonParameters.Each(func(param *cmv1.AddOnParameter) bool {
		r.FlagChecker.AddValidParameter(param.ID())
		return true
	})

	// Now that rosa knows the expected fields to validate,
	// Validate that all of the user-specified flags are valid.
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !r.FlagChecker.IsValidFlag(flag) {
			r.Reporter.Errorf("%q is not a valid flag", flag.Name)
			os.Exit(1)
		}
	})

	args.Parameters = map[string]string{}
	addonParameters.Each(func(param *cmv1.AddOnParameter) bool {
		flag := cmd.Flags().Lookup(param.ID())
		if flag != nil {
			if !param.Editable() {
				r.Reporter.Errorf("Cannot edit the parameter %q", param.ID())
			}
			args.Parameters[param.ID()] = flag.Value.String()
		}
		return true
	})

	r.Reporter.Debugf("Updating parameters for service %q", args.ID)
	err = r.OCMClient.UpdateManagedService(args)
	if err != nil {
		r.Reporter.Errorf("Failed to update service %q: %v", args.ID, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Service %q is now updating. To check the status run 'rosa describe service --id %s'",
		args.ID, args.ID)
}
