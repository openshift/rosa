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
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	verify "github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clientOnly bool
	verbose    bool
}

type RosaVersionOptions struct {
	reporter   *reporter.Object
	verifyRosa verify.VerifyRosa
}

func NewRosaVersionOptions() (*RosaVersionOptions, error) {
	verifyRosa, err := verify.NewVerifyRosaOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to build rosa verify options : %v", err)
	}

	rpt, err := reporter.CreateReporter()
	if err != nil {
		return nil, fmt.Errorf("these was a problem creating the reporter: %v", err)
	}

	return &RosaVersionOptions{
		verifyRosa: verifyRosa,
		reporter:   rpt,
	}, nil
}

func RosaVersionVisitor() rosa.RuntimeVisitor {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {
		return nil
	}
}

func RosaVersionRun(o *RosaVersionOptions) rosa.CommandRun {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {
		if err := o.Version(); err != nil {
			r.Reporter.Errorf("Failed to check ROSA version: %v", err)
			os.Exit(1)
		}
		return nil
	}
}

func NewRosaVersionCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{}
	options, err := NewRosaVersionOptions()
	if err != nil {
		return cmd, err
	}

	cmd.Use = "version"
	cmd.Short = "Prints the version of the tool"
	cmd.Long = "Prints the version number of the tool."

	cmd.Run = rosa.DefaultRosaCommandRun(RosaVersionVisitor(), RosaVersionRun(options))

	cmd.Flags().SortFlags = false
	cmd.Flags().BoolVar(
		&args.clientOnly,
		"client",
		false,
		"Client version only (no remote version check)",
	)
	cmd.Flags().BoolVarP(
		&args.verbose,
		"verbose",
		"v",
		false,
		"Display verbose version information, including download locations",
	)
	return cmd, nil
}

func (o *RosaVersionOptions) Version() error {
	o.reporter.Infof("%s (Build: %s)", info.Version, info.Build)

	if args.verbose {
		o.reporter.Infof("Information and download locations:\n\t%s\n\t%s\n",
			verify.ConsoleLatestFolder,
			verify.DownloadLatestMirrorFolder)
	}

	if !args.clientOnly {
		if err := o.verifyRosa.Verify(); err != nil {
			return fmt.Errorf("failed to verify rosa : %v", err)
		}
	}

	return nil
}
