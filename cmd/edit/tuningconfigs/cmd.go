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

package tuningconfigs

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var args struct {
	specPath string
}

var Cmd = &cobra.Command{
	Use:     "tuning-configs",
	Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
	Short:   "Edit tuning config",
	Long:    "Edit a tuning config for a cluster.",
	Example: `  # Update the tuning config with name 'tuning-1' with the spec defined in file1
  rosa edit tuning-config --cluster=mycluster tuning-1 --spec-path file1`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the name of the tuning config",
			)
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.specPath,
		"spec-path",
		"",
		"Path of the file containing the new spec section of the tuning config to edit.",
	)

}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	tuningConfigName := argv[0]

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	input.CheckIfHypershiftClusterOrExit(r, cluster)

	// Try to find the tuning config:
	r.Reporter.Debugf("Loading tuning configs for cluster '%s'", clusterKey)
	tuningConfig, err := r.OCMClient.FindTuningConfigByName(cluster.ID(), tuningConfigName)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	specPath := args.specPath
	if interactive.Enabled() {
		specPath, err = interactive.GetString(interactive.Input{
			Question: "Path of the file containing the spec of the tuning config",
			Help:     cmd.Flags().Lookup("spec-path").Usage,
			Default:  specPath,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid spec path: %v", err)
			os.Exit(1)
		}
	}

	tuningConfigPatch, err := buildPatchFromInputFile(specPath, tuningConfig, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating tuning config '%s' on cluster '%s'", tuningConfig.Name(), clusterKey)
	_, err = r.OCMClient.UpdateTuningConfig(cluster.ID(), tuningConfigPatch)
	if err != nil {
		r.Reporter.Errorf("Failed to update tuning config for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated tuning config '%s' for cluster '%s'", tuningConfig.Name(), clusterKey)
}

func buildPatchFromInputFile(specPath string, tuningConfig *cmv1.TuningConfig,
	clusterKey string) (*cmv1.TuningConfig, error) {
	// Read the new spec
	specJson, err := input.UnmarshalInputFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("Expected a valid spec file: %v", err)
	}

	tuningConfigPatchBuilder := cmv1.NewTuningConfig().ID(tuningConfig.ID()).Spec(specJson)
	tuningConfigPatch, err := tuningConfigPatchBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create tuning config patch for cluster '%s': %v", clusterKey, err)
	}
	return tuningConfigPatch, nil
}
