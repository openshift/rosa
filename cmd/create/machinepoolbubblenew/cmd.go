/*
Copyright (c) 2021 Red Hat, Inc.

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

package machinepoolbubblenew

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	bubbleteamp "github.com/openshift/rosa/pkg/interactive/bubbletea/machinepoolbubblenew"
	"github.com/openshift/rosa/pkg/machinepooldemo"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "machinepool-bubble-new",
	Aliases: []string{"machinepoolbubblenew"},
	Short:   "Walk the HCP machine pool golden path with a UX-first Bubble Tea wizard (demo dry run)",
	Long: "Interactive Bubble Tea wizard for the HCP create machinepool golden path. " +
		"Uses native bubbles components (list, textinput, help) with Charm styling. " +
		"Fake cluster and AWS fixtures with production validators. No OCM or AWS resources are created.",
	Example: `  # Walk the golden path with the UX-first Bubble Tea wizard
  rosa create machinepool-bubble-new -i`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	interactive.AddFlag(Cmd.Flags())
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime()
	defer r.Cleanup()

	if !interactive.Enabled() {
		interactive.Enable()
	}

	result, err := bubbleteamp.RunWizard()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	machinepooldemo.PrintSuccess(r.Reporter, result, "Bubble Tea (UX-first)")
}
