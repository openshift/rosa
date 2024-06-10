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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use   = "version"
	short = "Prints the version of the tool"
	long  = "Prints the version number of the tool."
)

func NewRosaVersionCommand() *cobra.Command {
	o := NewRosaVersionUserOptions()
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		Run:   rosa.DefaultRunner(rosa.DefaultRuntime(), RosaVersionRunner(o)),
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().BoolVar(
		&o.clientOnly,
		"client",
		false,
		"Client version only (no remote version check)",
	)
	cmd.Flags().BoolVarP(
		&o.verbose,
		"verbose",
		"v",
		false,
		"Display verbose version information, including download locations",
	)
	return cmd
}

func RosaVersionRunner(userOptions RosaVersionUserOptions) rosa.CommandRunner {
	return func(_ context.Context, _ *rosa.Runtime, _ *cobra.Command, _ []string) error {
		options, err := NewRosaVersionOptions()
		if err != nil {
			return fmt.Errorf("there was a problem creating version options: %v", err)
		}
		options.BindAndValidate(userOptions)
		return options.Version()
	}
}
