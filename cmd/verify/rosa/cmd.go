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

package rosa

import (
	context "context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/version"
)

var aliases = []string{"rosa"}

const (
	use     = "rosa-client"
	short   = "Verify ROSA client tools"
	long    = "Verify that the ROSA client tools is installed and compatible."
	example = `  # Verify rosa client tools
  rosa verify rosa`
)

func NewVerifyRosaCommand() *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Aliases: aliases,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.DefaultRuntime(), VerifyRosaRunner()),
	}
}

func VerifyRosaRunner() rosa.CommandRunner {
	return func(_ context.Context, _ *rosa.Runtime, _ *cobra.Command, _ []string) error {
		options, err := NewVerifyRosaOptions()
		if err != nil {
			return fmt.Errorf("failed to create rosa options: %v", err)
		}
		return options.Verify()
	}
}

//go:generate mockgen -source=cmd.go -package=rosa -destination=./cmd_mock.go
type VerifyRosa interface {
	Verify() error
}

var _ VerifyRosa = &VerifyRosaOptions{}

func NewVerifyRosaOptions() (VerifyRosa, error) {
	v, err := version.NewRosaVersion()
	if err != nil {
		return nil, err
	}

	rpt := reporter.CreateReporter()

	return &VerifyRosaOptions{
		rosaVersion: v,
		reporter:    rpt,
	}, nil
}

type VerifyRosaOptions struct {
	rosaVersion version.RosaVersion
	reporter    *reporter.Object
}

func (o *VerifyRosaOptions) Verify() error {
	latestVersion, isLatest, err := o.rosaVersion.IsLatest(info.DefaultVersion)
	if err != nil {
		return fmt.Errorf("there was a problem verifying if version is latest: %v", err)
	}

	if !isLatest {
		o.reporter.Infof(
			"There is a newer release version '%s', please consider updating: %s",
			latestVersion, version.ConsoleLatestFolder)
		return nil
	}

	if o.reporter.IsTerminal() {
		o.reporter.Infof("Your ROSA CLI is up to date.")
	}
	return nil
}
