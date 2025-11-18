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
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
)

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

func PrintEC2MetadataHttpTokens(aws *cmv1.AWSNodePool) cmv1.Ec2MetadataHttpTokens {
	if aws == nil || aws.Ec2MetadataHttpTokens() == "" {
		return cmv1.Ec2MetadataHttpTokensOptional
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

func PrintNodePoolMessage(status *cmv1.NodePoolStatus) string {
	if status != nil {
		return status.Message()
	}
	return ""
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
	if capacityReservation != nil {
		id, ok := capacityReservation.GetId()
		if !ok {
			return ""
		}
		marketType, ok := capacityReservation.GetMarketType()
		if !ok {
			return ""
		}
		return fmt.Sprintf("\n"+
			" - ID:                                 %s\n"+
			" - Type:                               %s",
			id, marketType)
	}
	return ""
}
