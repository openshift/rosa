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

package dnsdomains

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "dns-domain ID",
	Aliases: []string{"dnsdomain"},
	Short:   "Delete DNS domain",
	Long:    "Delete a specific DNS domain.",
	Example: `  # Delete a DNS domain with ID github-1
  rosa delete dns-domain github-1`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the ID of the DNS domain",
			)
		}
		return nil
	},
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	id := argv[0]

	r.Reporter.Debugf("Deleting dns domain '%s''", id)
	err := r.OCMClient.DeleteDNSDomain(id)
	if err != nil {
		r.Reporter.Errorf("Failed to delete dns domain '%s': %s",
			id, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Successfully deleted dns domain '%s'", id)
}
