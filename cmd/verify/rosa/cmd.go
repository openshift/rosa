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
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/version"
)

const (
	DownloadLatestMirrorFolder = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/latest/"
	baseReleasesFolder         = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/"
	ConsoleLatestFolder        = "https://console.redhat.com/openshift/downloads#tool-rosa"
)

//go:generate mockgen -source=cmd.go -package=rosa -destination=./cmd_mock.go
type VerifyRosa interface {
	Verify() error
}

var _ VerifyRosa = &VerifyRosaOptions{}

func VerifyRosaCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{}
	options, err := NewVerifyRosaOptions()
	if err != nil {
		return cmd, fmt.Errorf("failed to create rosa options: %v", err)
	}
	cmd.Use = "rosa-client"
	cmd.Aliases = []string{"rosa"}
	cmd.Short = "Verify ROSA client tools"
	cmd.Long = "Verify that the ROSA client tools is installed and compatible."
	cmd.Example = `  # Verify rosa client tools
  rosa verify rosa`

	cmd.Run = rosa.DefaultRosaCommandRun(VerifyRosaVisitor(), VerifyRosaRun(options))

	return cmd, nil
}

func VerifyRosaVisitor() rosa.RuntimeVisitor {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {
		return nil
	}
}

func VerifyRosaRun(o VerifyRosa) rosa.CommandRun {
	return func(ctx context.Context, r *rosa.Runtime, command *cobra.Command, args []string) error {
		if err := o.Verify(); err != nil {
			r.Reporter.Errorf("Failed to verify rosa : %v", err)
			os.Exit(1)
		}
		return nil
	}
}

func NewVerifyRosaOptions() (VerifyRosa, error) {
	v, err := version.NewRosaVersion()
	if err != nil {
		return nil, fmt.Errorf("there was a problem creating version: %v", err)
	}

	rpt, err := reporter.CreateReporter()
	if err != nil {
		return nil, fmt.Errorf("these was a problem creating the reporter: %v", err)
	}

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
	latestVersion, isLatest, err := o.rosaVersion.IsLatest(info.Version)
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
