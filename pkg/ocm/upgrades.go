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
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func (c *Client) GetUpgradePolicies(clusterID string) (upgradePolicies []*cmv1.UpgradePolicy, err error) {
	collection := c.ocm.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		UpgradePolicies()
	page := 1
	size := 100
	for {
		response, err := collection.List().
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return nil, handleErr(response.Error(), err)
		}
		upgradePolicies = append(upgradePolicies, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}

func (c *Client) GetScheduledUpgrade(clusterID string) (*cmv1.UpgradePolicy, *cmv1.UpgradePolicyState, error) {
	upgradePolicies, err := c.GetUpgradePolicies(clusterID)
	if err != nil {
		return nil, nil, err
	}

	for _, upgradePolicy := range upgradePolicies {
		if upgradePolicy.ScheduleType() == "manual" && upgradePolicy.UpgradeType() == "OSD" {
			state, err := c.ocm.ClustersMgmt().V1().
				Clusters().Cluster(clusterID).
				UpgradePolicies().UpgradePolicy(upgradePolicy.ID()).
				State().
				Get().
				Send()
			if err != nil {
				return nil, nil, err
			}

			return upgradePolicy, state.Body(), nil
		}
	}

	return nil, nil, nil
}

func (c *Client) ScheduleUpgrade(clusterID string, upgradePolicy *cmv1.UpgradePolicy) error {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		UpgradePolicies().
		Add().Body(upgradePolicy).
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}

func (c *Client) CancelUpgrade(clusterID string) (bool, error) {
	scheduledUpgrade, _, err := c.GetScheduledUpgrade(clusterID)
	if err != nil || scheduledUpgrade == nil {
		return false, err
	}
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		UpgradePolicies().UpgradePolicy(scheduledUpgrade.ID()).
		Delete().
		Send()
	if err != nil {
		return false, handleErr(response.Error(), err)
	}
	return true, nil
}
