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

// Result captures the answers collected by the demo wizards.
type Result struct {
	Name                        string
	ImageType                   string
	Version                     string
	Subnet                      string
	AvailabilityZone            string
	Autoscaling                 bool
	MinReplicas                 int
	MaxReplicas                 int
	Labels                      string
	Taints                      string
	Tags                        string
	SecurityGroupIDs            []string
	InstanceType                string
	Autorepair                  bool
	TuningConfigs               []string
	CapacityReservationID       string
	CapacityReservationPref     string
	KubeletConfigs              []string
	HTTPTokens                  string
	RootDiskSize                string
	NodeDrainGracePeriod        string
	MaxSurge                    string
	MaxUnavailable              string
}
