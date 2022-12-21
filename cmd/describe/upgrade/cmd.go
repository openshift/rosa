/*
Copyright (c) 2022 Red Hat, Inc.

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

package upgrade

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"appliance", "upgrade"},
	Short:   "Show details of an upgrade",
	Long:    "Show details of an upgrade",
	Example: `  # Describe an upgrade-policy"
  rosa describe upgrade`,
	Run:    run,
	Hidden: false,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()
	cluster := r.FetchCluster()

	// Try to find the cluster:
	r.Reporter.Debugf("Loading upgrade with id '%s'", cluster.ID())
	upgrades, err := r.OCMClient.GetUpgradePolicies(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get upgrade with cluster id '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	_, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	if len(upgrades) < 1 {
		r.Reporter.Errorf("No available upgrades for cluster id '%s'", cluster.ID())
		os.Exit(1)
	}

	for _, upgrade := range upgrades {
		fmt.Printf(`
                %-28s%s
		%-28s%s
		%-28s%s
                %-28s%s
`,
			"ID:", upgrade.ID(),
			"Cluster ID:", upgrade.ClusterID(),
			"Next Run:", upgrade.NextRun(),
			"Upgrade State:", upgradeState.Value())
		if upgrade.Schedule() != "" {
			fmt.Printf(`                %-28s%s
`, "Schedule At:", upgrade.Schedule())
		}
		if upgrade.Version() != "" {
			fmt.Printf(`                %-28s%s
`, "Version:", upgrade.Version())
		}
	}
}
