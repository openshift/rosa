/*
Copyright (c) 2023 Red Hat, Inc.

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

package oidcconfig

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "oidc-config",
	Aliases: []string{"oidcconfig", "oidcconfig"},
	Short:   "List OIDC Configuration resources",
	Long:    "List OIDC Configuration resources",
	Example: `  # List all OIDC Configurations tied to your organization ID"
  rosa list oidc-config`,
	Run: run,
}

func init() {
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// Load any existing ingresses for this cluster
	r.Reporter.Debugf("Loading oidc configs for current org id")
	oidcConfigs, err := r.OCMClient.ListOidcConfigs()
	if err != nil {
		r.Reporter.Errorf("Failed to list OIDC Configurations: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(oidcConfigs)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(oidcConfigs) == 0 {
		r.Reporter.Infof("There are no OIDC Configurations for your organization")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tMANAGED\tISSUER URL\tSECRET ARN\n")
	for _, oidcConfig := range oidcConfigs {
		fmt.Fprintf(writer, "%s\t%v\t%s\t%s\n",
			oidcConfig.ID(),
			oidcConfig.Managed(),
			oidcConfig.IssuerUrl(),
			oidcConfig.SecretArn(),
		)
	}
	writer.Flush()
}
