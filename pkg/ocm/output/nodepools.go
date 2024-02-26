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

	"github.com/openshift/rosa/pkg/ocm"
)

func PrintNodePoolAutoscaling(autoscaling *cmv1.NodePoolAutoscaling) string {
	if autoscaling != nil {
		return Yes
	}
	return No
}

func PrintNodePoolReplicas(autoscaling *cmv1.NodePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
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

func PrintNodePoolAdditionalSecurityGroups(aws *cmv1.AWSNodePool) string {
	if aws == nil {
		return ""
	}

	return PrintStringSlice(aws.AdditionalSecurityGroupIds())
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
		return Yes
	}
	return No
}

func PrintNodePoolTuningConfigs(tuningConfigs []string) string {
	if len(tuningConfigs) == 0 {
		return ""
	}
	return strings.Join(tuningConfigs, ",")
}
