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
	"math"
	"strings"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
)

const AcceleratedComputing = "accelerated_computing"

func (c *Client) GetMachineTypesInRegion(cloudProviderData *cmv1.CloudProviderData) (MachineTypeList, error) {
	collection := c.ocm.ClustersMgmt().V1().AWSInquiries().MachineTypes()
	page := 1
	size := 100

	var machineTypes MachineTypeList
	for {
		response, err := collection.Search().
			Parameter("order", "category asc").
			Body(cloudProviderData).
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return MachineTypeList{}, err
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

	return machineTypes, nil
}

func (c *Client) GetMachineTypes() (machineTypes MachineTypeList, err error) {
	collection := c.ocm.ClustersMgmt().V1().MachineTypes()
	page := 1
	size := 100
	for {
		var response *cmv1.MachineTypesListResponse
		response, err := collection.List().
			Search("cloud_provider.id = 'aws'").
			Order("category asc").
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
	MachineType *cmv1.MachineType
	Available   bool // TODO what exactly does this mean? for what kind of cluster?
	nodeQuota   int  // may be MaxInt when Cost == 0.
	// TODO: compute clusterQuota?
}

func (mt MachineType) HasQuota(multiAZ bool) bool {
	// Assumption: most machine types have unilimited quota for ROSA.
	// We didn't even fetch quotas other than GPU.
	return mt.MachineType.Category() != AcceleratedComputing || mt.nodeQuota > getDefaultNodes(multiAZ)
}

// GetAvailableMachineTypesInRegion get the supported machine type in the region.
// The function triggers the 'api/clusters_mgmt/v1/aws_inquiries/machine_types'
// and passes a role ARN for STS clusters or access keys for non-STS clusters.
func (c *Client) GetAvailableMachineTypesInRegion(region string, availabilityZones []string, roleARN string,
	awsClient aws.Client) (MachineTypeList, error) {
	cloudProviderDataBuilder, err := c.createCloudProviderDataBuilder(roleARN, awsClient, "")
	if err != nil {
		return MachineTypeList{}, err
	}
	if len(availabilityZones) > 0 {
		cloudProviderDataBuilder = cloudProviderDataBuilder.AvailabilityZones(availabilityZones...)
	}
	cloudProviderData, err := cloudProviderDataBuilder.Region(cmv1.NewCloudRegion().ID(region)).Build()
	if err != nil {
		return MachineTypeList{}, err
	}

	machineTypes, err := c.GetMachineTypesInRegion(cloudProviderData)
	if err != nil {
		return MachineTypeList{}, err
	}

	quotaCosts, err := c.getQuotaCosts()
	if err != nil {
		return MachineTypeList{}, err
	}

	machineTypes.UpdateAvailableQuota(quotaCosts)
	return machineTypes, nil
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
		// Assumption: most machine types have unilimited quota for ROSA.
		// TODO: this only matches "compute.node" quotas; also want "cluster"?
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
			// Assumption: most machine types have unilimited quota for ROSA.
			// We didn't even fetch quotas other than GPU.
			machineType.nodeQuota = math.MaxInt
			machineType.Available = true
			continue
		}
		quotaCosts.Each(func(quotaCost *amsv1.QuotaCost) bool {
			// Match at most one RelatedResource; in unlikely case several match, take highest quota (lowest cost).
			bestQuota := 0
			for _, relatedResource := range quotaCost.RelatedResources() {
				// TODO: check ResourceType is "compute.node"
				if machineType.MachineType.GenericName() == relatedResource.ResourceName() && isCompatible(relatedResource) {
					if relatedResource.Cost() == 0 {
						// Special case "infinite" quota
						machineType.nodeQuota = math.MaxInt
						machineType.Available = true
						// break from quotaCosts.Each.
						// To not waste time (won't find anything better)
						// but also to avoid `+=` overflowing the MaxInt!
						return false
					}
					// Integer division rounding down: 7 available at cost 4/node allows 1 node.
					foundQuota := (quotaCost.Allowed() - quotaCost.Consumed()) / relatedResource.Cost()
					if bestQuota < foundQuota {
						bestQuota = foundQuota
					}
				}
			}
			machineType.nodeQuota += bestQuota
			return true // continue quotaCosts.Each
		})
		machineType.Available = machineType.nodeQuota > 1
	}
}

func (mtl *MachineTypeList) GetAvailableIDs(multiAZ bool) (machineTypeList []string) {
	list := mtl.Filter(func(mt *MachineType) bool {
		return mt.Available && mt.HasQuota(multiAZ)
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

	if !v.HasQuota(multiAZ) {
		err := fmt.Errorf("Insufficient quota for instance type: %s", machineType)
		return err
	}

	return nil
}
