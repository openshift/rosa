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

package addon

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/addon"
	"github.com/openshift/rosa/pkg/rosa"
)

func NewDescribeAddonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addon ID",
		Aliases: []string{"add-on"},
		Short:   "Show details of an add-on",
		Long:    "Show details of an add-on",
		Example: `  # Describe an add-on named "codeready-workspaces"
	rosa describe addon codeready-workspaces`,
		Run: rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeAddonRunner()),
		Args: func(_ *cobra.Command, argv []string) error {
			if len(argv) != 1 {
				return fmt.Errorf(
					"Expected exactly one command line argument containing the identifier of the add-on")
			}
			return nil
		},
	}

	return cmd
}

func DescribeAddonRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {

		// Try to find the add-on:
		addOnID := argv[0]
		runtime.Reporter.Debugf("Loading add-on '%s'", addOnID)
		addOn, err := runtime.OCMClient.GetAddOn(addOnID)
		if err != nil {
			return fmt.Errorf("Failed to get add-on '%s': %s\n"+
				"Try running 'rosa list addons' to see all available add-ons.",
				addOnID, err)
		}

		addon.PrintDescription(addOn)
		addon.PrintCredentialRequests(addOn.CredentialsRequests())
		addon.PrintParameters(addOn.Parameters())

		return nil
	}
}
