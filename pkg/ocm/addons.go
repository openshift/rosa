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

package ocm

import (
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

type AddOnParam struct {
	Key string
	Val string
}

type AddOnResource struct {
	AddOn     *cmv1.AddOn
	AZType    string
	Available bool
}

type ClusterAddOn struct {
	ID    string
	Name  string
	State string
}

func (c *Client) InstallAddOn(clusterKey string, accountID string, addOnID string,
	params []AddOnParam) error {
	cluster, err := c.GetCluster(clusterKey, accountID)
	if err != nil {
		return err
	}

	addOnInstallationBuilder := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn().ID(addOnID))

	if len(params) > 0 {
		addOnParamList := make([]*cmv1.AddOnInstallationParameterBuilder, len(params))
		for i, param := range params {
			addOnParamList[i] = cmv1.NewAddOnInstallationParameter().ID(param.Key).Value(param.Val)
		}
		addOnInstallationBuilder = addOnInstallationBuilder.
			Parameters(cmv1.NewAddOnInstallationParameterList().Items(addOnParamList...))
	}

	addOnInstallation, err := addOnInstallationBuilder.Build()
	if err != nil {
		return err
	}

	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().
		Cluster(cluster.ID()).
		Addons().
		Add().
		Body(addOnInstallation).
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func (c *Client) UninstallAddOn(clusterKey string, accountID string, addOnID string) error {
	cluster, err := c.GetCluster(clusterKey, accountID)
	if err != nil {
		return err
	}

	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().
		Cluster(cluster.ID()).
		Addons().
		Addoninstallation(addOnID).
		Delete().
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func (c *Client) GetAddOnInstallation(clusterKey string, accountID string,
	addOnID string) (*cmv1.AddOnInstallation, error) {
	cluster, err := c.GetCluster(clusterKey, accountID)
	if err != nil {
		return nil, err
	}

	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().
		Cluster(cluster.ID()).
		Addons().
		Addoninstallation(addOnID).
		Get().
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}

func (c *Client) UpdateAddOnInstallation(clusterKey string, accountID string, addOnID string,
	params []AddOnParam) error {
	cluster, err := c.GetCluster(clusterKey, accountID)
	if err != nil {
		return err
	}

	addOnInstallationBuilder := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn().ID(addOnID))

	if len(params) > 0 {
		addOnParamList := make([]*cmv1.AddOnInstallationParameterBuilder, len(params))
		for i, param := range params {
			addOnParamList[i] = cmv1.NewAddOnInstallationParameter().ID(param.Key).Value(param.Val)
		}
		addOnInstallationBuilder = addOnInstallationBuilder.
			Parameters(cmv1.NewAddOnInstallationParameterList().Items(addOnParamList...))
	}

	addOnInstallation, err := addOnInstallationBuilder.Build()
	if err != nil {
		return err
	}

	response, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(cluster.ID()).
		Addons().Addoninstallation(addOnID).
		Update().Body(addOnInstallation).Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func (c *Client) GetAddOnParameters(clusterID string, addOnID string) (*cmv1.AddOnParameterList, error) {
	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).AddonInquiries().AddonInquiry(addOnID).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body().Parameters(), nil
}

// Get complete list of available add-ons for the current organization
func (c *Client) GetAvailableAddOns() ([]*AddOnResource, error) {
	// Get organization ID (used to get add-on quotas)
	acctResponse, err := c.ocm.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, handleErr(acctResponse.Error(), err)
	}
	organization := acctResponse.Body().Organization().ID()

	// Get a list of add-on quotas for the current organization
	quotaCostResponse, err := c.ocm.AccountsMgmt().V1().Organizations().
		Organization(organization).
		QuotaCost().
		List().
		Search("quota_id LIKE 'add-on%'").
		Parameter("fetchRelatedResources", true).
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(quotaCostResponse.Error(), err)
	}
	quotaCosts := quotaCostResponse.Items()

	// Get complete list of enabled add-ons
	addOnsResponse, err := c.ocm.ClustersMgmt().V1().Addons().
		List().
		Search("enabled='t'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(addOnsResponse.Error(), err)
	}

	var addOns []*AddOnResource

	// Populate enabled add-ons with if they are available for the current org
	addOnsResponse.Items().Each(func(addOn *cmv1.AddOn) bool {
		addOnResource := &AddOnResource{
			AddOn: addOn,
		}
		// Free add-ons are always available
		available := addOn.ResourceCost() == 0

		// Only return add-ons for which the org has quota
		quotaCosts.Each(func(quotaCost *amsv1.QuotaCost) bool {
			// Check all related resources to ensure we're checking the product of the correct addon
			for _, relatedResource := range quotaCost.RelatedResources() {
				// Only return compatible addons
				if addOn.ResourceName() == relatedResource.ResourceName() && isCompatible(relatedResource) {
					available = true

					// Addon is only available if quota allows it
					addOnResource.Available = quotaCost.Allowed()-quotaCost.Consumed() >= relatedResource.Cost()

					// Track AZ type so that we can compare against cluster
					addOnResource.AZType = relatedResource.AvailabilityZoneType()
					// Since add-on is considered available now, there's no need to check the other resources
					return false
				}
			}
			return true
		})

		// Only display add-ons that meet the above criteria
		if available {
			addOns = append(addOns, addOnResource)
		}

		return true
	})

	return addOns, nil
}

func (c *Client) GetAddOn(id string) (*cmv1.AddOn, error) {
	response, err := c.ocm.ClustersMgmt().V1().Addons().Addon(id).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

// Get all add-ons available for a cluster
func (c *Client) GetClusterAddOns(cluster *cmv1.Cluster) ([]*ClusterAddOn, error) {
	addOnResources, err := c.GetAvailableAddOns()
	if err != nil {
		return nil, err
	}

	// Get add-ons already installed on cluster
	addOnInstallationsResponse, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).
		Addons().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(addOnInstallationsResponse.Error(), err)
	}
	addOnInstallations := addOnInstallationsResponse.Items()

	var clusterAddOns []*ClusterAddOn

	// Populate add-on installations with all add-on metadata
	for _, addOnResource := range addOnResources {
		// Ensure add-on is compatible with the cluster's availability zones
		if !(addOnResource.AZType == ANY ||
			(cluster.MultiAZ() && addOnResource.AZType == "multi") ||
			(!cluster.MultiAZ() && addOnResource.AZType == "single")) {
			continue
		}
		clusterAddOn := ClusterAddOn{
			ID:    addOnResource.AddOn.ID(),
			Name:  addOnResource.AddOn.Name(),
			State: "not installed",
		}
		if !addOnResource.Available {
			clusterAddOn.State = "unavailable"
		}

		// Get the state of add-on installations on the cluster
		addOnInstallations.Each(func(addOnInstallation *cmv1.AddOnInstallation) bool {
			if addOnResource.AddOn.ID() == addOnInstallation.Addon().ID() {
				clusterAddOn.State = string(addOnInstallation.State())
				if clusterAddOn.State == "" {
					clusterAddOn.State = string(cmv1.AddOnInstallationStateInstalling)
				}
			}
			return true
		})

		clusterAddOns = append(clusterAddOns, &clusterAddOn)
	}

	return clusterAddOns, nil
}
