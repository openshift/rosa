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
	"fmt"
	"regexp"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/ocm/properties"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func HasClusters(client *cmv1.ClustersClient, creatorARN string) (bool, error) {
	query := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, creatorARN)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return false, fmt.Errorf("Failed to list clusters: %v", err)
	}

	return response.Total() > 0, nil
}

func GetCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string) (*cmv1.Cluster, error) {
	query := fmt.Sprintf(
		"(id = '%s' or name = '%s') and properties.%s = '%s'",
		clusterKey, clusterKey, properties.CreatorARN, creatorARN,
	)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to locate cluster '%s': %v", clusterKey, err)
	}

	switch response.Total() {
	case 0:
		return nil, fmt.Errorf("There is no cluster with identifier or name '%s'", clusterKey)
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, fmt.Errorf("There are %d clusters with identifier or name '%s'", response.Total(), clusterKey)
	}
}

func GetIdentityProviders(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.IdentityProvider, error) {
	idpClient := client.Cluster(clusterID).IdentityProviders()
	response, err := idpClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get identity providers for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetIngresses(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.Ingress, error) {
	ingressClient := client.Cluster(clusterID).Ingresses()
	response, err := ingressClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetUsers(client *cmv1.ClustersClient, clusterID string, group string) ([]*cmv1.User, error) {
	usersClient := client.Cluster(clusterID).Groups().Group(group).Users()
	response, err := usersClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get %s users for cluster '%s': %v", group, clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetAddOns(client *cmv1.AddOnsClient) ([]*cmv1.AddOn, error) {
	response, err := client.List().
		Search("enabled='t'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-ons: %v", err)
	}
	return response.Items().Slice(), nil
}

func GetAddOn(client *cmv1.AddOnsClient, id string) (*cmv1.AddOn, error) {
	addOn, err := client.Addon(id).Get().Send()
	if err != nil {
		return nil, err
	}
	return addOn.Body(), nil
}

type ClusterAddOn struct {
	ID        string
	Name      string
	State     string
	Available bool
}

// Get all add-ons available for a cluster
func GetClusterAddOns(connection *sdk.Connection, clusterID string) ([]*ClusterAddOn, error) {
	// Get organization ID (used to get add-on quotas)
	acctResponse, err := connection.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get current account: %s", err)
	}
	organization := acctResponse.Body().Organization().ID()

	// Get a list of add-on quotas for the current organization
	resourceQuotasResponse, err := connection.AccountsMgmt().V1().Organizations().
		Organization(organization).
		ResourceQuota().
		List().
		Search("resource_type='addon'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-ons: %v", err)
	}
	resourceQuotas := resourceQuotasResponse.Items()

	// Get complete list of enabled add-ons
	addOnsResponse, err := connection.ClustersMgmt().V1().Addons().
		List().
		Search("enabled='t'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-ons: %v", err)
	}
	addOns := addOnsResponse.Items()

	// Get add-ons already installed on cluster
	addOnInstallationsResponse, err := connection.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).
		Addons().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-on installations for cluster '%s': %v", clusterID, err)
	}
	addOnInstallations := addOnInstallationsResponse.Items()

	var clusterAddOns []*ClusterAddOn

	// Populate add-on installations with all add-on metadata
	addOns.Each(func(addOn *cmv1.AddOn) bool {
		clusterAddOn := ClusterAddOn{
			ID:        addOn.ID(),
			Name:      addOn.Name(),
			State:     "not installed",
			Available: addOn.ResourceCost() == 0,
		}

		// Only display add-ons for which the org has quota
		resourceQuotas.Each(func(resourceQuota *amsv1.ResourceQuota) bool {
			if addOn.ResourceName() == resourceQuota.ResourceName() {
				clusterAddOn.Available = float64(resourceQuota.Allowed()) >= addOn.ResourceCost()
			}
			return true
		})

		// Get the state of add-on installations on the cluster
		addOnInstallations.Each(func(addOnInstallation *cmv1.AddOnInstallation) bool {
			if addOn.ID() == addOnInstallation.Addon().ID() {
				clusterAddOn.State = string(addOnInstallation.State())
				if clusterAddOn.State == "" {
					clusterAddOn.State = string(cmv1.AddOnInstallationStateInstalling)
				}
			}
			return true
		})

		// Only display add-ons that meet the above criteria
		if clusterAddOn.Available {
			clusterAddOns = append(clusterAddOns, &clusterAddOn)
		}
		return true
	})

	return clusterAddOns, nil
}
