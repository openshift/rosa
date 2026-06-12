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

package machinepooldemo

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/machinepooldemo"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "machinepool-demo",
	Aliases: []string{"machinepooldemo"},
	Short:   "Walk the HCP machine pool golden path with Survey (demo dry run)",
	Long: "Interactive Survey copy of the HCP create machinepool golden path. " +
		"Uses fake cluster and AWS fixtures with production validators. " +
		"No OCM or AWS resources are created.",
	Example: `  # Walk the golden path with Survey prompts
  rosa create machinepool-demo -i`,
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

	result, err := machinepooldemo.RunSurveyGoldenPath()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	machinepooldemo.PrintSuccess(r.Reporter, result, "Survey")
}
