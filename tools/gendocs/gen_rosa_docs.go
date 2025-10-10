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

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/commands"
	"github.com/openshift/rosa/tools/gendocs/gendocs"
)

func main() {
	// Initialize the root command with all subcommands
	root := &cobra.Command{
		Use:   "rosa",
		Short: "Command line tool for ROSA.",
		Long: "Command line tool for Red Hat OpenShift Service on AWS.\n" +
			"For further documentation visit " +
			"https://access.redhat.com/documentation/en-us/red_hat_openshift_service_on_aws\n",
	}

	// Register all subcommands using shared function from pkg/commands
	commands.RegisterCommands(root)

	// Generate the documentation
	outputFile := "docs/generated/rosa-by-example-content.adoc"
	if err := gendocs.GenDocs(root, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Documentation generated successfully: %s\n", outputFile)
}
