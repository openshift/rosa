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
	"fmt"
	"strings"

	"github.com/openshift/rosa/pkg/reporter"
)

// PrintSuccess prints a fake create summary for the demo commands.
func PrintSuccess(r reporter.Logger, result Result, ui string) {
	if r.IsTerminal() {
		r.Infof("Creating machine pool '%s' on cluster '%s' (%s demo dry run)",
			result.Name, DemoClusterKey, ui)
	}
	r.Infof("Machine pool '%s' created successfully (demo dry run — no OCM or AWS calls were made)",
		result.Name)
	if r.IsTerminal() {
		printSummary(r, result)
	}
}

func printSummary(r reporter.Logger, result Result) {
	lines := []string{
		fmt.Sprintf("  Name:              %s", result.Name),
		fmt.Sprintf("  Image type:        %s", result.ImageType),
		fmt.Sprintf("  Version:           %s", result.Version),
		fmt.Sprintf("  Subnet:            %s", result.Subnet),
		fmt.Sprintf("  Availability zone: %s", result.AvailabilityZone),
		fmt.Sprintf("  Autoscaling:       %t (min %d, max %d)", result.Autoscaling, result.MinReplicas, result.MaxReplicas),
		fmt.Sprintf("  Labels:            %s", emptyDash(result.Labels)),
		fmt.Sprintf("  Taints:            %s", emptyDash(result.Taints)),
		fmt.Sprintf("  Security groups:   %s", emptyDash(strings.Join(result.SecurityGroupIDs, ", "))),
		fmt.Sprintf("  Tags:              %s", emptyDash(result.Tags)),
		fmt.Sprintf("  Instance type:     %s", result.InstanceType),
		fmt.Sprintf("  Autorepair:        %t", result.Autorepair),
		fmt.Sprintf("  Tuning configs:    %s", emptyDash(strings.Join(result.TuningConfigs, ", "))),
		fmt.Sprintf("  Capacity res. ID:  %s", emptyDash(result.CapacityReservationID)),
		fmt.Sprintf("  Capacity pref.:    %s", emptyDash(result.CapacityReservationPref)),
		fmt.Sprintf("  Kubelet config:    %s", emptyDash(strings.Join(result.KubeletConfigs, ", "))),
		fmt.Sprintf("  IMDSv2:            %s", result.HTTPTokens),
		fmt.Sprintf("  Root disk size:    %s", result.RootDiskSize),
		fmt.Sprintf("  Node drain:        %s", emptyDash(result.NodeDrainGracePeriod)),
		fmt.Sprintf("  Max surge:         %s", emptyDash(result.MaxSurge)),
		fmt.Sprintf("  Max unavailable:   %s", emptyDash(result.MaxUnavailable)),
	}
	r.Infof("Collected settings:\n%s", strings.Join(lines, "\n"))
}

func emptyDash(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
