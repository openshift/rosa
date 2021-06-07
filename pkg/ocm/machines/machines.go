/*
Copyright (c) 2020 Red Hat, Inc.

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

package machines

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/ocm"
)

const AcceleratedComputing = "accelerated_computing"

func GetMachineTypes(client *cmv1.Client) (machineTypes []*cmv1.MachineType, err error) {
	collection := client.MachineTypes()
	page := 1
	size := 100
	for {
		var response *cmv1.MachineTypesListResponse
		response, err = collection.List().
			Search("cloud_provider.id = 'aws'").
			Order("cpu asc").
			Page(page).
			Size(size).
			Send()
		if err != nil {
			errMsg := response.Error().Reason()
			if errMsg == "" {
				errMsg = err.Error()
			}
			return nil, errors.New(errMsg)
		}
		machineTypes = append(machineTypes, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}

// Validate AWS machine types
func ValidateMachineType(machineType string, machineTypes []*MachineType, multiAZ bool) (string, error) {
	if machineType != "" {
		var machineTypeList []string
		// Check and set the cluster machineType
		hasMachineType := false
		for _, v := range machineTypes {
			machineTypeList = append(machineTypeList, v.MachineType.ID())
			if v.MachineType.ID() == machineType {
				if v.MachineType.Category() == AcceleratedComputing && v.AvailableQuota < getDefaultNodes(multiAZ) {
					err := fmt.Errorf("Insufficient quota for instance type: %s", machineType)
					return machineType, err
				}
				hasMachineType = true
			}
		}
		if !hasMachineType {
			allMachineTypes := strings.Join(machineTypeList, " ")
			err := fmt.Errorf("A valid machine type number must be specified\nValid machine types: %s", allMachineTypes)
			return machineType, err
		}
	}

	return machineType, nil
}

func GetMachineTypeList(client *cmv1.Client) (machineTypeList []string, err error) {
	machineTypes, err := GetMachineTypes(client)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve machine types: %s", err)
		return
	}

	for _, v := range machineTypes {
		machineTypeList = append(machineTypeList, v.ID())
	}

	return
}

func GetAvailableMachineTypeList(machineTypes []*MachineType, multiAZ bool) (machineTypeList []string) {
	for _, v := range machineTypes {
		if !v.Available {
			continue
		}
		if v.MachineType.Category() != AcceleratedComputing || v.AvailableQuota > getDefaultNodes(multiAZ) {
			machineTypeList = append(machineTypeList, v.MachineType.ID())
		}
	}
	return
}

func getDefaultNodes(multiAZ bool) int {
	minimumNodes := 2
	if multiAZ {
		minimumNodes = 3
	}
	return minimumNodes
}

type MachineType struct {
	MachineType    *cmv1.MachineType
	Available      bool
	AvailableQuota int
}

func GetAvailableMachineTypes(ocmConnection *sdk.Connection) ([]*MachineType, error) {
	ocmClient := ocmConnection.ClustersMgmt().V1()

	machineTypes, err := GetMachineTypes(ocmClient)
	if err != nil {
		return nil, err
	}
	acctResponse, err := ocmConnection.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, ocm.HandleErr(acctResponse.Error(), err)
	}
	organization := acctResponse.Body().Organization().ID()
	quotaCostResponse, err := ocmConnection.AccountsMgmt().V1().Organizations().
		Organization(organization).
		QuotaCost().
		List().
		Parameter("fetchRelatedResources", true).
		Parameter("search", "quota_id~='gpu'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, ocm.HandleErr(quotaCostResponse.Error(), err)
	}
	var availableMachineTypes []*MachineType
	quotaCosts := quotaCostResponse.Items()

	for _, machineType := range machineTypes {
		availableMachineType := &MachineType{
			MachineType: machineType,
		}
		if machineType.Category() == AcceleratedComputing {
			quotaCosts.Each(func(quotaCost *amsv1.QuotaCost) bool {
				for _, relatedResource := range quotaCost.RelatedResources() {
					if machineType.GenericName() == relatedResource.ResourceName() && ocm.IsCompatible(relatedResource) {
						availableQuota := (quotaCost.Allowed() - quotaCost.Consumed()) / relatedResource.Cost()
						availableMachineType.Available = availableQuota > 1
						availableMachineType.AvailableQuota = availableQuota
						return false
					}
				}
				return true
			})
		} else {
			availableMachineType.Available = true
		}
		availableMachineTypes = append(availableMachineTypes, availableMachineType)
	}
	return availableMachineTypes, nil
}
