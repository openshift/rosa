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

package version

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	verify "github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	args struct {
		clientOnly bool
		verbose    bool
	}

	Cmd             = makeCmd()
	delegateCommand = verify.Cmd.Run // used in testing
)

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version of the tool",
		Long:  "Prints the version number of the tool.",
		Run:   run,
	}
}

func init() {
	initFlags(Cmd)
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(
		&args.clientOnly,
		"client",
		false,
		"Client version only (no remote version check)",
	)

	flags.BoolVarP(
		&args.verbose,
		"verbose",
		"v",
		false,
		"Display verbose version information, including download locations",
	)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	fmt.Fprintf(os.Stdout, "%s (Build: %s)\n", info.Version, info.Build)
	if args.verbose {
		fmt.Fprintf(os.Stdout, "Information and download locations:\n\t%s\n\t%s\n",
			verify.ConsoleLatestFolder,
			verify.DownloadLatestMirrorFolder)
	}
	if !args.clientOnly {
		delegateCommand(verify.Cmd, []string{})
	}

	return nil
}
