/*
Copyright (c) 2026 Red Hat, Inc.

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

package upgradeeligibility

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/upgrade/eligibility"
)

var args struct {
	targetVersion string
	output        string
}

var Cmd = &cobra.Command{
	Use:   "upgrade-eligibility",
	Short: "Check whether a cluster is eligible for an upgrade",
	Long: "Run all OCM-side eligibility checks for a cluster upgrade and report " +
		"any blockers with their owner (customer or service-team) and recommended action.\n\n" +
		"This command does not require backplane or in-cluster access. It checks:\n" +
		"  1. Cluster state (must be 'ready')\n" +
		"  2. Limited Support reasons\n" +
		"  3. Active upgrade policies\n" +
		"  4. Available upgrade targets\n" +
		"  5. Version gate acknowledgements\n\n" +
		"Use --output json for machine-readable output.",
	Example: `  # Check upgrade eligibility for a cluster
  rosa verify upgrade-eligibility -c mycluster

  # Check eligibility for a specific target version
  rosa verify upgrade-eligibility -c mycluster --target-version 4.18.12

  # Machine-readable output
  rosa verify upgrade-eligibility -c mycluster --output json`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()
	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.targetVersion,
		"target-version",
		"",
		"Specific target version to check eligibility for (optional)",
	)

	flags.StringVarP(
		&args.output,
		"output",
		"o",
		"",
		"Output format: '' (human-readable) or 'json'",
	)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	if args.output != "" && args.output != "json" {
		r.Reporter.Errorf("Invalid output format '%s'. Supported values: '' (human-readable), 'json'", args.output)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	r.Reporter.Debugf("Checking upgrade eligibility for cluster '%s'", clusterKey)

	result, err := eligibility.CheckAll(eligibility.NewOCMClient(r.OCMClient), cluster, args.targetVersion)
	if err != nil {
		r.Reporter.Errorf("Failed to check upgrade eligibility: %v", err)
		os.Exit(1)
	}

	if args.output == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			r.Reporter.Errorf("Failed to marshal result: %v", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		printHumanReadable(result)
	}

	if !result.Eligible {
		os.Exit(1)
	}
}

// printHumanReadable renders the eligibility result as formatted text.
func printHumanReadable(result *eligibility.Result) {
	fmt.Printf("Cluster: %s (%s, %s, %s)\n", result.Cluster.Name, result.Cluster.Product,
		result.Cluster.Version, result.Cluster.Region)

	if result.TargetVersion != "" {
		fmt.Printf("Target:  %s\n", result.TargetVersion)
	}
	fmt.Println()

	blockerCount := result.BlockerCount()

	if result.Eligible {
		fmt.Println("STATUS: ELIGIBLE")
		fmt.Println()
		if len(result.AvailableUpgrades) > 0 {
			fmt.Printf("Available targets: %s\n", strings.Join(result.AvailableUpgrades, ", "))
		}
	} else {
		fmt.Printf("STATUS: BLOCKED — %d issue(s)\n", blockerCount)
		fmt.Println()
	}

	for _, f := range result.Findings {
		switch f.Severity {
		case eligibility.SeverityBlocker:
			fmt.Printf("  [BLOCKER] %s (owner: %s)\n", f.Summary, f.Owner)
		case eligibility.SeverityWarning:
			fmt.Printf("  [WARNING] %s (owner: %s)\n", f.Summary, f.Owner)
		case eligibility.SeverityOK:
			fmt.Printf("  [OK] %s\n", f.Summary)
		}
		if f.Detail != "" {
			for _, line := range strings.Split(f.Detail, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
		if f.Action != "" {
			fmt.Printf("    Action: %s\n", f.Action)
		}
		fmt.Println()
	}

	if len(result.Gates) > 0 {
		fmt.Println("  Version gates:")
		for _, g := range result.Gates {
			status := "acknowledged"
			if !g.Acknowledged {
				status = "NOT acknowledged"
			}
			stsTag := ""
			if g.STSOnly {
				stsTag = " [STS-only]"
			}
			fmt.Printf("    - %s — %s%s\n", g.Description, status, stsTag)
		}
		fmt.Println()
	}

	if !result.Eligible {
		fmt.Println("Re-run after resolving blockers to verify eligibility.")
	}

	codes := result.ReasonCodes()
	if len(codes) > 0 {
		codeStrs := make([]string, len(codes))
		for i, c := range codes {
			codeStrs[i] = string(c)
		}
		fmt.Printf("Last evaluated: %s  [reason-codes: %s]\n",
			result.EvaluatedAt.Format("2006-01-02T15:04:05Z"), strings.Join(codeStrs, ", "))
	} else {
		fmt.Printf("Last evaluated: %s\n", result.EvaluatedAt.Format("2006-01-02T15:04:05Z"))
	}
}
