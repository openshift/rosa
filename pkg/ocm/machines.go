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

package ocm

import (
	"errors"
	"fmt"
	"strings"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const AcceleratedComputing = "accelerated_computing"

func (c *Client) GetMachineTypes() (machineTypes MachineTypeList, err error) {
	collection := c.ocm.ClustersMgmt().V1().MachineTypes()
	page := 1
	size := 100
	for {
		var response *cmv1.MachineTypesListResponse
		response, err := collection.List().
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

		response.Items().Each(func(item *cmv1.MachineType) bool {
			machineTypes = append(machineTypes, &MachineType{
				MachineType: item,
			})
			return true
		})

		if response.Size() < size {
			break
		}
		page++
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

func (c *Client) GetAvailableMachineTypes() (MachineTypeList, error) {
	machineTypes, err := c.GetMachineTypes()
	if err != nil {
		return nil, err
	}

	quotaCosts, err := c.getQuotaCosts()
	if err != nil {
		return nil, err
	}

	machineTypes.UpdateAvailableQuota(quotaCosts)
	return machineTypes, nil
}

func (c *Client) getQuotaCosts() (*amsv1.QuotaCostList, error) {
	acctResponse, err := c.ocm.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, handleErr(acctResponse.Error(), err)
	}
	organization := acctResponse.Body().Organization().ID()
	quotaCostResponse, err := c.ocm.AccountsMgmt().V1().Organizations().
		Organization(organization).
		QuotaCost().
		List().
		Parameter("fetchRelatedResources", true).
		Parameter("search", "quota_id~='gpu'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(quotaCostResponse.Error(), err)
	}
	quotaCosts := quotaCostResponse.Items()
	return quotaCosts, nil
}

// A list of MachineTypes with additional information
type MachineTypeList []*MachineType

// IDs extracts list of IDs from a MachineTypeList
func (mtl *MachineTypeList) IDs() []string {
	res := make([]string, len(*mtl))
	for i, v := range *mtl {
		res[i] = v.MachineType.ID()
	}
	return res
}

// Find returns the first MachineType matching the ID
func (mtl *MachineTypeList) Find(id string) *MachineType {
	for _, v := range *mtl {
		if v.MachineType.ID() == id {
			return v
		}
	}
	return nil
}

// Filter returns a new MachineTypeList with only elements for which fn returned true
func (mtl *MachineTypeList) Filter(fn func(*MachineType) bool) MachineTypeList {
	var res MachineTypeList
	for _, v := range *mtl {
		if fn(v) {
			res = append(res, v)
		}
	}
	return res
}

func (mtl *MachineTypeList) UpdateAvailableQuota(quotaCosts *amsv1.QuotaCostList) {
	for _, machineType := range *mtl {
		if machineType.MachineType.Category() != AcceleratedComputing {
			machineType.Available = true
			continue
		}
		quotaCosts.Each(func(quotaCost *amsv1.QuotaCost) bool {
			for _, relatedResource := range quotaCost.RelatedResources() {
				if machineType.MachineType.GenericName() == relatedResource.ResourceName() && isCompatible(relatedResource) {
					availableQuota := (quotaCost.Allowed() - quotaCost.Consumed()) / relatedResource.Cost()
					machineType.Available = availableQuota > 1
					machineType.AvailableQuota = availableQuota
					return false
				}
			}
			return true
		})
	}
}

func (mtl *MachineTypeList) GetAvailableIDs(multiAZ bool) (machineTypeList []string) {
	list := mtl.Filter(func(mt *MachineType) bool {
		return mt.Available &&
			(mt.MachineType.Category() != AcceleratedComputing || mt.AvailableQuota > getDefaultNodes(multiAZ))
	})
	return list.IDs()
}

// Validate AWS machine type is available with enough quota in the list
func (mtl *MachineTypeList) ValidateMachineType(machineType string, multiAZ bool) error {
	if machineType == "" {
		return nil
	}
	v := mtl.Find(machineType)

	if v == nil {
		allMachineTypes := strings.Join(mtl.IDs(), " ")
		err := fmt.Errorf("A valid machine type number must be specified\nValid machine types: %s", allMachineTypes)
		return err
	}

	if v.MachineType.Category() == AcceleratedComputing && v.AvailableQuota < getDefaultNodes(multiAZ) {
		err := fmt.Errorf("Insufficient quota for instance type: %s", machineType)
		return err
	}

	return nil
}
