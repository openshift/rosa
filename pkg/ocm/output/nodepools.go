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

package output

import (
	"fmt"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
)

// nodePoolMessageBulletRegexp matches a leading "* " style bullet in a node
// pool status message, capturing any leading whitespace so callers can tell
// a top-level bullet (e.g. "* Machine foo:") from a nested one
// (e.g. "    * NodeHealthy: ...").
var nodePoolMessageBulletRegexp = regexp.MustCompile(`^(\s*)\*\s*`)

func PrintNodePoolAutoscaling(autoscaling *cmv1.NodePoolAutoscaling) string {
	if autoscaling != nil {
		return output.Yes
	}
	return output.No
}

func PrintNodePoolReplicasInline(autoscaling *cmv1.NodePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
			autoscaling.MinReplica(),
			autoscaling.MaxReplica())
	}
	return fmt.Sprintf("%d", replicas)
}

func PrintNodePoolReplicas(autoscaling *cmv1.NodePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf(`
 - Min replicas: %d
 - Max replicas: %d`,
			autoscaling.MinReplica(),
			autoscaling.MaxReplica())
	}
	return fmt.Sprintf("%d", replicas)
}

func PrintNodePoolInstanceType(aws *cmv1.AWSNodePool) string {
	if aws == nil {
		return ""
	}
	return aws.InstanceType()
}

func PrintNodePoolImageType(imgType cmv1.ImageType) string {
	return string(imgType)
}

func PrintNodePoolAdditionalSecurityGroups(aws *cmv1.AWSNodePool) string {
	if aws == nil {
		return ""
	}

	return output.PrintStringSlice(aws.AdditionalSecurityGroupIds())
}

// PrintNodePoolSpot returns a human-readable Spot instance status for a node pool.
func PrintNodePoolSpot(aws *cmv1.AWSNodePool) string {
	if aws != nil {
		if spot := aws.SpotMarketOptions(); spot != nil {
			price := "on-demand"
			if maxPrice, ok := spot.GetMaxPrice(); ok && maxPrice != "" {
				price = fmt.Sprintf("max $%s", maxPrice)
			}
			return fmt.Sprintf("Yes (%s)", price)
		}
	}
	return output.No
}

func PrintEC2MetadataHttpTokens(aws *cmv1.AWSNodePool) cmv1.Ec2MetadataHttpTokens {
	if aws == nil {
		return ""
	}

	return aws.Ec2MetadataHttpTokens()
}

func PrintNodePoolCurrentReplicas(status *cmv1.NodePoolStatus) string {
	if status != nil {
		return fmt.Sprintf("%d", status.CurrentReplicas())
	}
	return ""
}

func PrintNodePoolReplicasShort(currentReplicas, desiredReplicas string) string {
	return fmt.Sprintf("%s/%s", currentReplicas, desiredReplicas)
}

// PrintNodePoolMessage renders a node pool's free-form status message, if
// any, as a YAML-style list, preserving whatever bullet nesting the service
// actually sends: each bullet more indented than the one before it becomes
// one level deeper (e.g. "  - Machine foo:" with "    - NodeHealthy: ..."
// nested beneath it), using 2 spaces per level. Nesting depth is relative,
// not tied to an exact column width, so it isn't capped at any fixed depth.
// A line that isn't itself a bullet (e.g. a wrapped continuation of the
// previous condition's text) is appended to the preceding list item rather
// than becoming a new one, and blank/whitespace-only lines are skipped
// entirely.
func PrintNodePoolMessage(status *cmv1.NodePoolStatus) string {
	if status == nil {
		return ""
	}

	message := status.Message()
	if strings.TrimSpace(message) == "" {
		return ""
	}

	var items []string
	var indentStack []int
	for _, line := range strings.Split(message, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		loc := nodePoolMessageBulletRegexp.FindStringSubmatchIndex(line)
		if loc == nil {
			if len(items) == 0 {
				items = append(items, "  - "+strings.TrimSpace(line))
			} else {
				items[len(items)-1] += " " + strings.TrimSpace(line)
			}
			continue
		}

		bulletIndent := loc[3] - loc[2]
		for len(indentStack) > 0 && indentStack[len(indentStack)-1] > bulletIndent {
			indentStack = indentStack[:len(indentStack)-1]
		}
		if len(indentStack) == 0 || indentStack[len(indentStack)-1] < bulletIndent {
			indentStack = append(indentStack, bulletIndent)
		}

		indent := strings.Repeat("  ", len(indentStack)) + "- "
		items = append(items, indent+line[loc[1]:])
	}
	if len(items) == 0 {
		return ""
	}

	return "\n" + strings.Join(items, "\n")
}

func PrintNodePoolVersion(version *cmv1.Version) string {
	return ocm.GetRawVersionId(version.ID())
}

func PrintNodePoolAutorepair(autorepair bool) string {
	if autorepair {
		return output.Yes
	}
	return output.No
}

func PrintNodePoolConfigs(configs []string) string {
	if len(configs) == 0 {
		return ""
	}
	return strings.Join(configs, ",")
}

func PrintNodeDrainGracePeriod(period *cmv1.Value) string {
	if period != nil && period.Value() != 0 {
		unit := "minute"
		if period.Value() > 1 {
			unit += "s"
		}
		return fmt.Sprintf("%d %s", int(period.Value()), unit)
	}

	return ""
}

func PrintNodePoolManagementUpgrade(upgrade *cmv1.NodePoolManagementUpgrade) string {
	if upgrade != nil {
		return fmt.Sprintf("\n"+
			" - Type:                               %s\n"+
			" - Max surge:                          %s\n"+
			" - Max unavailable:                    %s", upgrade.Type(), upgrade.MaxSurge(), upgrade.MaxUnavailable())
	}

	return ""
}

func PrintNodePoolDiskSize(aws *cmv1.AWSNodePool) string {
	diskSizeStr := "default"
	if aws != nil && aws.RootVolume() != nil {
		diskSize, ok := aws.RootVolume().GetSize()
		if ok {
			diskSizeStr = helper.GigybyteStringer(diskSize)
		}
	}

	return diskSizeStr
}

func PrintCapacityReservationDetails(capacityReservation *cmv1.AWSCapacityReservation) string {
	if capacityReservation == nil {
		return ""
	}

	var details []string

	if id, ok := capacityReservation.GetId(); ok {
		details = append(details, fmt.Sprintf(" - ID:                                 %s", id))
	}
	if marketType, ok := capacityReservation.GetMarketType(); ok {
		details = append(details, fmt.Sprintf(" - Type:                               %s", marketType))
	}
	if preference, ok := capacityReservation.GetPreference(); ok && preference != "" {
		details = append(details, fmt.Sprintf(" - Preference:                         %s", preference))
	}

	if len(details) == 0 {
		return ""
	}
	return "\n" + strings.Join(details, "\n")
}
