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
)

const (
	Yes = "Yes"
	No  = "No"
)

// Methods shared between node pools and machine pools

func PrintStringSlice(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return strings.Join(in, ", ")
}

func PrintLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range labels {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}

func PrintTaints(taints []*cmv1.Taint) string {
	if len(taints) == 0 {
		return ""
	}
	output := []string{}
	for _, taint := range taints {
		output = append(output, fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect()))
	}

	return strings.Join(output, ", ")
}

// Methods dedicated for Machine pools

func PrintMachinePoolAutoscaling(autoscaling *cmv1.MachinePoolAutoscaling) string {
	if autoscaling != nil {
		return Yes
	}
	return No
}

func PrintMachinePoolReplicas(autoscaling *cmv1.MachinePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
			autoscaling.MinReplicas(),
			autoscaling.MaxReplicas())
	}
	return fmt.Sprintf("%d", replicas)
}

func PrintMachinePoolSpot(mp *cmv1.MachinePool) string {
	if mp.AWS() != nil {
		if spot := mp.AWS().SpotMarketOptions(); spot != nil {
			price := "on-demand"
			if maxPrice, ok := spot.GetMaxPrice(); ok {
				price = fmt.Sprintf("max $%g", maxPrice)
			}
			return fmt.Sprintf("Yes (%s)", price)
		}
	}
	return No
}

func PrintMachinePoolDiskSize(mp *cmv1.MachinePool) string {
	if rootVolume, ok := mp.GetRootVolume(); ok {
		if aws, ok := rootVolume.GetAWS(); ok {
			if size, ok := aws.GetSize(); ok {
				return helper.GigybyteStringer(size)
			}
		}
	}

	return "default"
}
