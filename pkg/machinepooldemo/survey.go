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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
)

// RunSurveyGoldenPath walks the HCP golden path using Survey prompts and fake fixtures.
func RunSurveyGoldenPath() (Result, error) {
	result := Result{}

	name, err := interactive.GetString(interactive.Input{
		Question: "Machine pool name",
		Required: true,
		Validators: []interactive.Validator{
			interactive.RegExp(machinepool.MachinePoolKeyRE.String()),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid name for the machine pool: %s", err)
	}
	result.Name = name

	imageType, err := interactive.GetOption(interactive.Input{
		Question: "Image Type",
		Default:  string(cmv1.ImageTypeDefault),
		Options:  ImageTypes(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid image type: %s", err)
	}
	result.ImageType = imageType

	version, err := interactive.GetOption(interactive.Input{
		Question: "OpenShift version",
		Options:  Versions(),
		Default:  DemoClusterVersion,
		Required: true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid OpenShift version: %s", err)
	}
	result.Version = version

	selectSubnet, err := promptGoldenPathSubnet()
	if err != nil {
		return Result{}, err
	}
	if selectSubnet {
		return Result{}, fmt.Errorf("unexpected subnet selection in golden path demo")
	}

	availabilityZone, err := interactive.GetOption(interactive.Input{
		Question: "AWS availability zone",
		Options:  AvailabilityZones(),
		Default:  AvailabilityZones()[0],
		Required: true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid AWS availability zone: %s", err)
	}
	result.AvailabilityZone = availabilityZone

	fmt.Printf("There are several subnets for availability zone '%s'\n", availabilityZone)
	subnetOption, err := interactive.GetOption(interactive.Input{
		Question: "Subnet ID",
		Options:  SubnetOptions(),
		Default:  SubnetOptions()[0],
		Required: true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid AWS subnet: %s", err)
	}
	result.Subnet = aws.ParseOption(subnetOption)

	autoscaling, err := promptGoldenPathAutoscaling()
	if err != nil {
		return Result{}, err
	}
	result.Autoscaling = autoscaling

	replicaValidation := &machinepool.ReplicaSizeValidation{
		ClusterVersion: DemoClusterVersion,
		MultiAz:        false,
		IsHostedCp:     true,
		Autoscaling:    true,
	}

	minReplicas, err := interactive.GetInt(interactive.Input{
		Question: "Min replicas",
		Default:  2,
		Required: true,
		Validators: []interactive.Validator{
			replicaValidation.MinReplicaValidator(),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid number of min replicas: %s", err)
	}
	result.MinReplicas = minReplicas
	replicaValidation.MinReplicas = minReplicas

	maxReplicas, err := interactive.GetInt(interactive.Input{
		Question: "Max replicas",
		Default:  4,
		Required: true,
		Validators: []interactive.Validator{
			replicaValidation.MaxReplicaValidator(),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid number of max replicas: %s", err)
	}
	result.MaxReplicas = maxReplicas

	labels, err := interactive.GetString(interactive.Input{
		Question: "Labels",
		Validators: []interactive.Validator{
			mpHelpers.LabelValidator,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid comma-separated list of attributes: %s", err)
	}
	result.Labels = labels

	taints, err := interactive.GetString(interactive.Input{
		Question: "Taints",
		Validators: []interactive.Validator{
			taintValidator,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid comma-separated list of attributes: %s", err)
	}
	result.Taints = taints

	securityGroupOptions := SecurityGroupOptions()
	selectedSGs, err := interactive.GetMultipleOptions(interactive.Input{
		Question: "Additional 'Machine Pool' Security Group IDs",
		Options:  securityGroupOptions,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected valid Security Group IDs: %s", err)
	}
	for i, sg := range selectedSGs {
		selectedSGs[i] = aws.ParseOption(sg)
	}
	result.SecurityGroupIDs = selectedSGs

	tags, err := interactive.GetString(interactive.Input{
		Question: "Tags",
		Validators: []interactive.Validator{
			aws.UserTagValidator,
			aws.UserTagDuplicateValidator,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid set of tags: %s", err)
	}
	result.Tags = tags

	instanceType, err := interactive.GetOption(interactive.Input{
		Question: "Instance type",
		Options:  InstanceTypes(),
		Default:  InstanceTypes()[0],
		Required: true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid instance type: %s", err)
	}
	result.InstanceType = instanceType

	autorepair, err := interactive.GetBool(interactive.Input{
		Question: "Autorepair",
		Default:  true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for autorepair: %s", err)
	}
	result.Autorepair = autorepair

	tuningConfigs, err := interactive.GetMultipleOptions(interactive.Input{
		Question: "Tuning configs",
		Options:  TuningConfigNames(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for tuning configs: %s", err)
	}
	result.TuningConfigs = tuningConfigs

	capacityReservationID, err := interactive.GetString(interactive.Input{
		Question: "Capacity Reservation ID",
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for Capacity Reservation ID: %s", err)
	}
	result.CapacityReservationID = capacityReservationID

	capacityOptions := CapacityPreferenceOptionsAll()
	if capacityReservationID != "" {
		capacityOptions = CapacityPreferenceOptionsWithID()
	}
	capacityPreference, err := interactive.GetOption(interactive.Input{
		Question: "Capacity Reservation Preference",
		Options:  capacityOptions,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for Capacity Reservation Preference: %s", err)
	}
	if capacityPreference != "" {
		if err = mpHelpers.ValidateCapacityReservationPreference(capacityPreference, capacityReservationID); err != nil {
			return Result{}, fmt.Errorf("expected a valid value for Capacity Reservation Preference: %s", err)
		}
	}
	result.CapacityReservationPref = capacityPreference

	kubeletConfigs, err := interactive.GetMultipleOptions(interactive.Input{
		Question: "Kubelet config",
		Options:  KubeletConfigNames(),
		Validators: []interactive.Validator{
			machinepool.ValidateKubeletConfig,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for kubelet config: %s", err)
	}
	if err = machinepool.ValidateKubeletConfig(kubeletConfigs); err != nil {
		return Result{}, err
	}
	result.KubeletConfigs = kubeletConfigs

	httpTokens, err := interactive.GetOption(interactive.Input{
		Question: "Configure the use of IMDSv2 for ec2 instances",
		Options:  HttpTokenOptions(),
		Default:  string(cmv1.Ec2MetadataHttpTokensOptional),
		Required: true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid http tokens value: %v", err)
	}
	if err = ocm.ValidateHttpTokensValue(httpTokens); err != nil {
		return Result{}, fmt.Errorf("expected a valid http tokens value: %v", err)
	}
	result.HTTPTokens = httpTokens

	rootDiskSize, err := interactive.GetString(interactive.Input{
		Question: "Root disk size (GiB or TiB)",
		Default:  DefaultDiskSize,
		Validators: []interactive.Validator{
			interactive.NodePoolRootDiskSizeValidator(),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid node pool root disk size value: %v", err)
	}
	result.RootDiskSize = rootDiskSize

	nodeDrainGracePeriod, err := interactive.GetString(interactive.Input{
		Question: "Node drain grace period",
		Validators: []interactive.Validator{
			mpHelpers.ValidateNodeDrainGracePeriod,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for Node drain grace period: %s", err)
	}
	result.NodeDrainGracePeriod = nodeDrainGracePeriod

	maxSurge, err := interactive.GetString(interactive.Input{
		Question: "Max surge",
		Default:  DefaultMaxSurge,
		Validators: []interactive.Validator{
			mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for max surge: %s", err)
	}
	result.MaxSurge = maxSurge

	maxUnavailable, err := interactive.GetString(interactive.Input{
		Question: "Max unavailable",
		Default:  DefaultMaxUnavail,
		Validators: []interactive.Validator{
			mpHelpers.ValidateUpgradeMaxSurgeUnavailable,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("expected a valid value for max unavailable: %s", err)
	}
	result.MaxUnavailable = maxUnavailable

	return result, nil
}

func promptGoldenPathSubnet() (bool, error) {
	for {
		selectSubnet, err := interactive.GetBool(interactive.Input{
			Question: "Select subnet for a hosted machine pool",
			Default:  false,
		})
		if err != nil {
			return false, fmt.Errorf("expected a valid value for subnet for a hosted machine pool: %s", err)
		}
		if !selectSubnet {
			return false, nil
		}
		fmt.Println(MsgGoldenPathSubnet)
	}
}

func promptGoldenPathAutoscaling() (bool, error) {
	for {
		autoscaling, err := interactive.GetBool(interactive.Input{
			Question: "Enable autoscaling",
			Default:  false,
		})
		if err != nil {
			return false, fmt.Errorf("expected a valid value for enable-autoscaling: %s", err)
		}
		if autoscaling {
			return true, nil
		}
		fmt.Println(MsgGoldenPathAutoscaling)
	}
}
