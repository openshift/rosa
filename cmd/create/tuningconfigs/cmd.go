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
	name     string
	specPath string
}

var Cmd = &cobra.Command{
	Use:     "tuning-configs",
	Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
	Short:   "Add tuning config",
	Long:    "Add a tuning config to a cluster.",
	Example: `  # Add a tuning config with name "tuned1" and spec from a file "file1" to a cluster named "mycluster"
 rosa create tuning-config --name=tuned1 --spec-path=file1 --cluster=mycluster"`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.name,
		"name",
		"",
		"Name of the tuning config to add.",
	)
	flags.StringVar(
		&args.specPath,
		"spec-path",
		"",
		"Path of the file containing the spec section of the tuning config to add.",
	)

	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	input.CheckIfHypershiftClusterOrExit(r, cluster)

	var err error
	name := args.name
	if name == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	if interactive.Enabled() {
		name, err = interactive.GetString(interactive.Input{
			Question: "Name of the tuning config",
			Help:     cmd.Flags().Lookup("name").Usage,
			Default:  name,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid name: %s", err)
			os.Exit(1)
		}
	}

	specPath := args.specPath
	if specPath == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}
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

	tuningConfig, err := buildTuningConfigFromInputFile(specPath, name, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	_, err = r.OCMClient.CreateTuningConfig(cluster.ID(), tuningConfig)
	if err != nil {
		r.Reporter.Errorf("Failed to add tuning config to cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Tuning config '%s' has been created on cluster '%s'.", name, clusterKey)
	r.Reporter.Infof("To view all tuning configs, run 'rosa list tuning-configs -c %s'", clusterKey)
}

func buildTuningConfigFromInputFile(specPath string, name string, clusterKey string) (*cmv1.TuningConfig, error) {
	specJson, err := input.UnmarshalInputFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("Expected a valid TuneD spec file: %v", err)
	}

	tuningConfigBuilder := cmv1.NewTuningConfig().Name(name).Spec(specJson)

	tuningConfig, err := tuningConfigBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to add tuning config to cluster '%s': %v", clusterKey, err)
	}
	return tuningConfig, nil
}
