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

	"gitlab.cee.redhat.com/service/moactl/cmd/completion"
	"gitlab.cee.redhat.com/service/moactl/cmd/create"
	"gitlab.cee.redhat.com/service/moactl/cmd/describe"
	"gitlab.cee.redhat.com/service/moactl/cmd/dlt"
	"gitlab.cee.redhat.com/service/moactl/cmd/initialize"
	"gitlab.cee.redhat.com/service/moactl/cmd/list"
	"gitlab.cee.redhat.com/service/moactl/cmd/login"
	"gitlab.cee.redhat.com/service/moactl/cmd/logout"
	"gitlab.cee.redhat.com/service/moactl/cmd/version"

	"gitlab.cee.redhat.com/service/moactl/pkg/arguments"
)

var root = &cobra.Command{
	Use:  "moactl",
	Long: "Command line tool for MOA.",
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
	root.AddCommand(list.Cmd)
	root.AddCommand(initialize.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(version.Cmd)
}

func main() {
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err := root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't execute root command: %s\n", err)
		os.Exit(1)
	}
}
