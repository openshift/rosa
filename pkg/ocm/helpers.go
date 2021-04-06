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
	"net"
	"net/http"
	"regexp"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"

	"github.com/openshift/rosa/pkg/ocm/properties"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)
var badUsernameRE = regexp.MustCompile(`^(~|\.?\.|.*[:\/%].*)$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func IsValidUsername(username string) bool {
	return !badUsernameRE.MatchString(username)
}

func HasClusters(client *cmv1.ClustersClient, creatorARN string) (bool, error) {
	query := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, creatorARN)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return false, handleErr(response.Error(), err)
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
		return nil, handleErr(response.Error(), err)
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
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

func IdentityProviderType(idp *cmv1.IdentityProvider) string {
	switch idp.Type() {
	case "GithubIdentityProvider":
		return "GitHub"
	case "GitlabIdentityProvider":
		return "GitLab"
	case "GoogleIdentityProvider":
		return "Google"
	case "HTPasswdIdentityProvider":
		return "htpasswd"
	case "LDAPIdentityProvider":
		return "LDAP"
	case "OpenIDIdentityProvider":
		return "OpenID"
	}

	return ""
}

func GetIngresses(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.Ingress, error) {
	ingressClient := client.Cluster(clusterID).Ingresses()
	response, err := ingressClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

func GetUser(client *cmv1.ClustersClient, clusterID string, group string, username string) (*cmv1.User, error) {
	response, err := client.Cluster(clusterID).
		Groups().Group(group).
		Users().User(username).
		Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}

func GetUsers(client *cmv1.ClustersClient, clusterID string, group string) ([]*cmv1.User, error) {
	usersClient := client.Cluster(clusterID).Groups().Group(group).Users()
	response, err := usersClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

type AddOnResource struct {
	AddOn     *cmv1.AddOn
	AZType    string
	Available bool
}

// Get complete list of available add-ons for the current organization
func GetAvailableAddOns(connection *sdk.Connection) ([]*AddOnResource, error) {
	// Get organization ID (used to get add-on quotas)
	acctResponse, err := connection.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, handleErr(acctResponse.Error(), err)
	}
	organization := acctResponse.Body().Organization().ID()

	// Get a list of add-on quotas for the current organization
	quotaCostResponse, err := connection.AccountsMgmt().V1().Organizations().
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
	addOnsResponse, err := connection.ClustersMgmt().V1().Addons().
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

// Determine whether an add-on is compatible with ROSA clusters in general
func isCompatible(relatedResource *amsv1.RelatedResource) bool {
	product := strings.ToLower(relatedResource.Product())
	cloudProvider := strings.ToLower(relatedResource.CloudProvider())
	byoc := strings.ToLower(relatedResource.BYOC())

	// nolint:goconst
	return (product == "any" || product == "rosa" || product == "moa") &&
		(cloudProvider == "any" || cloudProvider == "aws") &&
		(byoc == "any" || byoc == "byoc")
}

func GetAddOn(client *cmv1.AddOnsClient, id string) (*cmv1.AddOn, error) {
	response, err := client.Addon(id).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

type ClusterAddOn struct {
	ID    string
	Name  string
	State string
}

// Get all add-ons available for a cluster
func GetClusterAddOns(connection *sdk.Connection, cluster *cmv1.Cluster) ([]*ClusterAddOn, error) {
	addOnResources, err := GetAvailableAddOns(connection)
	if err != nil {
		return nil, err
	}

	// Get add-ons already installed on cluster
	addOnInstallationsResponse, err := connection.ClustersMgmt().V1().Clusters().
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
		if !(addOnResource.AZType == "any" ||
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

func GetClusterState(client *cmv1.ClustersClient, clusterID string) (cmv1.ClusterState, error) {
	response, err := client.Cluster(clusterID).Status().Get().Send()
	if err != nil || response.Body() == nil {
		return cmv1.ClusterState(""), err
	}
	return response.Body().State(), nil
}

func GetMachinePools(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.MachinePool, error) {
	response, err := client.Cluster(clusterID).MachinePools().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}

func handleErr(res *ocmerrors.Error, err error) error {
	msg := res.Reason()
	if msg == "" {
		msg = err.Error()
	}
	return errors.New(msg)
}

func GetDefaultClusterFlavors(ocmClient *cmv1.Client, flavour string) (dMachinecidr *net.IPNet, dPodcidr *net.IPNet,
	dServicecidr *net.IPNet, dhostPrefix int) {
	flavourGetResponse, err := ocmClient.Flavours().Flavour(flavour).Get().Send()
	if err != nil {
		flavourGetResponse, _ = ocmClient.Flavours().Flavour("osd-4").Get().Send()
	}
	network, ok := flavourGetResponse.Body().GetNetwork()
	if !ok {
		return nil, nil, nil, 0
	}
	_, dMachinecidr, err = net.ParseCIDR(network.MachineCIDR())
	if err != nil {
		dMachinecidr = nil
	}
	_, dPodcidr, err = net.ParseCIDR(network.PodCIDR())
	if err != nil {
		dPodcidr = nil
	}
	_, dServicecidr, err = net.ParseCIDR(network.ServiceCIDR())
	if err != nil {
		dServicecidr = nil
	}
	dhostPrefix, _ = network.GetHostPrefix()
	return dMachinecidr, dPodcidr, dServicecidr, dhostPrefix
}
