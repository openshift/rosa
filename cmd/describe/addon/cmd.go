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

	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"add-on"},
	Short:   "Show details of an add-on",
	Long:    "Show details of an add-on",
	Example: `  # Describe an add-on named "codeready-workspaces"
  rosa describe addon codeready-workspaces`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line argument containing the identifier of the add-on")
		}
		return nil
	},
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// Try to find the add-on:
	addOnID := argv[0]
	r.Reporter.Debugf("Loading add-on '%s'", addOnID)
	addOn, err := r.OCMClient.GetAddOn(addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on '%s': %s\n"+
			"Try running 'rosa list addons' to see all available add-ons.",
			addOnID, err)
		os.Exit(1)
	}

	printDescription(addOn)
	printCredentialRequests(addOn.CredentialsRequests())
	printParameters(addOn.Parameters())
}

func printDescription(addOn *asv1.Addon) {
	fmt.Printf("ADD-ON\n"+
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

func printCredentialRequests(requests []*asv1.CredentialRequest) {
	if len(requests) > 0 {
		fmt.Printf("CREDENTIALS REQUESTS\n")
		for _, cr := range requests {
			fmt.Printf(""+
				"- Service account:  %s\n"+
				"  Secret name:      %s\n"+
				"  Secret namespace: %s\n",
				cr.ServiceAccount(),
				cr.Name(),
				cr.Namespace(),
			)
			if len(cr.PolicyPermissions()) > 0 {
				fmt.Printf("  Policy permissions:\n")
				for _, p := range cr.PolicyPermissions() {
					fmt.Printf("  - %s\n", p)
				}
			}
		}
	}
	fmt.Println()
}

func printParameters(params *asv1.AddonParameterList) {
	if params.Len() > 0 {
		fmt.Printf("ADD-ON PARAMETERS\n")
		params.Each(func(param *asv1.AddonParameter) bool {
			if !param.Enabled() {
				return true
			}
			fmt.Printf(""+
				"- ID:             %s\n"+
				"  Name:           %s\n"+
				"  Description:    %s\n"+
				"  Type:           %s\n"+
				"  Required:       %s\n"+
				"  Editable:       %s\n",
				param.ID(),
				param.Name(),
				wrapText(param.Description()),
				param.ValueType(),
				printBool(param.Required()),
				printBool(param.Editable()),
			)
			if param.DefaultValue() != "" {
				fmt.Printf("  Default Value:  %s\n", param.DefaultValue())
			}
			if param.Validation() != "" {
				fmt.Printf("  Validation:     /%s/\n", param.Validation())
			}
			fmt.Println()
			return true
		})
	}
}

func printBool(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}

func wrapText(text string) string {
	return strings.TrimSpace(
		regexp.MustCompile(`(.{1,80})( +|$\n?)|(.{1,80})`).
			ReplaceAllString(text, "$1$3\n                  "),
	)
}
