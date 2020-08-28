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
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/moactl/cmd/completion"
	"github.com/openshift/moactl/cmd/create"
	"github.com/openshift/moactl/cmd/describe"
	"github.com/openshift/moactl/cmd/dlt"
	"github.com/openshift/moactl/cmd/docs"
	"github.com/openshift/moactl/cmd/edit"
	"github.com/openshift/moactl/cmd/initialize"
	"github.com/openshift/moactl/cmd/list"
	"github.com/openshift/moactl/cmd/login"
	"github.com/openshift/moactl/cmd/logout"
	"github.com/openshift/moactl/cmd/logs"
	"github.com/openshift/moactl/cmd/verify"
	"github.com/openshift/moactl/cmd/version"
	"github.com/openshift/moactl/cmd/whoami"

	"github.com/openshift/moactl/pkg/arguments"
)

var root = &cobra.Command{
	Use:  "moactl",
	Long: "Command line tool for AMRO (formerly known as MOA).",
}

func init() {
	// Register the options that are managed by the 'flag' package, so that they will also be parsed
	// by the 'pflag' package:
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Add the command line flags:
	fs := root.PersistentFlags()
	arguments.AddDebugFlag(fs)

	// Register the subcommands:
	root.AddCommand(completion.Cmd)
	root.AddCommand(create.Cmd)
	root.AddCommand(describe.Cmd)
	root.AddCommand(dlt.Cmd)
	root.AddCommand(docs.Cmd)
	root.AddCommand(edit.Cmd)
	root.AddCommand(list.Cmd)
	root.AddCommand(initialize.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(logs.Cmd)
	root.AddCommand(verify.Cmd)
	root.AddCommand(version.Cmd)
	root.AddCommand(whoami.Cmd)
}

func main() {
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err := root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute root command: %s\n", err)
		os.Exit(1)
	}
}
