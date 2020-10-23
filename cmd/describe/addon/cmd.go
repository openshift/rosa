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
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "addon [ID|NAME]",
	Aliases: []string{"add-on"},
	Hidden:  true,
	Short:   "Show details of an add-on",
	Long:    "Show details of an add-on",
	Example: `  # Describe an add-on named "codeready-workspaces"
  rosa describe addon codeready-workspaces`,
	Run: run,
}

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check command line arguments:
	if len(argv) != 1 {
		reporter.Errorf(
			"Expected exactly one command line argument or flag containing the identifier of the add-on",
		)
		os.Exit(1)
	}
	addOnID := argv[0]

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of add-ons:
	addOnsCollection := ocmConnection.ClustersMgmt().V1().Addons()

	// Try to find the add-on:
	reporter.Debugf("Loading add-on '%s'", addOnID)
	addOn, err := ocm.GetAddOn(addOnsCollection, addOnID)
	if err != nil {
		reporter.Errorf("Failed to get add-on '%s': %s\n"+
			"Try running 'rosa list addons' to see all available add-ons.",
			addOnID, err)
		os.Exit(1)
	}

	// Print add-on description:
	fmt.Printf(""+
		"ID:               %s\n"+
		"Name:             %s\n"+
		"Description:      %s\n"+
		"Documentation:    %s\n"+
		"Operator:         %s\n"+
		"Target namespace: %s\n"+
		"Install mode:     %s\n",
		addOn.ID(),
		addOn.Name(),
		wrapText(addOn.Description()),
		addOn.DocsLink(),
		addOn.OperatorName(),
		addOn.TargetNamespace(),
		addOn.InstallMode(),
	)
	fmt.Println()
}

func wrapText(text string) string {
	return strings.TrimSpace(
		regexp.MustCompile(`(.{1,80})( +|$\n?)|(.{1,80})`).
			ReplaceAllString(text, "$1$3\n                  "),
	)
}
