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

package upgrades

import (
	"errors"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"
)

func GetUpgradePolicies(client *cmv1.Client, clusterID string) (upgradePolicies []*cmv1.UpgradePolicy, err error) {
	collection := client.Clusters().Cluster(clusterID).UpgradePolicies()
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

func GetScheduledUpgrade(client *cmv1.Client, clusterID string) (*cmv1.UpgradePolicy, error) {
	upgradePolicies, err := GetUpgradePolicies(client, clusterID)
	if err != nil {
		return nil, err
	}

	for _, upgradePolicy := range upgradePolicies {
		if upgradePolicy.ScheduleType() == "manual" && upgradePolicy.UpgradeType() == "OSD" {
			return upgradePolicy, nil
		}
	}

	return nil, nil
}

func CancelUpgrade(client *cmv1.Client, clusterID string) (bool, error) {
	scheduledUpgrade, err := GetScheduledUpgrade(client, clusterID)
	if err != nil || scheduledUpgrade == nil {
		return false, err
	}

	response, err := client.Clusters().
		Cluster(clusterID).
		UpgradePolicies().
		UpgradePolicy(scheduledUpgrade.ID()).
		Delete().
		Send()
	if err != nil {
		return false, handleErr(response.Error(), err)
	}

	return true, nil
}

func handleErr(res *ocmerrors.Error, err error) error {
	msg := res.Reason()
	if msg == "" {
		msg = err.Error()
	}
	return errors.New(msg)
}
