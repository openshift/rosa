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
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
)

const (
	DemoClusterVersion = "4.16.0"
	DemoClusterKey     = "demo-hcp-cluster"
	DefaultDiskSize    = "100 GiB"
	DefaultMaxSurge    = "1"
	DefaultMaxUnavail  = "0"
)

// ImageTypes returns selectable image types for the demo.
func ImageTypes() []string {
	return append([]string{}, mpHelpers.ImageTypes...)
}

// Versions returns fake OpenShift versions for node pools.
func Versions() []string {
	return []string{"4.16.0", "4.16.8", "4.17.2"}
}

// AvailabilityZones returns fake AZs for subnet selection.
func AvailabilityZones() []string {
	return []string{"us-east-1a", "us-east-1b"}
}

// SubnetOptions returns formatted subnet options for us-east-1a (golden path: multiple subnets).
func SubnetOptions() []string {
	return []string{
		"subnet-0demoaaa111111111 ('private-subnet-a-1')",
		"subnet-0demobbb222222222 ('private-subnet-a-2')",
	}
}

// SecurityGroupOptions returns formatted security group options.
func SecurityGroupOptions() []string {
	return []string{
		"sg-0demo1111111111111 ('app-sg')",
		"sg-0demo2222222222222 ('extra-sg')",
		"sg-0demo3333333333333 ('monitoring-sg')",
	}
}

// InstanceTypes returns fake instance types (filtered by image type in real code; demo lists all).
func InstanceTypes() []string {
	return []string{"m7i.xlarge", "m7i.2xlarge", "r7i.xlarge"}
}

// TuningConfigNames returns fake tuning config names on the demo cluster.
func TuningConfigNames() []string {
	return []string{"tuning-default", "tuning-high-perf"}
}

// KubeletConfigNames returns fake kubelet config names on the demo cluster.
func KubeletConfigNames() []string {
	return []string{"kubelet-compact", "kubelet-standard"}
}

// CapacityPreferenceOptionsAll returns preference options when no reservation ID is set.
func CapacityPreferenceOptionsAll() []string {
	return []string{
		mpHelpers.CapacityReservationPreferenceNone,
		mpHelpers.CapacityReservationPreferenceOnly,
		mpHelpers.CapacityReservationPreferenceOpen,
	}
}

// CapacityPreferenceOptionsWithID returns preference options when a reservation ID is set.
func CapacityPreferenceOptionsWithID() []string {
	return []string{mpHelpers.CapacityReservationPreferenceOnly}
}

// HttpTokenOptions returns IMDSv2 settings.
func HttpTokenOptions() []string {
	return []string{
		string(cmv1.Ec2MetadataHttpTokensOptional),
		string(cmv1.Ec2MetadataHttpTokensRequired),
	}
}
